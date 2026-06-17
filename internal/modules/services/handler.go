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
	maxListLimit        = 50
	defaultPage         = 1
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
// @Description  Returns popular active services when q is empty, or matching active services when q is provided. Search matches the service code plus Spanish and English synonyms stored in multilingual search_tags. Results can be filtered by category_id and paginated with page/page_size. The legacy limit parameter is still supported as a shorthand for page=1&page_size=limit.
// @Tags         Services
// @Produce      json
// @Param        q            query     string              false  "Search term matched against service code and multilingual search_tags (es/en)"
// @Param        category_id  query     int                 false  "Service category ID"
// @Param        page         query     int                 false  "Page number" default(1)
// @Param        page_size    query     int                 false  "Results per page"
// @Param        limit        query     int                 false  "Legacy alias for page_size when page/page_size are omitted"
// @Success      200    {object}  ListServicesResult  "List of services"
// @Failure      400    {object}  shared.ErrorResponse "Invalid query params"
// @Failure      500    {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/services [get]
func (h *Handler) listServices(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	categoryID, err := resolveCategoryID(c.Query("category_id"))
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	page, pageSize, err := resolvePagination(c.Query("page"), c.Query("page_size"), c.Query("limit"), resolveDefaultLimit(query))
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validateListServicesRequest(page, pageSize, categoryID); err != nil {
		shared.BadRequest(c, err)
		return
	}

	ctx := c.Request.Context()

	var result ListServicesResult
	if query == "" {
		result, err = h.service.ListPopularServices(ctx, ListPopularInput{
			CategoryID: categoryID,
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
	} else {
		result, err = h.service.SearchServices(ctx, SearchInput{
			Query:      query,
			CategoryID: categoryID,
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
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

func resolvePagination(rawPage string, rawPageSize string, rawLimit string, fallbackPageSize int) (int, int, error) {
	pageTrimmed := strings.TrimSpace(rawPage)
	pageSizeTrimmed := strings.TrimSpace(rawPageSize)

	page := defaultPage
	if pageTrimmed != "" {
		resolvedPage, err := strconv.Atoi(pageTrimmed)
		if err != nil {
			return 0, 0, errors.New("page must be a valid integer")
		}
		page = resolvedPage
	}

	pageSize := fallbackPageSize
	if pageSizeTrimmed != "" {
		resolvedPageSize, err := strconv.Atoi(pageSizeTrimmed)
		if err != nil {
			return 0, 0, errors.New("page_size must be a valid integer")
		}
		pageSize = resolvedPageSize
	} else if pageTrimmed == "" {
		limitTrimmed := strings.TrimSpace(rawLimit)
		legacyLimit, err := resolveLimit(rawLimit)
		if err != nil {
			return 0, 0, err
		}
		if limitTrimmed != "" {
			pageSize = legacyLimit
		}
	}

	return page, pageSize, nil
}

func resolveLimit(rawLimit string) (int, error) {
	trimmed := strings.TrimSpace(rawLimit)
	if trimmed == "" {
		return 0, nil
	}

	limit, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errors.New("limit must be a valid integer")
	}

	return limit, nil
}

func resolveCategoryID(rawCategoryID string) (int32, error) {
	trimmed := strings.TrimSpace(rawCategoryID)
	if trimmed == "" {
		return 0, nil
	}

	categoryID, err := strconv.ParseInt(trimmed, 10, 32)
	if err != nil {
		return 0, errors.New("category_id must be a valid integer")
	}

	return int32(categoryID), nil
}

func validateListServicesRequest(page int, pageSize int, categoryID int32) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: page <= 0, Msg: "page must be greater than 0"},
		{Fail: pageSize <= 0, Msg: "page_size must be greater than 0"},
		{Fail: pageSize > maxListLimit, Msg: "page_size must be less than or equal to 50"},
		{Fail: categoryID < 0, Msg: "category_id must be greater than or equal to 0"},
	})
}
