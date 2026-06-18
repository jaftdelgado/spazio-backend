package users

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/email"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/storage"
)

const refreshTokenMaxAgeSeconds = 30 * 24 * 60 * 60

type Module struct {
	handler    *Handler
	db         *pgxpool.Pool
	jwtService auth.JWTService
}

func NewModule(db *pgxpool.Pool, cfg *config.Config, emailSender email.EmailSender, jwtService auth.JWTService, r2Client *storage.R2Client) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, emailSender, jwtService, r2Client, cfg.AppName, cfg.JWTSecret)
	handler := NewHandler(svc, CookieConfig{
		Secure:             cfg.IsProduction,
		AccessTokenMaxAge:  cfg.JWTExpiryMinutes * 60,
		RefreshTokenMaxAge: refreshTokenMaxAgeSeconds,
	})

	return &Module{
		handler:    handler,
		db:         db,
		jwtService: jwtService,
	}
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/api/v1/users")
	{
		userGroup.POST("/pre-register", m.handler.PreRegister)
		userGroup.POST("/verify-email", m.handler.VerifyEmail)
		userGroup.POST("/register", m.handler.Register)
		userGroup.POST("/login", m.handler.Login)
		userGroup.POST("/refresh", m.handler.Refresh)
		userGroup.POST("/forgot-password", m.handler.RequestPasswordReset)
		userGroup.POST("/forgot-password/verify", m.handler.VerifyPasswordResetCode)
		userGroup.POST("/forgot-password/reset", m.handler.ResetPassword)

		protected := userGroup.Group("")
		protected.Use(middleware.Auth(m.jwtService, m.db))
		protected.POST("/logout", m.handler.Logout)
		protected.GET("/profile", m.handler.GetProfile)
		protected.PUT("/profile", m.handler.UpdateProfile)
		protected.POST("/profile/photo", m.handler.UploadProfilePhoto)
		protected.POST("/profile/email/request", m.handler.RequestEmailChange)
		protected.POST("/profile/email/verify", m.handler.VerifyEmailChange)
		protected.PUT("/profile/email", m.handler.ConfirmEmailChange)
		protected.PUT("/profile/password", m.handler.ChangePassword)
		protected.DELETE("/profile", m.handler.DeleteAccount)

		adminOnly := protected.Group("")
		adminOnly.Use(middleware.RequireRole("Admin"))
		adminOnly.GET("/agents", m.handler.ListAgents)
		adminOnly.POST("/staff", m.handler.AdminCreateUser)
	}
}
