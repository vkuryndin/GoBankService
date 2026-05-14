package router

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bank-service/internal/audit"
	"bank-service/internal/config"
	"bank-service/internal/handlers"
	"bank-service/internal/middleware"
	"bank-service/internal/repositories"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Dependencies struct {
	AppContext            context.Context
	HealthHandler         *handlers.HealthHandler
	AuthHandler           *handlers.AuthHandler
	AccountHandler        *handlers.AccountHandler
	TransferHandler       *handlers.TransferHandler
	CardHandler           *handlers.CardHandler
	RateHandler           *handlers.RateHandler
	CreditHandler         *handlers.CreditHandler
	NotificationHandler   *handlers.NotificationHandler
	AnalyticsHandler      *handlers.AnalyticsHandler
	MFAHandler            *handlers.MFAHandler
	AdminHandler          *handlers.AdminHandler
	TokenRepository       *repositories.TokenRepository
	UserRepository        *repositories.UserRepository
	IdempotencyRepository *repositories.IdempotencyRepository
	JWTSecret             string
	RequestTimeout        time.Duration
	SecurityConfig        config.SecurityConfig
	AuditRecorder         audit.Recorder
	Logger                *logrus.Logger
}

func NewRouter(deps Dependencies) *mux.Router {
	r := mux.NewRouter()

	middleware.ConfigureTrustedProxies(deps.SecurityConfig.RateLimit.TrustedProxies)

	r.Use(middleware.RequestID())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(middleware.CORSConfig{
		Enabled:          deps.SecurityConfig.CORS.Enabled,
		AllowedOrigins:   deps.SecurityConfig.CORS.AllowedOrigins,
		AllowedMethods:   deps.SecurityConfig.CORS.AllowedMethods,
		AllowedHeaders:   deps.SecurityConfig.CORS.AllowedHeaders,
		AllowCredentials: deps.SecurityConfig.CORS.AllowCredentials,
		MaxAgeSeconds:    deps.SecurityConfig.CORS.MaxAgeSeconds,
	}))
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(middleware.RequestContextTimeout(deps.RequestTimeout))
	r.Use(middleware.MaxRequestBodySize(deps.SecurityConfig.MaxRequestBodyBytes))
	r.Use(buildPublicRateLimiter(deps.AppContext, deps.SecurityConfig.RateLimit, deps.AuditRecorder))

	r.HandleFunc("/health", deps.HealthHandler.Health).Methods(http.MethodGet)
	r.HandleFunc("/register", deps.AuthHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", deps.AuthHandler.Login).Methods(http.MethodPost)

	tokenChecker := middleware.NewCachedTokenRevocationChecker(
		deps.TokenRepository,
		deps.SecurityConfig.TokenRevocationCacheTTL,
		deps.Logger,
	)

	protected := r.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware(deps.JWTSecret, tokenChecker))
	protected.Use(buildProtectedRateLimiter(deps.AppContext, deps.SecurityConfig.RateLimit, deps.AuditRecorder))
	protected.Use(middleware.IdempotencyMiddleware(
		deps.IdempotencyRepository,
		middleware.IdempotencyConfig{
			Enabled:  deps.SecurityConfig.Idempotency.Enabled,
			Required: deps.SecurityConfig.Idempotency.Required,
		},
		deps.Logger,
		deps.AuditRecorder,
	))

	protected.HandleFunc("/auth/check", deps.AuthHandler.CheckAuth).Methods(http.MethodGet)
	protected.HandleFunc("/logout", deps.AuthHandler.Logout).Methods(http.MethodPost)

	protected.HandleFunc("/mfa/request", deps.MFAHandler.RequestCode).Methods(http.MethodPost)

	protected.HandleFunc("/accounts", deps.AccountHandler.CreateAccount).Methods(http.MethodPost)
	protected.HandleFunc("/accounts", deps.AccountHandler.GetUserAccounts).Methods(http.MethodGet)
	protected.HandleFunc("/accounts/{accountId}", deps.AccountHandler.GetAccount).Methods(http.MethodGet)
	protected.HandleFunc("/accounts/{accountId}/deposit", deps.AccountHandler.Deposit).Methods(http.MethodPost)
	protected.HandleFunc("/accounts/{accountId}/withdraw", deps.AccountHandler.Withdraw).Methods(http.MethodPost)
	protected.HandleFunc("/accounts/{accountId}/close", deps.AccountHandler.CloseAccount).Methods(http.MethodPost)
	protected.HandleFunc("/accounts/{accountId}/predict", deps.AnalyticsHandler.PredictBalance).Methods(http.MethodGet)
	protected.HandleFunc("/accounts/{accountId}/operations/statistics", deps.AnalyticsHandler.GetAccountOperationStatistics).Methods(http.MethodGet)

	protected.HandleFunc("/transfer", deps.TransferHandler.Transfer).Methods(http.MethodPost)

	protected.HandleFunc("/cards", deps.CardHandler.CreateCard).Methods(http.MethodPost)
	protected.HandleFunc("/cards", deps.CardHandler.GetUserCards).Methods(http.MethodGet)
	protected.HandleFunc("/cards/{cardId}", deps.CardHandler.GetCard).Methods(http.MethodGet)
	protected.HandleFunc("/cards/{cardId}/reveal", deps.CardHandler.RevealCard).Methods(http.MethodPost)
	protected.HandleFunc("/cards/{cardId}/operations/statistics", deps.AnalyticsHandler.GetCardOperationStatistics).Methods(http.MethodGet)
	protected.HandleFunc("/cards/{cardId}/close", deps.CardHandler.CloseCard).Methods(http.MethodPost)
	protected.HandleFunc("/cards/{cardId}/pay", deps.CardHandler.PayByCard).Methods(http.MethodPost)
	protected.HandleFunc("/cards/{cardId}/transfer", deps.CardHandler.TransferByCard).Methods(http.MethodPost)

	protected.HandleFunc("/rates/key", deps.RateHandler.GetKeyRate).Methods(http.MethodGet)

	protected.HandleFunc("/credits/check", deps.CreditHandler.CheckCredit).Methods(http.MethodPost)
	protected.HandleFunc("/credits", deps.CreditHandler.CreateCredit).Methods(http.MethodPost)
	protected.HandleFunc("/credits", deps.CreditHandler.GetUserCredits).Methods(http.MethodGet)
	protected.HandleFunc("/credits/{creditId}", deps.CreditHandler.GetCredit).Methods(http.MethodGet)
	protected.HandleFunc("/credits/{creditId}/schedule", deps.CreditHandler.GetCreditSchedule).Methods(http.MethodGet)

	protected.HandleFunc("/notifications/test", deps.NotificationHandler.SendTestEmail).Methods(http.MethodGet)

	protected.HandleFunc("/analytics", deps.AnalyticsHandler.GetAnalytics).Methods(http.MethodGet)

	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.AdminMiddleware(deps.UserRepository))

	admin.HandleFunc("/users", deps.AdminHandler.GetUsers).Methods(http.MethodGet)
	admin.HandleFunc("/logged-in-users", deps.AdminHandler.GetLoggedInUsers).Methods(http.MethodGet)
	admin.HandleFunc("/accounts/{accountId}/block", deps.AdminHandler.BlockAccount).Methods(http.MethodPost)
	admin.HandleFunc("/accounts/{accountId}/unblock", deps.AdminHandler.UnblockAccount).Methods(http.MethodPost)

	return r
}

func buildPublicRateLimiter(ctx context.Context, config config.RateLimitConfig, auditRecorder audit.Recorder) func(http.Handler) http.Handler {
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

	return middleware.NewRateLimiterWithContext(ctx, config.Enabled, rules, config.CleanupInterval, auditRecorder)
}

func buildProtectedRateLimiter(ctx context.Context, config config.RateLimitConfig, auditRecorder audit.Recorder) func(http.Handler) http.Handler {
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

	return middleware.NewRateLimiterWithContext(ctx, config.Enabled, rules, config.CleanupInterval, auditRecorder)
}

func isFinancialEndpoint(r *http.Request) bool {
	path := r.URL.Path

	if r.Method == http.MethodPost && (path == "/transfer" || path == "/credits") {
		return true
	}

	if r.Method == http.MethodPost && strings.HasPrefix(path, "/accounts/") {
		return strings.HasSuffix(path, "/deposit") || strings.HasSuffix(path, "/withdraw") || strings.HasSuffix(path, "/close")
	}

	if r.Method == http.MethodPost && strings.HasPrefix(path, "/cards/") {
		return strings.HasSuffix(path, "/pay") ||
			strings.HasSuffix(path, "/close") ||
			strings.HasSuffix(path, "/transfer") ||
			strings.HasSuffix(path, "/reveal")
	}

	return false
}
