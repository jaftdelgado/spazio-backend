package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/db"
	"github.com/jaftdelgado/spazio-backend/internal/handlers"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/repository"
	"github.com/jaftdelgado/spazio-backend/internal/services"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func main() {
	cfg := config.Load()
	log.Println("DATABASE_URL:", cfg.DatabaseURL)
	database, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	queries := sqlcgen.New(database)
	propertyRepository := repository.NewPropertyRepository(queries)
	propertyService := services.NewPropertyService(propertyRepository)
	propertyHandler := handlers.NewPropertyHandler(propertyService)

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/properties", propertyHandler.CreateProperty)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
