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
	SupabaseURL       string
	SupabaseAnonKey   string
	SupabaseJWTSecret string
}

// Load reads application configuration from the environment.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	return &Config{
		Port:            getEnv("APP_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		MigrateURL:      getEnv("MIGRATE_URL", ""),
		SupabaseURL:     getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey: getEnv("SUPABASE_ANON_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		log.Println("Cargando URL de Supabase:", os.Getenv("SUPABASE_URL"))
		return v
	}
	return fallback
}
