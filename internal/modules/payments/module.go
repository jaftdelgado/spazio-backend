package payments

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Handler *Handler
}

func NewModule(db *pgxpool.Pool, mpAccessToken string, mpWebhookSecret string) *Module {
	repo := NewRepository(db)
	service := NewService(repo, mpAccessToken, mpWebhookSecret)
	handler := NewHandler(service)

	return &Module{
		Handler: handler,
	}
}

func (m *Module) RegisterRoutes(protected, public *gin.RouterGroup) {
	m.Handler.RegisterRoutes(protected, public)
}
