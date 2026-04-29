package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/modules/catalogs"
	"github.com/jaftdelgado/spazio-backend/internal/modules/clauses"
	"github.com/jaftdelgado/spazio-backend/internal/modules/locations"
	"github.com/jaftdelgado/spazio-backend/internal/modules/properties"
	"github.com/jaftdelgado/spazio-backend/internal/modules/services"
)

func main() {
	cfg := config.Load()
	log.Println("DATABASE_URL:", cfg.DatabaseURL)

	database, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	propertiesModule := properties.NewModule(database)
	servicesModule := services.NewModule(database)
	catalogsModule := catalogs.NewModule(database)
	clausesModule := clauses.NewModule(database)
	locationsModule := locations.NewModule(database)

	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("")
	propertiesModule.RegisterRoutes(api)
	servicesModule.RegisterRoutes(api)
	catalogsModule.RegisterRoutes(api)
	clausesModule.RegisterRoutes(api)
	locationsModule.RegisterRoutes(api)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
