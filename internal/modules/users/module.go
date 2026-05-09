package users

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
)

type Module struct {
	handler *Handler
	db      *pgxpool.Pool
	cfg     *config.Config
}

func NewModule(db *pgxpool.Pool, cfg *config.Config) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, cfg)
	handler := NewHandler(svc)

	return &Module{
		handler: handler,
		db:      db,
		cfg:     cfg,
	}
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/users")
	{

		userGroup.POST("/register", m.handler.Register)
		userGroup.POST("/verify", m.handler.Verify)
		userGroup.POST("/login", m.handler.Login)

		userGroup.PUT("/profile", middleware.Auth(m.cfg.SupabaseURL, m.cfg.SupabaseAnonKey, m.db), m.handler.UpdateProfile)
		userGroup.DELETE("/DeleteProfile", middleware.Auth(m.cfg.SupabaseURL, m.cfg.SupabaseAnonKey, m.db), m.handler.DeleteAccount)
	}
}
