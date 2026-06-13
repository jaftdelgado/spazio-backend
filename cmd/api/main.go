package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/jaftdelgado/spazio-backend/docs"
	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/email"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/modules/catalogs"
	"github.com/jaftdelgado/spazio-backend/internal/modules/clauses"
	"github.com/jaftdelgado/spazio-backend/internal/modules/contracts"
	"github.com/jaftdelgado/spazio-backend/internal/modules/locations"
	"github.com/jaftdelgado/spazio-backend/internal/modules/payments"
	"github.com/jaftdelgado/spazio-backend/internal/modules/properties"
	"github.com/jaftdelgado/spazio-backend/internal/modules/rentals"
	"github.com/jaftdelgado/spazio-backend/internal/modules/sales"
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

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiryMinutes)
	emailSender := email.NewResendSenderWithFrom(cfg.ResendAPIKey, cfg.ResendFromEmail)

	propertiesModule := properties.NewModule(database, r2)
	servicesModule := services.NewModule(database)
	catalogsModule := catalogs.NewModule(database)
	clausesModule := clauses.NewModule(database)
	locationsModule := locations.NewModule(database)
	paymentsModule := payments.NewModule(database, cfg.MercadoPagoAccessToken, cfg.MercadoPagoWebhookSecret)
	usersModule := users.NewModule(database, cfg, emailSender, jwtService)
	uploadsModule := uploads.NewModule(database, r2)
	visitsModule := visits.NewModule(database)
	contractsModule := contracts.NewModule(database, r2)
	rentalsModule := rentals.NewModule(database, cfg.Port)
	salesModule := sales.NewModule(database, cfg.Port)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.SetTrustedProxies(nil)
	r.Use(middleware.CORS(cfg.AllowedOrigin))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	public := r.Group("")
	usersModule.RegisterRoutes(public)
	catalogsModule.RegisterRoutes(public)
	locationsModule.RegisterRoutes(public)

	protected := r.Group("")
	protected.Use(middleware.Auth(jwtService, database))
	{
		propertiesModule.RegisterRoutes(protected)
		servicesModule.RegisterRoutes(protected)
		clausesModule.RegisterRoutes(protected)
		paymentsModule.RegisterRoutes(protected, public)
		uploadsModule.RegisterRoutes(protected)
		visitsModule.RegisterRoutes(protected)
		contractsModule.RegisterRoutes(protected)
		rentalsModule.RegisterRoutes(protected)
		salesModule.RegisterRoutes(protected)
	}

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}

	log.Default().Printf("Server running on port %s", cfg.Port)
}
