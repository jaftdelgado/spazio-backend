package payments

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module wires dependencies and routes for the payments vertical slice.
type Module struct {
	Handler *Handler
}

// NewModule constructs a payments module with manual dependency wiring.
func NewModule(db *pgxpool.Pool) *Module {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	return &Module{
		Handler: handler,
	}
}

// RegisterRoutes registers module routes in the provided router group.
func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.Handler.RegisterRoutes(r)
}
