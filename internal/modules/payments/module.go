package payments

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Handler *Handler
}

func NewModule(db *pgxpool.Pool) *Module {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	return &Module{
		Handler: handler,
	}
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.Handler.RegisterRoutes(r)
}
