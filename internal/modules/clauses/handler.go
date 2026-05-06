package clauses

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 50
)

type Handler struct {
	service ClausesService
}

func NewHandler(service ClausesService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/api/v1/clauses", h.listClauses)
}

// listClauses godoc
// @Summary      List clauses
// @Description  Returns clauses available for the provided modality. When q is present, the endpoint performs a filtered search. Results are paginated.
// @Tags         Clauses
// @Produce      json
// @Param        modality_id  query     int                true   "Modality ID"
// @Param        q            query     string             false  "Search term"
// @Param        page         query     int                false  "Page number" default(1)
// @Param        page_size    query     int                false  "Results per page" default(20)
// @Success      200          {object}  ListClausesResult  "List of clauses"
// @Failure      400          {object}  shared.ErrorResponse "Invalid query params"
// @Failure      500          {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/clauses [get]
func (h *Handler) listClauses(c *gin.Context) {
	rawModalityID := strings.TrimSpace(c.Query("modality_id"))
	query := strings.TrimSpace(c.Query("q"))
	rawPage := strings.TrimSpace(c.Query("page"))
	rawPageSize := strings.TrimSpace(c.Query("page_size"))

	modalityID, err := resolveRequiredInt(rawModalityID, "modality_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	page, err := resolveOptionalInt(rawPage, defaultPage, "page")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	pageSize, err := resolveOptionalInt(rawPageSize, defaultPageSize, "page_size")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validateListClausesRequest(modalityID, page, pageSize); err != nil {
		shared.BadRequest(c, err)
		return
	}

	ctx := c.Request.Context()

	var result ListClausesResult
	if query == "" {
		result, err = h.service.ListClauses(ctx, ListClausesInput{
			ModalityID: int32(modalityID),
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
	} else {
		result, err = h.service.SearchClauses(ctx, SearchClausesInput{
			ModalityID: int32(modalityID),
			Query:      query,
			Page:       int32(page),
			PageSize:   int32(pageSize),
		})
	}
	if err != nil {
		shared.InternalError(c, "could not list clauses")
		return
	}

	c.JSON(http.StatusOK, result)
}

func resolveRequiredInt(rawValue string, field string) (int, error) {
	if rawValue == "" {
		return 0, errors.New(field + " is required")
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, errors.New(field + " must be a valid integer")
	}

	return value, nil
}

func resolveOptionalInt(rawValue string, fallback int, field string) (int, error) {
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, errors.New(field + " must be a valid integer")
	}

	return value, nil
}

func validateListClausesRequest(modalityID, page, pageSize int) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: modalityID <= 0, Msg: "modality_id must be greater than 0"},
		{Fail: page <= 0, Msg: "page must be greater than 0"},
		{Fail: pageSize <= 0, Msg: "page_size must be greater than 0"},
		{Fail: pageSize > maxPageSize, Msg: "page_size must be less than or equal to 50"},
	})
}
