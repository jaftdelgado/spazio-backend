package catalogs

import (
	"log"
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
	catalogs := r.Group("/api/v1/catalogs")
	{
		catalogs.GET("/modalities", h.listModalities)
		catalogs.GET("/property-types", h.listPropertyTypes)
		catalogs.GET("/rent-periods", h.listRentPeriods)
		catalogs.GET("/orientations", h.listOrientations)
	}
}

// listModalities godoc
// @Summary      List modalities
// @Description  Returns all modalities ordered by modality_id ascending.
// @Tags         Catalogs
// @Produce      json
// @Success      200  {object}  ListModalitiesResult  "List of modalities"
// @Failure      500  {object}  shared.ErrorResponse  "Internal error"
// @Router       /api/v1/catalogs/modalities [get]
func (h *Handler) listModalities(c *gin.Context) {
	result, err := h.service.ListModalities(c.Request.Context())
	if err != nil {
		shared.InternalError(c, "could not list modalities")
		return
	}

	c.JSON(http.StatusOK, result)
}

// listPropertyTypes godoc
// @Summary      List property types
// @Description  Returns all non-deprecated property types ordered by property_type_id ascending.
// @Tags         Catalogs
// @Produce      json
// @Success      200  {object}  ListPropertyTypesResult  "List of property types"
// @Failure      500  {object}  shared.ErrorResponse     "Internal error"
// @Router       /api/v1/catalogs/property-types [get]
func (h *Handler) listPropertyTypes(c *gin.Context) {
	result, err := h.service.ListPropertyTypes(c.Request.Context())
	if err != nil {
		shared.InternalError(c, "could not list property types")
		return
	}

	c.JSON(http.StatusOK, result)
}

// listRentPeriods godoc
// @Summary      List rent periods
// @Description  Returns all rent periods ordered by period_id ascending.
// @Tags         Catalogs
// @Produce      json
// @Success      200  {object}  ListRentPeriodsResult  "List of rent periods"
// @Failure      500  {object}  shared.ErrorResponse   "Internal error"
// @Router       /api/v1/catalogs/rent-periods [get]
func (h *Handler) listRentPeriods(c *gin.Context) {
	result, err := h.service.ListRentPeriods(c.Request.Context())
	if err != nil {
		shared.InternalError(c, "could not list rent periods")
		return
	}

	c.JSON(http.StatusOK, result)
}

// listOrientations godoc
// @Summary      List orientations
// @Description  Returns all orientations ordered by name ascending.
// @Tags         Catalogs
// @Produce      json
// @Success      200  {object}  ListOrientationsResult  "List of orientations"
// @Failure      500  {object}  shared.ErrorResponse    "Internal error"
// @Router       /api/v1/catalogs/orientations [get]
func (h *Handler) listOrientations(c *gin.Context) {
	result, err := h.service.ListOrientations(c.Request.Context())
	if err != nil {
		log.Printf("list orientations: %v", err)
		shared.InternalError(c, "could not list orientations")
		return
	}

	c.JSON(http.StatusOK, result)
}
