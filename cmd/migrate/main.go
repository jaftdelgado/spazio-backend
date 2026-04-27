package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found, using system environment variables")
	}

	direction := flag.String("direction", "up", "Dirección de la migración: up | down | force")
	version := flag.Int("version", 0, "Versión a forzar (solo con -direction force)")
	flag.Parse()

	databaseURL := os.Getenv("MIGRATE_URL")
	if databaseURL == "" {
		log.Fatal("MIGRATE_URL is not defined")
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

	case "force":
		if *version == 0 {
			log.Fatal("Debes indicar la versión con -version <número>, e.g.: -direction force -version 3")
		}
		if err := m.Force(*version); err != nil {
			log.Fatal("Error forzando versión: ", err)
		}
		log.Printf("[OK] Versión forzada a %s", strconv.Itoa(*version))

	default:
		log.Fatalf("Invalid direction: '%s' - use 'up', 'down' or 'force'", *direction)
	}
}
