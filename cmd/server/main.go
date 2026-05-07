package main

import (
	"context"
	"net/http"
	"time"

	"bank-service/internal/config"
	appdb "bank-service/internal/db"
	"bank-service/internal/handlers"
	"bank-service/internal/integrations/cbr"
	smtpclient "bank-service/internal/integrations/smtp"
	"bank-service/internal/repositories"
	"bank-service/internal/router"
	"bank-service/internal/scheduler"
	"bank-service/internal/services"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}

	logger.WithFields(logrus.Fields{
		"server_port":            cfg.ServerPort,
		"max_request_body_bytes": cfg.Security.MaxRequestBodyBytes,
		"rate_limit_enabled":     cfg.Security.RateLimit.Enabled,
		"idempotency_enabled":    cfg.Security.Idempotency.Enabled,
		"idempotency_required":   cfg.Security.Idempotency.Required,
		"cbr_cache_ttl_seconds":  cfg.Security.CBRCacheTTL.Seconds(),
	}).Info("config loaded")

	logger.WithFields(logrus.Fields{
		"enabled": cfg.SMTP.Enabled,
		"host":    cfg.SMTP.Host,
		"port":    cfg.SMTP.Port,
		"user":    cfg.SMTP.User,
		"from":    cfg.SMTP.From,
	}).Info("smtp config loaded")

	database, err := appdb.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("database connection error: %v", err)
	}
	defer database.Close()

	logger.Info("database connected")

	dbInfoCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbInfo, err := appdb.GetInfo(dbInfoCtx, database)
	if err != nil {
		logger.Warnf("database info unavailable: %v", err)
	} else {
		logger.WithFields(logrus.Fields{
			"name":   dbInfo.Name,
			"user":   dbInfo.User,
			"schema": dbInfo.Schema,
			"host":   dbInfo.Host,
			"port":   dbInfo.Port,
		}).Info("database info")
	}

	userRepository := repositories.NewUserRepository(database)
	tokenRepository := repositories.NewTokenRepository(database)
	mfaRepository := repositories.NewMFARepository(database)
	accountRepository := repositories.NewAccountRepository(database)
	cardRepository := repositories.NewCardRepository(database)
	creditRepository := repositories.NewCreditRepository(database)
	creditPaymentRepository := repositories.NewCreditPaymentRepository(database)
	analyticsRepository := repositories.NewAnalyticsRepository(database)
	userSessionRepository := repositories.NewUserSessionRepository(database)
	adminRepository := repositories.NewAdminRepository(database)
	idempotencyRepository := repositories.NewIdempotencyRepository(database)

	cbrClient := cbr.NewClient(cfg.CBRURL)
	smtpClient := smtpclient.NewClient(cfg.SMTP)

	authService := services.NewAuthService(
		userRepository,
		tokenRepository,
		userSessionRepository,
		cfg.JWTSecret,
	)
	accountService := services.NewAccountService(accountRepository)
	notificationService := services.NewNotificationService(userRepository, smtpClient)
	adminService := services.NewAdminService(adminRepository)
	mfaService := services.NewMFAService(
		mfaRepository,
		accountRepository,
		cardRepository,
		notificationService,
		cfg.Security.MFA.MaxFailures,
		cfg.Security.MFA.Lockout,
	)
	transferService := services.NewTransferService(accountRepository, mfaService)
	cardProcessingService := services.NewCardProcessingService()
	cardService := services.NewCardService(
		cardRepository,
		accountRepository,
		cardProcessingService,
		mfaService,
		cfg.CardPGPKey,
		cfg.CardHMACSecret,
		cfg.Security.CVV.MaxFailures,
		cfg.Security.CVV.Lockout,
	)
	rateService := services.NewRateService(cbrClient, cfg.Security.CBRCacheTTL)
	creditService := services.NewCreditService(creditRepository, rateService, mfaService)
	creditPaymentService := services.NewCreditPaymentService(
		creditPaymentRepository,
		notificationService,
		logger,
	)
	analyticsService := services.NewAnalyticsService(analyticsRepository)

	creditPaymentScheduler := scheduler.NewCreditPaymentScheduler(creditPaymentService, logger)
	creditPaymentScheduler.Start(context.Background())
	tokenCleanupScheduler := scheduler.NewTokenCleanupScheduler(tokenRepository, logger)
	tokenCleanupScheduler.Start(context.Background())
	mfaCleanupScheduler := scheduler.NewMFACleanupScheduler(mfaRepository, logger)
	mfaCleanupScheduler.Start(context.Background())

	healthHandler := handlers.NewHealthHandler(database)
	authHandler := handlers.NewAuthHandler(authService)
	accountHandler := handlers.NewAccountHandler(accountService)
	transferHandler := handlers.NewTransferHandler(transferService)
	cardHandler := handlers.NewCardHandler(cardService)
	rateHandler := handlers.NewRateHandler(rateService)
	creditHandler := handlers.NewCreditHandler(creditService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	mfaHandler := handlers.NewMFAHandler(mfaService)
	adminHandler := handlers.NewAdminHandler(adminService)

	appRouter := router.NewRouter(
		healthHandler,
		authHandler,
		accountHandler,
		transferHandler,
		cardHandler,
		rateHandler,
		creditHandler,
		notificationHandler,
		analyticsHandler,
		mfaHandler,
		adminHandler,
		tokenRepository,
		userRepository,
		idempotencyRepository,
		cfg.JWTSecret,
		cfg.Security,
		logger,
	)

	addr := ":" + cfg.ServerPort
	server := &http.Server{
		Addr:              addr,
		Handler:           appRouter,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}

	logger.Infof("server started on %s", addr)

	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("server stopped: %v", err)
	}
}
