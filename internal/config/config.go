package config

import (
	"errors"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const defaultCBRURL = "https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx"

var moneyValueRegexp = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

type Config struct {
	ServerPort     string
	DatabaseURL    string
	JWTSecret      string
	CardPGPKey     string
	CardHMACSecret string
	CBRURL         string
	SMTP           SMTPConfig
	Server         ServerConfig
	Security       SecurityConfig
	CreditPolicy   CreditPolicyConfig
}

type SMTPConfig struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

type SecurityConfig struct {
	MaxRequestBodyBytes int64
	RateLimit           RateLimitConfig
	MFA                 AttemptLimitConfig
	CVV                 AttemptLimitConfig
	CBRCacheTTL         time.Duration
	Idempotency         IdempotencyConfig
}

type RateLimitConfig struct {
	Enabled           bool
	CleanupInterval   time.Duration
	GlobalRequests    int
	GlobalWindow      time.Duration
	LoginRequests     int
	LoginWindow       time.Duration
	RegisterRequests  int
	RegisterWindow    time.Duration
	MFARequests       int
	MFAWindow         time.Duration
	FinancialRequests int
	FinancialWindow   time.Duration
	AdminRequests     int
	AdminWindow       time.Duration
	RateRequests      int
	RateWindow        time.Duration
}

type AttemptLimitConfig struct {
	MaxFailures int
	Lockout     time.Duration
}

type IdempotencyConfig struct {
	Enabled         bool
	Required        bool
	Retention       time.Duration
	CleanupInterval time.Duration
}

type CreditPolicyConfig struct {
	Enabled                 bool
	MaxActiveCredits        int
	MaxPrincipalAmount      string
	MaxTotalPrincipalAmount string
	MaxDebtLoadPercent      int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	serverPort := strings.TrimSpace(os.Getenv("SERVER_PORT"))
	if serverPort == "" {
		serverPort = "8080"
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	cardPGPKey := strings.TrimSpace(os.Getenv("CARD_PGP_KEY"))
	if cardPGPKey == "" {
		return Config{}, errors.New("CARD_PGP_KEY is required")
	}

	cardHMACSecret := strings.TrimSpace(os.Getenv("CARD_HMAC_SECRET"))
	if cardHMACSecret == "" {
		return Config{}, errors.New("CARD_HMAC_SECRET is required")
	}

	cbrURL := strings.TrimSpace(os.Getenv("CBR_URL"))
	if cbrURL == "" {
		cbrURL = defaultCBRURL
	}

	smtpConfig, err := loadSMTPConfig()
	if err != nil {
		return Config{}, err
	}

	serverConfig, err := loadServerConfig()
	if err != nil {
		return Config{}, err
	}

	securityConfig, err := loadSecurityConfig()
	if err != nil {
		return Config{}, err
	}

	creditPolicyConfig, err := loadCreditPolicyConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		ServerPort:     serverPort,
		DatabaseURL:    databaseURL,
		JWTSecret:      jwtSecret,
		CardPGPKey:     cardPGPKey,
		CardHMACSecret: cardHMACSecret,
		CBRURL:         cbrURL,
		SMTP:           smtpConfig,
		Server:         serverConfig,
		Security:       securityConfig,
		CreditPolicy:   creditPolicyConfig,
	}, nil
}

func loadCreditPolicyConfig() (CreditPolicyConfig, error) {
	maxActiveCredits, err := envInt("CREDIT_MAX_ACTIVE_CREDITS", 3)
	if err != nil {
		return CreditPolicyConfig{}, err
	}
	if maxActiveCredits < 0 {
		return CreditPolicyConfig{}, errors.New("CREDIT_MAX_ACTIVE_CREDITS must be non-negative")
	}

	maxPrincipalAmount, err := envMoney("CREDIT_MAX_PRINCIPAL_AMOUNT", "1000000.00")
	if err != nil {
		return CreditPolicyConfig{}, err
	}

	maxTotalPrincipalAmount, err := envMoney("CREDIT_MAX_TOTAL_PRINCIPAL_AMOUNT", "3000000.00")
	if err != nil {
		return CreditPolicyConfig{}, err
	}

	maxDebtLoadPercent, err := envInt("CREDIT_MAX_DEBT_LOAD_PERCENT", 50)
	if err != nil {
		return CreditPolicyConfig{}, err
	}
	if maxDebtLoadPercent < 0 || maxDebtLoadPercent > 100 {
		return CreditPolicyConfig{}, errors.New("CREDIT_MAX_DEBT_LOAD_PERCENT must be between 0 and 100")
	}

	return CreditPolicyConfig{
		Enabled:                 envBool("CREDIT_POLICY_ENABLED", true),
		MaxActiveCredits:        maxActiveCredits,
		MaxPrincipalAmount:      maxPrincipalAmount,
		MaxTotalPrincipalAmount: maxTotalPrincipalAmount,
		MaxDebtLoadPercent:      maxDebtLoadPercent,
	}, nil
}

func loadSMTPConfig() (SMTPConfig, error) {
	enabledRaw := strings.TrimSpace(os.Getenv("SMTP_ENABLED"))
	enabled := strings.EqualFold(enabledRaw, "true")

	port, err := envInt("SMTP_PORT", 587)
	if err != nil {
		return SMTPConfig{}, errors.New("SMTP_PORT must be a number")
	}

	config := SMTPConfig{
		Enabled:  enabled,
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     port,
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}

	if !enabled {
		return config, nil
	}

	if config.Host == "" {
		return SMTPConfig{}, errors.New("SMTP_HOST is required when SMTP_ENABLED=true")
	}

	if config.User == "" {
		return SMTPConfig{}, errors.New("SMTP_USER is required when SMTP_ENABLED=true")
	}

	if config.Password == "" {
		return SMTPConfig{}, errors.New("SMTP_PASSWORD is required when SMTP_ENABLED=true")
	}

	if config.From == "" {
		return SMTPConfig{}, errors.New("SMTP_FROM is required when SMTP_ENABLED=true")
	}

	return config, nil
}

func loadServerConfig() (ServerConfig, error) {
	readHeaderTimeout, err := envDurationSeconds("SERVER_READ_HEADER_TIMEOUT_SECONDS", 5*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}

	readTimeout, err := envDurationSeconds("SERVER_READ_TIMEOUT_SECONDS", 15*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}

	writeTimeout, err := envDurationSeconds("SERVER_WRITE_TIMEOUT_SECONDS", 30*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}

	idleTimeout, err := envDurationSeconds("SERVER_IDLE_TIMEOUT_SECONDS", 60*time.Second)
	if err != nil {
		return ServerConfig{}, err
	}

	maxHeaderBytes, err := envInt("SERVER_MAX_HEADER_BYTES", 1<<20)
	if err != nil {
		return ServerConfig{}, err
	}

	return ServerConfig{
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}, nil
}

func loadSecurityConfig() (SecurityConfig, error) {
	maxRequestBodyBytes, err := envInt64("SECURITY_MAX_REQUEST_BODY_BYTES", 1<<20)
	if err != nil {
		return SecurityConfig{}, err
	}

	cbrCacheTTL, err := envDurationSeconds("CBR_CACHE_TTL_SECONDS", time.Hour)
	if err != nil {
		return SecurityConfig{}, err
	}

	rateLimitConfig, err := loadRateLimitConfig()
	if err != nil {
		return SecurityConfig{}, err
	}

	mfaConfig, err := loadAttemptLimitConfig("MFA", 5, 10*time.Minute)
	if err != nil {
		return SecurityConfig{}, err
	}

	cvvConfig, err := loadAttemptLimitConfig("CVV", 5, 10*time.Minute)
	if err != nil {
		return SecurityConfig{}, err
	}

	idempotencyRetention, err := envDurationSeconds("IDEMPOTENCY_RETENTION_SECONDS", 24*time.Hour)
	if err != nil {
		return SecurityConfig{}, err
	}

	idempotencyCleanupInterval, err := envDurationSeconds("IDEMPOTENCY_CLEANUP_INTERVAL_SECONDS", time.Hour)
	if err != nil {
		return SecurityConfig{}, err
	}

	return SecurityConfig{
		MaxRequestBodyBytes: maxRequestBodyBytes,
		RateLimit:           rateLimitConfig,
		MFA:                 mfaConfig,
		CVV:                 cvvConfig,
		CBRCacheTTL:         cbrCacheTTL,
		Idempotency: IdempotencyConfig{
			Enabled:         envBool("IDEMPOTENCY_ENABLED", true),
			Required:        envBool("IDEMPOTENCY_REQUIRED", false),
			Retention:       idempotencyRetention,
			CleanupInterval: idempotencyCleanupInterval,
		},
	}, nil
}

func loadRateLimitConfig() (RateLimitConfig, error) {
	cleanupInterval, err := envDurationSeconds("RATE_LIMIT_CLEANUP_INTERVAL_SECONDS", 5*time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	globalWindow, err := envDurationSeconds("RATE_LIMIT_GLOBAL_WINDOW_SECONDS", time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	loginWindow, err := envDurationSeconds("RATE_LIMIT_LOGIN_WINDOW_SECONDS", time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	registerWindow, err := envDurationSeconds("RATE_LIMIT_REGISTER_WINDOW_SECONDS", time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	mfaWindow, err := envDurationSeconds("RATE_LIMIT_MFA_WINDOW_SECONDS", 5*time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	financialWindow, err := envDurationSeconds("RATE_LIMIT_FINANCIAL_WINDOW_SECONDS", time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	adminWindow, err := envDurationSeconds("RATE_LIMIT_ADMIN_WINDOW_SECONDS", time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	rateWindow, err := envDurationSeconds("RATE_LIMIT_RATE_WINDOW_SECONDS", time.Minute)
	if err != nil {
		return RateLimitConfig{}, err
	}

	return RateLimitConfig{
		Enabled:           envBool("RATE_LIMIT_ENABLED", true),
		CleanupInterval:   cleanupInterval,
		GlobalRequests:    mustEnvInt("RATE_LIMIT_GLOBAL_REQUESTS", 1000),
		GlobalWindow:      globalWindow,
		LoginRequests:     mustEnvInt("RATE_LIMIT_LOGIN_REQUESTS", 30),
		LoginWindow:       loginWindow,
		RegisterRequests:  mustEnvInt("RATE_LIMIT_REGISTER_REQUESTS", 30),
		RegisterWindow:    registerWindow,
		MFARequests:       mustEnvInt("RATE_LIMIT_MFA_REQUESTS", 30),
		MFAWindow:         mfaWindow,
		FinancialRequests: mustEnvInt("RATE_LIMIT_FINANCIAL_REQUESTS", 120),
		FinancialWindow:   financialWindow,
		AdminRequests:     mustEnvInt("RATE_LIMIT_ADMIN_REQUESTS", 120),
		AdminWindow:       adminWindow,
		RateRequests:      mustEnvInt("RATE_LIMIT_RATE_REQUESTS", 120),
		RateWindow:        rateWindow,
	}, nil
}

func loadAttemptLimitConfig(prefix string, defaultMaxFailures int, defaultLockout time.Duration) (AttemptLimitConfig, error) {
	maxFailures, err := envInt(prefix+"_MAX_FAILED_ATTEMPTS", defaultMaxFailures)
	if err != nil {
		return AttemptLimitConfig{}, err
	}

	lockout, err := envDurationSeconds(prefix+"_LOCKOUT_SECONDS", defaultLockout)
	if err != nil {
		return AttemptLimitConfig{}, err
	}

	return AttemptLimitConfig{
		MaxFailures: maxFailures,
		Lockout:     lockout,
	}, nil
}

func envMoney(name string, defaultValue string) (string, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		raw = defaultValue
	}

	if !moneyValueRegexp.MatchString(raw) {
		return "", errors.New(name + " must be a positive money amount with up to 2 decimal places")
	}

	value, ok := new(big.Rat).SetString(raw)
	if !ok || value.Sign() <= 0 {
		return "", errors.New(name + " must be positive")
	}

	return raw, nil
}

func envBool(name string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}

	return strings.EqualFold(raw, "true") || raw == "1"
}

func envInt(name string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(name + " must be a number")
	}

	return value, nil
}

func mustEnvInt(name string, defaultValue int) int {
	value, err := envInt(name, defaultValue)
	if err != nil {
		return defaultValue
	}

	return value
}

func envInt64(name string, defaultValue int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, errors.New(name + " must be a number")
	}

	return value, nil
}

func envDurationSeconds(name string, defaultValue time.Duration) (time.Duration, error) {
	seconds, err := envInt64(name, int64(defaultValue/time.Second))
	if err != nil {
		return 0, err
	}

	if seconds < 0 {
		return 0, errors.New(name + " must be non-negative")
	}

	return time.Duration(seconds) * time.Second, nil
}
