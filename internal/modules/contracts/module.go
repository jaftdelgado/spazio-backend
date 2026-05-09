package contracts

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/storage"
)

type Module struct {
	handler *Handler
}

func NewModule(db *pgxpool.Pool, r2Client *storage.R2Client) *Module {
	repository := NewRepository(db)
	service := NewService(repository, r2Client)
	handler := NewHandler(service)

	return &Module{handler: handler}
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.RegisterRoutes(r)
}
