package properties

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	defaultPropertiesPage     = 1
	defaultPropertiesPageSize = 20
	maxPropertiesPageSize     = 100
)

// listProperties godoc
<<<<<<< HEAD
// @Summary      List properties
// @Description  Returns a paginated list of property cards with optional search, status, type, modality, and location filters. Deleted properties are always excluded. The selected card price prefers sale price unless the property modality is rent or no current sale price exists; in that case the best current rent price is used following monthly, annual, weekly, then daily priority.
// @Tags         Properties
// @Produce      json
// @Param        page              query     int                   false  "Page number" default(1)
// @Param        page_size         query     int                   false  "Results per page" default(20)
// @Param        q                 query     string                false  "Search term across title, street, neighborhood, city, state, and country"
// @Param        status_id         query     []int                 false  "Filter by property status. Repeat the parameter to send multiple values."
=======
// @Summary      List properties with advanced filters
// @Description  Returns a paginated list of property cards. Supports filtering by search query, status, property type, modality, location (country, state, city), price range, and minimum bedrooms. Price selection logic prioritizes sale price, then the best current rent price (Monthly > Annual > Weekly > Daily).
// @Tags         Properties
// @Produce      json
// @Param        page              query     int                   false  "Page number (starts at 1)" default(1)
// @Param        page_size         query     int                   false  "Items per page (max 100)" default(20)
// @Param        q                 query     string                false  "Search term (matches title, address, city, state, country)"
// @Param        status_id         query     []int                 false  "Filter by status IDs (Available=2 by default for clients)"
>>>>>>> origin/main
// @Param        property_type_id  query     int                   false  "Filter by property type ID"
// @Param        modality_id       query     int                   false  "Filter by modality ID"
// @Param        country_id        query     int                   false  "Filter by country ID"
// @Param        state_id          query     int                   false  "Filter by state ID"
// @Param        city_id           query     int                   false  "Filter by city ID"
<<<<<<< HEAD
// @Param        sort              query     string                false  "Sort field: created_at, title, or price"
// @Param        order             query     string                false  "Sort order: asc or desc"
// @Success      200               {object}  ListPropertiesResult  "Paginated property cards"
// @Failure      400               {object}  shared.ErrorResponse  "Invalid query params"
// @Failure      500               {object}  shared.ErrorResponse  "Internal error"
=======
// @Param        min_price         query     number                false  "Minimum price filter"
// @Param        max_price         query     number                false  "Maximum price filter"
// @Param        min_bedrooms      query     int                   false  "Minimum bedrooms filter (Residential only)"
// @Param        sort              query     string                false  "Sort by: created_at, title, price"
// @Param        order             query     string                false  "Sort order: asc, desc"
// @Success      200               {object}  ListPropertiesResult  "Paginated list of property cards"
// @Failure      400               {object}  shared.ErrorResponse  "Invalid input parameters"
// @Failure      500               {object}  shared.ErrorResponse  "Internal server error"
>>>>>>> origin/main
// @Router       /api/v1/properties [get]
func (h *Handler) listProperties(c *gin.Context) {
	page, err := resolveOptionalPropertyInt(strings.TrimSpace(c.Query("page")), defaultPropertiesPage, "page")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	pageSize, err := resolveOptionalPropertyInt(strings.TrimSpace(c.Query("page_size")), defaultPropertiesPageSize, "page_size")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	statusIDs, err := resolveStatusIDs(c.QueryArray("status_id"))
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	propertyTypeID, err := resolveOptionalPositivePropertyInt(strings.TrimSpace(c.Query("property_type_id")), "property_type_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	modalityID, err := resolveOptionalPositivePropertyInt(strings.TrimSpace(c.Query("modality_id")), "modality_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	countryID, err := resolveOptionalPositivePropertyInt(strings.TrimSpace(c.Query("country_id")), "country_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	stateID, err := resolveOptionalPositivePropertyInt(strings.TrimSpace(c.Query("state_id")), "state_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	cityID, err := resolveOptionalPositivePropertyInt(strings.TrimSpace(c.Query("city_id")), "city_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

<<<<<<< HEAD
=======
	minPrice, err := resolveOptionalFloat64Query(strings.TrimSpace(c.Query("min_price")), 0, "min_price")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	maxPrice, err := resolveOptionalFloat64Query(strings.TrimSpace(c.Query("max_price")), 0, "max_price")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	minBedrooms, err := resolveOptionalPositivePropertyInt(strings.TrimSpace(c.Query("min_bedrooms")), "min_bedrooms")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

>>>>>>> origin/main
	sortField := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	sortOrder := strings.ToLower(strings.TrimSpace(c.Query("order")))

	if err := validateListPropertiesRequest(page, pageSize, sortField, sortOrder); err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.ListProperties(c.Request.Context(), ListPropertiesInput{
		Page:           int32(page),
		PageSize:       int32(pageSize),
		Query:          strings.TrimSpace(c.Query("q")),
		StatusIDs:      statusIDs,
		PropertyTypeID: int32(propertyTypeID),
		ModalityID:     int32(modalityID),
		CountryID:      int32(countryID),
		StateID:        int32(stateID),
		CityID:         int32(cityID),
<<<<<<< HEAD
=======
		MinPrice:       minPrice,
		MaxPrice:       maxPrice,
		MinBedrooms:    int32(minBedrooms),
>>>>>>> origin/main
		Sort:           sortField,
		Order:          resolvePropertySortOrder(sortOrder),
	})
	if err != nil {
		log.Printf("list properties: %v", err)
		shared.InternalError(c, "could not list properties")
		return
	}

	c.JSON(http.StatusOK, result)
}

<<<<<<< HEAD
=======
// getPropertyHistory godoc
// @Summary      Get property status history
// @Description  Returns the chronological history of status changes for a specific property. Administrators can view any property history, while Agents or Clients can only view history of properties they own.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        uuid       path      string                true  "Property UUID"
// @Param        X-User-ID  header    string                true  "User ID for RBAC validation"
// @Param        X-Role-ID  header    string                true  "Role ID (1: Admin, 2: Agent, 3: Client)"
// @Success      200        {object}  GetPropertyHistoryResult  "Chronological status history retrieved successfully"
// @Failure      400        {object}  shared.ErrorResponse      "Invalid UUID or missing required headers"
// @Failure      403        {object}  shared.ErrorResponse      "Forbidden: You are not authorized to view this history"
// @Failure      404        {object}  shared.ErrorResponse      "Property not found"
// @Failure      500        {object}  shared.ErrorResponse      "Internal server error"
// @Router       /api/v1/properties/{uuid}/history [get]
func (h *Handler) getPropertyHistory(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	userID, err := strconv.Atoi(c.GetHeader("X-User-ID"))
	if err != nil {
		shared.BadRequest(c, errors.New("invalid X-User-ID header"))
		return
	}

	roleID, err := strconv.Atoi(c.GetHeader("X-Role-ID"))
	if err != nil {
		shared.BadRequest(c, errors.New("invalid X-Role-ID header"))
		return
	}

	result, err := h.service.GetPropertyHistory(c.Request.Context(), propertyUUID, int32(userID), int32(roleID))
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "forbidden") {
			shared.Forbidden(c, err.Error())
			return
		}

		log.Printf("get property history: %v", err)
		shared.InternalError(c, "could not get property history")
		return
	}

	c.JSON(http.StatusOK, result)
}

>>>>>>> origin/main
// getProperty godoc
// @Summary      Get property by UUID
// @Description  Returns property base data, subtype, and location for the given UUID. When full=true, the response also includes consolidated prices, price history, photos, services, and clauses. Deleted properties are treated as not found.
// @Tags         Properties
// @Produce      json
// @Param        uuid  path      string                true   "Property UUID"
// @Param        full  query     bool                  false  "Include prices, history, photos, services, and clauses"
// @Success      200   {object}  GetPropertyResult     "Property data"
// @Failure      400   {object}  shared.ErrorResponse  "Invalid path parameter"
// @Failure      404   {object}  shared.ErrorResponse  "Property not found"
// @Failure      500   {object}  shared.ErrorResponse  "Internal error"
// @Router       /api/v1/properties/{uuid} [get]
func (h *Handler) getProperty(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	full, err := resolveOptionalBoolQuery(strings.TrimSpace(c.Query("full")), false, "full")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	if full {
		result, err := h.service.GetFullProperty(c.Request.Context(), propertyUUID)
		if err != nil {
			if errors.Is(err, ErrPropertyNotFound) {
				shared.NotFound(c, err.Error())
				return
			}

			log.Printf("get full property: %v", err)
			shared.InternalError(c, "could not get property")
			return
		}

		c.JSON(http.StatusOK, result)
		return
	}

	result, err := h.service.GetProperty(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property: %v", err)
		shared.InternalError(c, "could not get property")
		return
	}

	c.JSON(http.StatusOK, result)
}

func resolveOptionalPropertyInt(rawValue string, fallback int, field string) (int, error) {
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, errors.New(field + " must be a valid integer")
	}

	return value, nil
}

func resolveOptionalPositivePropertyInt(rawValue string, field string) (int, error) {
	if rawValue == "" {
		return 0, nil
	}

	value, err := resolveOptionalPropertyInt(rawValue, 0, field)
	if err != nil {
		return 0, err
	}

	if value <= 0 {
		return 0, errors.New(field + " must be greater than 0")
	}

	return value, nil
}

func resolveOptionalBoolQuery(rawValue string, fallback bool, field string) (bool, error) {
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false, errors.New(field + " must be a valid boolean")
	}

	return value, nil
}

func resolveStatusIDs(rawValues []string) ([]int32, error) {
	statusIDs := make([]int32, 0, len(rawValues))
	for _, rawValue := range rawValues {
		trimmed := strings.TrimSpace(rawValue)
		if trimmed == "" {
			return nil, errors.New("status_id must be a valid integer")
		}

		value, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, errors.New("status_id must be a valid integer")
		}
		if value <= 0 {
			return nil, errors.New("status_id must be greater than 0")
		}

		statusIDs = append(statusIDs, int32(value))
	}

	return statusIDs, nil
}

func validateListPropertiesRequest(page, pageSize int, sortField, sortOrder string) error {
	if err := shared.Validate([]shared.ValidationRule{
		{Fail: page <= 0, Msg: "page must be greater than 0"},
		{Fail: pageSize <= 0, Msg: "page_size must be greater than 0"},
		{Fail: pageSize > maxPropertiesPageSize, Msg: "page_size must be less than or equal to 100"},
	}); err != nil {
		return err
	}

	if sortField != "" && sortField != "created_at" && sortField != "title" && sortField != "price" {
		return errors.New("sort must be one of created_at, title, or price")
	}

	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return errors.New("order must be asc or desc")
	}

	return nil
}

func resolvePropertySortOrder(sortOrder string) string {
	if sortOrder == "" {
		return "desc"
	}

	return sortOrder
}
<<<<<<< HEAD
=======

func resolveOptionalFloat64Query(rawValue string, fallback float64, field string) (float64, error) {
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		return 0, errors.New(field + " must be a valid number")
	}

	return value, nil
}
>>>>>>> origin/main
