package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultCBRURL = "https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx"

type Config struct {
	ServerPort     string
	DatabaseURL    string
	JWTSecret      string
	CardPGPKey     string
	CardHMACSecret string
	CBRURL         string
	SMTP           SMTPConfig
}

type SMTPConfig struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	Password string
	From     string
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

	return Config{
		ServerPort:     serverPort,
		DatabaseURL:    databaseURL,
		JWTSecret:      jwtSecret,
		CardPGPKey:     cardPGPKey,
		CardHMACSecret: cardHMACSecret,
		CBRURL:         cbrURL,
		SMTP:           smtpConfig,
	}, nil
}

func loadSMTPConfig() (SMTPConfig, error) {
	enabledRaw := strings.TrimSpace(os.Getenv("SMTP_ENABLED"))
	enabled := strings.EqualFold(enabledRaw, "true")

	port := 587
	portRaw := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if portRaw != "" {
		parsedPort, err := strconv.Atoi(portRaw)
		if err != nil {
			return SMTPConfig{}, errors.New("SMTP_PORT must be a number")
		}

		port = parsedPort
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
