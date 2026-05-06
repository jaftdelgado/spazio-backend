package properties

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/storage"
)

// Module wires dependencies and routes for the properties vertical slice.
type Module struct {
	handler *Handler
}

// NewModule constructs a properties module with manual dependency wiring.
func NewModule(db *pgxpool.Pool, r2Client *storage.R2Client) *Module {
	repository := NewRepository(db)
	service := NewService(repository, r2Client)
	handler := NewHandler(service)

	return &Module{handler: handler}
}

// RegisterRoutes registers module routes in the provided router group.
func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.RegisterRoutes(r)
}
