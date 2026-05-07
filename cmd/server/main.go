package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
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
		"server_port":                          cfg.ServerPort,
		"max_request_body_bytes":               cfg.Security.MaxRequestBodyBytes,
		"rate_limit_enabled":                   cfg.Security.RateLimit.Enabled,
		"idempotency_enabled":                  cfg.Security.Idempotency.Enabled,
		"idempotency_required":                 cfg.Security.Idempotency.Required,
		"idempotency_retention_seconds":        cfg.Security.Idempotency.Retention.Seconds(),
		"idempotency_cleanup_interval_seconds": cfg.Security.Idempotency.CleanupInterval.Seconds(),
		"cbr_cache_ttl_seconds":                cfg.Security.CBRCacheTTL.Seconds(),
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

	dbInfoCtx, cancelDBInfo := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelDBInfo()

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

	appCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	auditRepository := repositories.NewAuditRepository(database)

	cbrClient := cbr.NewClient(cfg.CBRURL)
	smtpClient := smtpclient.NewClient(cfg.SMTP)

	auditService := services.NewAuditService(auditRepository, logger)
	authService := services.NewAuthService(
		userRepository,
		tokenRepository,
		userSessionRepository,
		cfg.JWTSecret,
	)
	notificationService := services.NewNotificationService(userRepository, smtpClient)
	adminService := services.NewAdminService(adminRepository)
	mfaService := services.NewMFAService(
		mfaRepository,
		accountRepository,
		cardRepository,
		notificationService,
		cfg.Security.MFA.MaxFailures,
		cfg.Security.MFA.Lockout,
		auditService,
	)
	accountService := services.NewAccountService(accountRepository, mfaService)
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
		auditService,
		logger,
	)
	analyticsService := services.NewAnalyticsService(analyticsRepository)

	creditPaymentScheduler := scheduler.NewCreditPaymentScheduler(creditPaymentService, logger)
	creditPaymentScheduler.Start(appCtx)
	tokenCleanupScheduler := scheduler.NewTokenCleanupScheduler(tokenRepository, logger)
	tokenCleanupScheduler.Start(appCtx)
	mfaCleanupScheduler := scheduler.NewMFACleanupScheduler(mfaRepository, logger)
	mfaCleanupScheduler.Start(appCtx)
	idempotencyCleanupScheduler := scheduler.NewIdempotencyCleanupScheduler(
		idempotencyRepository,
		logger,
		cfg.Security.Idempotency.CleanupInterval,
		cfg.Security.Idempotency.Retention,
	)
	idempotencyCleanupScheduler.Start(appCtx)

	healthHandler := handlers.NewHealthHandler(database)
	authHandler := handlers.NewAuthHandler(authService, auditService)
	accountHandler := handlers.NewAccountHandler(accountService, auditService)
	transferHandler := handlers.NewTransferHandler(transferService, auditService)
	cardHandler := handlers.NewCardHandler(cardService, auditService)
	rateHandler := handlers.NewRateHandler(rateService)
	creditHandler := handlers.NewCreditHandler(creditService, auditService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	mfaHandler := handlers.NewMFAHandler(mfaService, auditService)
	adminHandler := handlers.NewAdminHandler(adminService, auditService)

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
		auditService,
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

	go func() {
		logger.Infof("server started on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("server stopped unexpectedly")
		}
	}()

	<-appCtx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("graceful shutdown failed")
		return
	}

	logger.Info("server stopped gracefully")
}
