package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found, using system environment variables")
	}

	direction := flag.String("direction", "up", "Dirección de la migración: up | down")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not defined")
	}

	m, err := migrate.New(
		"file://migration/sql",
		databaseURL,
	)

	if err != nil {
		log.Fatal("Error creating migrator: ", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal("Error applying migrations: ", err)
		}

		log.Println("[OK] UP migrations applied")
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal("Error rolling back migrations: ", err)
		}
		log.Println("[OK] DOWN migrations applied")
	default:
		log.Fatalf("Invalid direction: '%s' - use 'up' or 'down'", *direction)
	}
}
