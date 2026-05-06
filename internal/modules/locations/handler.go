package locations

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultCitiesPage     = 1
	defaultCitiesPageSize = 50
	maxCitiesPageSize     = 100
)

type Handler struct {
	service LocationsService
}

func NewHandler(service LocationsService) *Handler {
	return &Handler{
		service: service,
	}
}

// listCountries godoc
// @Summary      List countries
// @Description  Returns all available countries ordered by name.
// @Tags         Locations
// @Produce      json
// @Success      200  {object}  ListCountriesResult  "List of countries"
// @Failure      500  {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/locations/countries [get]
func (h *Handler) listCountries(c *gin.Context) {
	ctx := c.Request.Context()

	result, err := h.service.ListCountries(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list countries"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// listStates godoc
// @Summary      List states
// @Description  Returns all states for the selected country ordered by name.
// @Tags         Locations
// @Produce      json
// @Param        country_id  query     int  true  "Country ID"
// @Success      200         {object}  ListStatesResult    "List of states"
// @Failure      400         {object}  shared.ErrorResponse "Invalid query params"
// @Failure      500         {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/locations/states [get]
func (h *Handler) listStates(c *gin.Context) {
	countryIDStr := strings.TrimSpace(c.Query("country_id"))
	if countryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country_id is required"})
		return
	}

	countryID, err := strconv.ParseInt(countryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country_id must be a valid integer"})
		return
	}

	ctx := c.Request.Context()
	result, err := h.service.ListStates(ctx, ListStatesInput{
		CountryID: int32(countryID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list states"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// listCities godoc
// @Summary      List cities
// @Description  Returns cities for the selected state. Results are paginated.
// @Tags         Locations
// @Produce      json
// @Param        state_id   query     int  true   "State ID"
// @Param        page       query     int  false  "Page number" default(1)
// @Param        page_size  query     int  false  "Results per page" default(50)
// @Success      200        {object}  ListCitiesResult     "List of cities"
// @Failure      400        {object}  shared.ErrorResponse "Invalid query params"
// @Failure      500        {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/locations/cities [get]
func (h *Handler) listCities(c *gin.Context) {
	stateIDStr := strings.TrimSpace(c.Query("state_id"))
	if stateIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state_id is required"})
		return
	}

	stateID, err := strconv.ParseInt(stateIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state_id must be a valid integer"})
		return
	}

	pageStr := strings.TrimSpace(c.Query("page"))
	page, err := resolvePage(pageStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pageSizeStr := strings.TrimSpace(c.Query("page_size"))
	pageSize, err := resolvePageSize(pageSizeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	result, err := h.service.ListCities(ctx, ListCitiesInput{
		StateID:  int32(stateID),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list cities"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func resolvePage(pageValue string) (int, error) {
	if pageValue == "" {
		return defaultCitiesPage, nil
	}

	page, err := strconv.Atoi(pageValue)
	if err != nil {
		return 0, errInvalidPage()
	}

	if page < 1 {
		return 0, errInvalidPage()
	}

	return page, nil
}

func resolvePageSize(pageSizeValue string) (int, error) {
	if pageSizeValue == "" {
		return defaultCitiesPageSize, nil
	}

	pageSize, err := strconv.Atoi(pageSizeValue)
	if err != nil {
		return 0, errInvalidPageSize()
	}

	if pageSize < 1 || pageSize > maxCitiesPageSize {
		return 0, errInvalidPageSize()
	}

	return pageSize, nil
}

func errInvalidPage() error {
	return errors.New("page must be an integer greater than 0")
}

func errInvalidPageSize() error {
	return errors.New("page_size must be an integer between 1 and 100")
}
