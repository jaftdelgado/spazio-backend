package catalogs

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service CatalogsService
}

func NewHandler(service CatalogsService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/catalogs/modalities", h.listModalities)
	r.GET("/catalogs/property-types", h.listPropertyTypes)
}

func (h *Handler) listModalities(c *gin.Context) {
	result, err := h.service.ListModalities(c.Request.Context())
	if err != nil {
		shared.InternalError(c, "could not list modalities")
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) listPropertyTypes(c *gin.Context) {
	result, err := h.service.ListPropertyTypes(c.Request.Context())
	if err != nil {
		shared.InternalError(c, "could not list property types")
		return
	}

	c.JSON(http.StatusOK, result)
}
