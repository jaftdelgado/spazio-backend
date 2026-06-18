package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                     string
	DatabaseURL              string
	MigrateURL               string
	R2                       R2Config
	JWTSecret                string
	JWTExpiryMinutes         int
	ResendAPIKey             string
	ResendFromEmail          string
	AppName                  string
	AllowedOrigin            string
	IsProduction             bool
	MercadoPagoAccessToken   string
	MercadoPagoWebhookSecret string
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Println("warning: JWT_SECRET is not configured")
	}

	jwtExpiryMinutes := 60
	if rawExpiry := getEnv("JWT_EXPIRY_MINUTES", "60"); rawExpiry != "" {
		parsedExpiry, err := strconv.Atoi(rawExpiry)
		if err != nil {
			log.Printf("warning: invalid JWT_EXPIRY_MINUTES=%q, using default 60", rawExpiry)
		} else {
			jwtExpiryMinutes = parsedExpiry
		}
	}

	return &Config{
		Port:        getEnvAny([]string{"APP_PORT", "PORT"}, "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		MigrateURL:  getEnv("MIGRATE_URL", ""),
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnv("R2_BUCKET_NAME", ""),
			PublicURL:       getEnv("R2_PUBLIC_BASE_URL", getEnv("R2_PUBLIC_URL", "")),
		},
		JWTSecret:                jwtSecret,
		JWTExpiryMinutes:         jwtExpiryMinutes,
		ResendAPIKey:             getEnv("RESEND_API_KEY", ""),
		ResendFromEmail:          getEnv("RESEND_FROM_EMAIL", "noreply@spazio.com"),
		AppName:                  getEnv("APP_NAME", "Spazio"),
		AllowedOrigin:            getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		IsProduction:             strings.EqualFold(getEnv("APP_ENV", "development"), "production"),
		MercadoPagoAccessToken:   getEnv("MERCADOPAGO_ACCESS_TOKEN", ""),
		MercadoPagoWebhookSecret: getEnv("MERCADOPAGO_WEBHOOK_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAny(keys []string, fallback string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}
