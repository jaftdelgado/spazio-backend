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

// RegisterRoutes registers protected module routes in the provided router group.
func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.RegisterRoutes(r)
}

// RegisterPublicRoutes registers read-only property routes that can be used
// without an authenticated session. If OptionalAuth resolves a user, the
// handlers can still use that context to apply role-aware visibility.
func (m *Module) RegisterPublicRoutes(r *gin.RouterGroup) {
	properties := r.Group("/api/v1/properties")

	properties.GET("", m.handler.listProperties)
	properties.GET("/:uuid", m.handler.getProperty)
	properties.GET("/:uuid/photos", m.handler.getPhotos)
	properties.GET("/:uuid/services", m.handler.getServices)
	properties.GET("/:uuid/clauses", m.handler.getClauses)
	properties.GET("/:uuid/prices", m.handler.getPrices)
}
