package app

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"sync"
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

func Run() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}

	if cfg.LogFormat == "json" {
		formatter := &logrus.JSONFormatter{}
		logger.SetFormatter(formatter)
		logrus.SetFormatter(formatter)
	} else {
		formatter := &logrus.TextFormatter{FullTimestamp: true}
		logger.SetFormatter(formatter)
		logrus.SetFormatter(formatter)
	}

	logger.WithFields(logrus.Fields{
		"server_port":                          cfg.ServerPort,
		"log_format":                           cfg.LogFormat,
		"request_timeout_seconds":              cfg.Server.RequestTimeout.Seconds(),
		"max_request_body_bytes":               cfg.Security.MaxRequestBodyBytes,
		"rate_limit_enabled":                   cfg.Security.RateLimit.Enabled,
		"idempotency_enabled":                  cfg.Security.Idempotency.Enabled,
		"idempotency_required":                 cfg.Security.Idempotency.Required,
		"idempotency_retention_seconds":        cfg.Security.Idempotency.Retention.Seconds(),
		"idempotency_cleanup_interval_seconds": cfg.Security.Idempotency.CleanupInterval.Seconds(),
		"cbr_cache_ttl_seconds":                cfg.Security.CBRCacheTTL.Seconds(),
		"cbr_breaker_failure_limit":            cfg.Security.CBRBreakerFailureLimit,
		"cbr_breaker_reset_timeout_seconds":    cfg.Security.CBRBreakerResetTimeout.Seconds(),
		"token_revocation_cache_ttl_seconds":   cfg.Security.TokenRevocationCacheTTL.Seconds(),
		"mfa_request_cooldown_seconds":         cfg.Security.MFARequestCooldown.Seconds(),
		"cors_enabled":                         cfg.Security.CORS.Enabled,
	}).Info("config loaded")

	logger.WithFields(logrus.Fields{
		"enabled": cfg.SMTP.Enabled,
		"host":    cfg.SMTP.Host,
		"port":    cfg.SMTP.Port,
		"user":    cfg.SMTP.User,
		"from":    cfg.SMTP.From,
	}).Info("smtp config loaded")

	logger.WithFields(logrus.Fields{
		"enabled":                    cfg.CreditPolicy.Enabled,
		"max_active_credits":         cfg.CreditPolicy.MaxActiveCredits,
		"max_principal_amount":       cfg.CreditPolicy.MaxPrincipalAmount,
		"max_total_principal_amount": cfg.CreditPolicy.MaxTotalPrincipalAmount,
		"max_debt_load_percent":      cfg.CreditPolicy.MaxDebtLoadPercent,
	}).Info("credit policy loaded")

	database, err := appdb.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("database connection error: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Warnf("close database: %v", err)
		}
	}()

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
		cfg.Security.MFARequestCooldown,
		auditService,
		cfg.CardPGPKey,
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
	rateService := services.NewRateService(
		cbrClient,
		cfg.Security.CBRCacheTTL,
		cfg.Security.CBRBreakerFailureLimit,
		cfg.Security.CBRBreakerResetTimeout,
		logger,
	)
	creditService := services.NewCreditService(
		creditRepository,
		accountRepository,
		rateService,
		mfaService,
		services.CreditPolicy{
			Enabled:                 cfg.CreditPolicy.Enabled,
			MaxActiveCredits:        cfg.CreditPolicy.MaxActiveCredits,
			MaxPrincipalAmount:      cfg.CreditPolicy.MaxPrincipalAmount,
			MaxTotalPrincipalAmount: cfg.CreditPolicy.MaxTotalPrincipalAmount,
			MaxDebtLoadPercent:      cfg.CreditPolicy.MaxDebtLoadPercent,
		},
	)
	creditPaymentService := services.NewCreditPaymentService(
		creditPaymentRepository,
		notificationService,
		auditService,
		logger,
	)
	analyticsService := services.NewAnalyticsService(analyticsRepository)

	var schedulerWG sync.WaitGroup

	creditPaymentScheduler := scheduler.NewCreditPaymentScheduler(creditPaymentService, logger)
	creditPaymentScheduler.Start(appCtx, &schedulerWG)
	tokenCleanupScheduler := scheduler.NewTokenCleanupScheduler(tokenRepository, logger)
	tokenCleanupScheduler.Start(appCtx, &schedulerWG)
	mfaCleanupScheduler := scheduler.NewMFACleanupScheduler(mfaRepository, logger)
	mfaCleanupScheduler.Start(appCtx, &schedulerWG)
	idempotencyCleanupScheduler := scheduler.NewIdempotencyCleanupScheduler(
		idempotencyRepository,
		logger,
		cfg.Security.Idempotency.CleanupInterval,
		cfg.Security.Idempotency.Retention,
	)
	idempotencyCleanupScheduler.Start(appCtx, &schedulerWG)

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

	appRouter := router.NewRouter(router.Dependencies{
		AppContext:            appCtx,
		HealthHandler:         healthHandler,
		AuthHandler:           authHandler,
		AccountHandler:        accountHandler,
		TransferHandler:       transferHandler,
		CardHandler:           cardHandler,
		RateHandler:           rateHandler,
		CreditHandler:         creditHandler,
		NotificationHandler:   notificationHandler,
		AnalyticsHandler:      analyticsHandler,
		MFAHandler:            mfaHandler,
		AdminHandler:          adminHandler,
		TokenRepository:       tokenRepository,
		UserRepository:        userRepository,
		IdempotencyRepository: idempotencyRepository,
		JWTSecret:             cfg.JWTSecret,
		RequestTimeout:        cfg.Server.RequestTimeout,
		SecurityConfig:        cfg.Security,
		AuditRecorder:         auditService,
		Logger:                logger,
	})

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

	schedulerStopped := make(chan struct{})
	go func() {
		schedulerWG.Wait()
		close(schedulerStopped)
	}()

	select {
	case <-schedulerStopped:
		logger.Info("background schedulers stopped")
	case <-shutdownCtx.Done():
		logger.Warn("background scheduler shutdown timeout")
	}

	logger.Info("server stopped gracefully")
}
