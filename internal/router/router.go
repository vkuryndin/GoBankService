package router

import (
	"net/http"

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
	jwtSecret string,
	logger *logrus.Logger,
) *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.RequestLogger(logger))

	r.HandleFunc("/health", healthHandler.Health).Methods(http.MethodGet)
	r.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	protected := r.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware(jwtSecret, tokenRepository))

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
