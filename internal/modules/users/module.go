package users

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/config"
)

type Module struct {
	handler *Handler
}

func NewModule(db *pgxpool.Pool, cfg *config.Config) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, cfg)
	handler := NewHandler(svc)
	return &Module{handler: handler}
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/users")
	{
		userGroup.POST("/register", m.handler.Register)
	}
}
