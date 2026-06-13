package sales

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	handler *Handler
}

func NewModule(db *pgxpool.Pool, port string) *Module {
	repo := NewRepository(db)
	contractsClient := NewHTTPContractsClient(fmt.Sprintf("http://localhost:%s", port))
	service := NewService(repo, contractsClient)
	handler := NewHandler(service)

	return &Module{handler: handler}
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.RegisterRoutes(r)
}
