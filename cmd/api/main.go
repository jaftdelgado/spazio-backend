package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/modules/properties"
)

func main() {
	cfg := config.Load()
	database, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	propertiesModule := properties.NewModule(database)

	r := gin.Default()
	r.Use(middleware.CORS())
	api := r.Group("")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	propertiesModule.RegisterRoutes(api)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
