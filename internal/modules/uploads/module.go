package uploads

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

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	uploads := router.Group("/api/v1/uploads")
	{
		uploads.POST("/properties/:property_uuid/photos", m.handler.uploadPropertyPhoto)
	}
}
