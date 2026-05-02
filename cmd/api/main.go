package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/jaftdelgado/spazio-backend/docs"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/modules/catalogs"
	"github.com/jaftdelgado/spazio-backend/internal/modules/clauses"
	"github.com/jaftdelgado/spazio-backend/internal/modules/locations"
	"github.com/jaftdelgado/spazio-backend/internal/modules/properties"
	"github.com/jaftdelgado/spazio-backend/internal/modules/services"
	"github.com/jaftdelgado/spazio-backend/internal/modules/uploads"
	"github.com/jaftdelgado/spazio-backend/internal/modules/users"
	"github.com/jaftdelgado/spazio-backend/internal/modules/visits"
	"github.com/jaftdelgado/spazio-backend/internal/storage"
)

// @title Spazio API
// @version 1.0
// @description API de Spazio Backend
// @host localhost:8080
// @BasePath /
func main() {
	cfg := config.Load()

	database, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	r2, err := storage.NewR2Client(cfg.R2)
	if err != nil {
		log.Fatal(err)
	}

	propertiesModule := properties.NewModule(database)
	servicesModule := services.NewModule(database)
	catalogsModule := catalogs.NewModule(database)
	clausesModule := clauses.NewModule(database)
	locationsModule := locations.NewModule(database)
	usersModule := users.NewModule(database, cfg)
	uploadsModule := uploads.NewModule(database, r2)
	visitsModule := visits.NewModule(database)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.SetTrustedProxies(nil)
	r.Use(middleware.CORS())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("")
	propertiesModule.RegisterRoutes(api)
	servicesModule.RegisterRoutes(api)
	catalogsModule.RegisterRoutes(api)
	clausesModule.RegisterRoutes(api)
	locationsModule.RegisterRoutes(api)
	usersModule.RegisterRoutes(api)
	uploadsModule.RegisterRoutes(api)
	visitsModule.RegisterRoutes(api)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}

}
