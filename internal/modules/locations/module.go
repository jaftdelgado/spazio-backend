package locations

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	handler *Handler
}

func NewModule(db *pgxpool.Pool) *Module {
	repository := NewRepository(db)
	service := NewService(repository)
	handler := NewHandler(service)
	return &Module{handler: handler}
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	locations := router.Group("/api/v1/locations")
	{
		locations.GET("/countries", m.handler.listCountries)
		locations.GET("/states", m.handler.listStates)
		locations.GET("/cities", m.handler.listCities)
	}
}
