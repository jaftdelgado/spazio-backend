package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	MigrateURL        string
	R2                R2Config
	SupabaseURL       string
	SupabaseAnonKey   string
	SupabaseJWTSecret string
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

	return &Config{
		Port:        getEnv("APP_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		MigrateURL:  getEnv("MIGRATE_URL", ""),
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnv("R2_BUCKET_NAME", ""),
			PublicURL:       getEnv("R2_PUBLIC_URL", ""),
		},
		SupabaseURL:       getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:   getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseJWTSecret: getEnv("SUPABASE_JWT_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
