package catalogs

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module wires dependencies and routes for the catalogs vertical slice.
type Module struct {
	handler *Handler
}

// NewModule constructs a catalogs module with manual dependency wiring.
func NewModule(db *pgxpool.Pool) *Module {
	repository := NewRepository(db)
	service := NewService(repository)
	handler := NewHandler(service)

	return &Module{handler: handler}
}

// RegisterRoutes registers module routes in the provided router group.
func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.RegisterRoutes(r)
}
