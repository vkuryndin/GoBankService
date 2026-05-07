package router

import (
	"net/http"
	"strings"

	"bank-service/internal/audit"
	"bank-service/internal/config"
	"bank-service/internal/handlers"
	"bank-service/internal/middleware"
	"bank-service/internal/repositories"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func NewRouter(
	healthHandler *handlers.HealthHandler,
	authHandler *handlers.AuthHandler,
	accountHandler *handlers.AccountHandler,
	transferHandler *handlers.TransferHandler,
	cardHandler *handlers.CardHandler,
	rateHandler *handlers.RateHandler,
	creditHandler *handlers.CreditHandler,
	notificationHandler *handlers.NotificationHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	mfaHandler *handlers.MFAHandler,
	adminHandler *handlers.AdminHandler,
	tokenRepository *repositories.TokenRepository,
	userRepository *repositories.UserRepository,
	idempotencyRepository *repositories.IdempotencyRepository,
	jwtSecret string,
	securityConfig config.SecurityConfig,
	auditRecorder audit.Recorder,
	logger *logrus.Logger,
) *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.RequestID())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.MaxRequestBodySize(securityConfig.MaxRequestBodyBytes))
	r.Use(buildPublicRateLimiter(securityConfig.RateLimit, auditRecorder))

	r.HandleFunc("/health", healthHandler.Health).Methods(http.MethodGet)
	r.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	protected := r.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware(jwtSecret, tokenRepository))
	protected.Use(buildProtectedRateLimiter(securityConfig.RateLimit, auditRecorder))
	protected.Use(middleware.IdempotencyMiddleware(
		idempotencyRepository,
		middleware.IdempotencyConfig{
			Enabled:  securityConfig.Idempotency.Enabled,
			Required: securityConfig.Idempotency.Required,
		},
		logger,
		auditRecorder,
	))

	protected.HandleFunc("/auth/check", authHandler.CheckAuth).Methods(http.MethodGet)
	protected.HandleFunc("/logout", authHandler.Logout).Methods(http.MethodPost)

	protected.HandleFunc("/mfa/request", mfaHandler.RequestCode).Methods(http.MethodPost)

	protected.HandleFunc("/accounts", accountHandler.CreateAccount).Methods(http.MethodPost)
	protected.HandleFunc("/accounts", accountHandler.GetUserAccounts).Methods(http.MethodGet)
	protected.HandleFunc("/accounts/{accountId}", accountHandler.GetAccount).Methods(http.MethodGet)
	protected.HandleFunc("/accounts/{accountId}/deposit", accountHandler.Deposit).Methods(http.MethodPost)
	protected.HandleFunc("/accounts/{accountId}/withdraw", accountHandler.Withdraw).Methods(http.MethodPost)
	protected.HandleFunc("/accounts/{accountId}/predict", analyticsHandler.PredictBalance).Methods(http.MethodGet)

	protected.HandleFunc("/transfer", transferHandler.Transfer).Methods(http.MethodPost)

	protected.HandleFunc("/cards", cardHandler.CreateCard).Methods(http.MethodPost)
	protected.HandleFunc("/cards", cardHandler.GetUserCards).Methods(http.MethodGet)
	protected.HandleFunc("/cards/{cardId}", cardHandler.GetCard).Methods(http.MethodGet)
	protected.HandleFunc("/cards/{cardId}/pay", cardHandler.PayByCard).Methods(http.MethodPost)

	protected.HandleFunc("/rates/key", rateHandler.GetKeyRate).Methods(http.MethodGet)

	protected.HandleFunc("/credits", creditHandler.CreateCredit).Methods(http.MethodPost)
	protected.HandleFunc("/credits", creditHandler.GetUserCredits).Methods(http.MethodGet)
	protected.HandleFunc("/credits/{creditId}", creditHandler.GetCredit).Methods(http.MethodGet)
	protected.HandleFunc("/credits/{creditId}/schedule", creditHandler.GetCreditSchedule).Methods(http.MethodGet)

	protected.HandleFunc("/notifications/test", notificationHandler.SendTestEmail).Methods(http.MethodGet)

	protected.HandleFunc("/analytics", analyticsHandler.GetAnalytics).Methods(http.MethodGet)

	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.AdminMiddleware(userRepository))

	admin.HandleFunc("/users", adminHandler.GetUsers).Methods(http.MethodGet)
	admin.HandleFunc("/logged-in-users", adminHandler.GetLoggedInUsers).Methods(http.MethodGet)
	admin.HandleFunc("/accounts/{accountId}/block", adminHandler.BlockAccount).Methods(http.MethodPost)
	admin.HandleFunc("/accounts/{accountId}/unblock", adminHandler.UnblockAccount).Methods(http.MethodPost)

	return r
}

func buildPublicRateLimiter(config config.RateLimitConfig, auditRecorder audit.Recorder) func(http.Handler) http.Handler {
	rules := []middleware.RateLimitRule{
		{
			Name:   "login",
			Limit:  config.LoginRequests,
			Window: config.LoginWindow,
			Match:  middleware.MatchMethodPath(http.MethodPost, "/login"),
			Key:    middleware.ClientIPKey,
		},
		{
			Name:   "register",
			Limit:  config.RegisterRequests,
			Window: config.RegisterWindow,
			Match:  middleware.MatchMethodPath(http.MethodPost, "/register"),
			Key:    middleware.ClientIPKey,
		},
		{
			Name:   "global",
			Limit:  config.GlobalRequests,
			Window: config.GlobalWindow,
			Match:  middleware.MatchAny(),
			Key:    middleware.ClientIPKey,
		},
	}

	return middleware.NewRateLimiter(config.Enabled, rules, config.CleanupInterval, auditRecorder)
}

func buildProtectedRateLimiter(config config.RateLimitConfig, auditRecorder audit.Recorder) func(http.Handler) http.Handler {
	rules := []middleware.RateLimitRule{
		{
			Name:   "mfa",
			Limit:  config.MFARequests,
			Window: config.MFAWindow,
			Match:  middleware.MatchMethodPath(http.MethodPost, "/mfa/request"),
			Key:    middleware.UserIDKey,
		},
		{
			Name:   "rates",
			Limit:  config.RateRequests,
			Window: config.RateWindow,
			Match:  middleware.MatchMethodPath(http.MethodGet, "/rates/key"),
			Key:    middleware.UserIDKey,
		},
		{
			Name:   "admin",
			Limit:  config.AdminRequests,
			Window: config.AdminWindow,
			Match:  middleware.MatchPathPrefix("/admin"),
			Key:    middleware.UserIDKey,
		},
		{
			Name:   "financial",
			Limit:  config.FinancialRequests,
			Window: config.FinancialWindow,
			Match:  isFinancialEndpoint,
			Key:    middleware.UserIDKey,
		},
	}

	return middleware.NewRateLimiter(config.Enabled, rules, config.CleanupInterval, auditRecorder)
}

func isFinancialEndpoint(r *http.Request) bool {
	path := r.URL.Path

	if r.Method == http.MethodPost && (path == "/transfer" || path == "/credits") {
		return true
	}

	if r.Method == http.MethodPost && strings.HasPrefix(path, "/accounts/") {
		return strings.HasSuffix(path, "/deposit") || strings.HasSuffix(path, "/withdraw")
	}

	if r.Method == http.MethodPost && strings.HasPrefix(path, "/cards/") {
		return strings.HasSuffix(path, "/pay")
	}

	return false
}
