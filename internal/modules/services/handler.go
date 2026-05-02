package services

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	defaultPopularLimit = 12
	defaultSearchLimit  = 10
)

type Handler struct {
	service ServicesService
}

func NewHandler(service ServicesService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/api/v1/services", h.listServices)
}

// listServices godoc
// @Summary      List services
// @Description  Returns popular active services when q is empty, or matching active services when q is provided. Results include metadata and honor the optional limit parameter.
// @Tags         Services
// @Produce      json
// @Param        q      query     string              false  "Search term"
// @Param        limit  query     int                 false  "Results limit"
// @Success      200    {object}  ListServicesResult  "List of services"
// @Failure      400    {object}  shared.ErrorResponse "Invalid query params"
// @Failure      500    {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/services [get]
func (h *Handler) listServices(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	limit, err := resolveLimit(c.Query("limit"), resolveDefaultLimit(query))
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validateListServicesRequest(limit); err != nil {
		shared.BadRequest(c, err)
		return
	}

	ctx := c.Request.Context()

	var result ListServicesResult
	if query == "" {
		result, err = h.service.ListPopularServices(ctx, ListPopularInput{Limit: int32(limit)})
	} else {
		result, err = h.service.SearchServices(ctx, SearchInput{Query: query, Limit: int32(limit)})
	}
	if err != nil {
		shared.InternalError(c, "could not list services")
		return
	}

	c.JSON(http.StatusOK, result)
}

func resolveDefaultLimit(query string) int {
	if query == "" {
		return defaultPopularLimit
	}
	return defaultSearchLimit
}

func resolveLimit(rawLimit string, fallback int) (int, error) {
	trimmed := strings.TrimSpace(rawLimit)
	if trimmed == "" {
		return fallback, nil
	}

	limit, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errors.New("limit must be a valid integer")
	}

	return limit, nil
}

func validateListServicesRequest(limit int) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: limit <= 0, Msg: "limit must be greater than 0"},
	})
}
