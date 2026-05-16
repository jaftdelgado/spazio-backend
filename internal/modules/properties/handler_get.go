package properties

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	defaultPropertiesPage     = 1
	defaultPropertiesPageSize = 20
	maxPropertiesPageSize     = 100
)

// listProperties godoc
// @Summary      List properties with advanced filters
// @Description  Returns a paginated list of property cards. Administrators can view non-deleted properties, while agents can only view properties assigned in property_agents. Supports filtering by search query, status, property type, modality, location (country, state, city), price range, and minimum bedrooms. Price selection logic prioritizes sale price, then the best current rent price (Monthly > Annual > Weekly > Daily).
// @Tags         Properties
// @Produce      json
// @Security     BearerAuth
// @Param        page              query     int                   false  "Page number (starts at 1)" default(1)
// @Param        page_size         query     int                   false  "Items per page (max 100)" default(20)
// @Param        q                 query     string                false  "Search term (matches title, address, city, state, country)"
// @Param        status_id         query     []int                 false  "Filter by status IDs (defaults to Available=2 when omitted)"
// @Param        property_type_id  query     int                   false  "Filter by property type ID"
// @Param        modality_id       query     int                   false  "Filter by modality ID"
// @Param        country_id        query     int                   false  "Filter by country ID"
// @Param        state_id          query     int                   false  "Filter by state ID"
// @Param        city_id           query     int                   false  "Filter by city ID"
// @Param        min_price         query     number                false  "Minimum price filter"
// @Param        max_price         query     number                false  "Maximum price filter"
// @Param        min_bedrooms      query     int                   false  "Minimum bedrooms filter (Residential only)"
// @Param        sort              query     string                false  "Sort by: created_at, title, price"
// @Param        order             query     string                false  "Sort order: asc, desc"
// @Success      200               {object}  ListPropertiesResult  "Paginated list of property cards"
// @Failure      400               {object}  shared.ErrorResponse  "Invalid input parameters"
// @Failure      401               {object}  shared.ErrorResponse  "Missing or invalid authenticated session"
// @Failure      403               {object}  shared.ErrorResponse  "Forbidden"
// @Failure      500               {object}  shared.ErrorResponse  "Internal server error"
// @Router       /api/v1/properties [get]
func (h *Handler) listProperties(c *gin.Context) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return
	}

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
		MinPrice:       minPrice,
		MaxPrice:       maxPrice,
		MinBedrooms:    int32(minBedrooms),
		UserID:         userID,
		RoleID:         roleID,
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

// getPropertyHistory godoc
// @Summary      Get property status history
// @Description  Returns the chronological history of status changes for a specific property. Requires an authenticated admin session.
// @Tags         Properties
// @Produce      json
// @Security     BearerAuth
// @Param        uuid       path      string                    true  "Property UUID"
// @Success      200        {object}  GetPropertyHistoryResult  "Chronological status history retrieved successfully"
// @Failure      400        {object}  shared.ErrorResponse      "Invalid UUID"
// @Failure      401        {object}  shared.ErrorResponse      "Missing or invalid authenticated session"
// @Failure      403        {object}  shared.ErrorResponse      "Forbidden"
// @Failure      404        {object}  shared.ErrorResponse      "Property not found"
// @Failure      500        {object}  shared.ErrorResponse      "Internal server error"
// @Router       /api/v1/properties/{uuid}/history [get]
func (h *Handler) getPropertyHistory(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	result, err := h.service.GetPropertyHistory(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property history: %v", err)
		shared.InternalError(c, "could not get property history")
		return
	}

	c.JSON(http.StatusOK, result)
}

// getProperty godoc
// @Summary      Get property by UUID
// @Description  Returns property base data, subtype, and location for the given UUID. Administrators can view all fields including registered_by. Agents can only access assigned properties and do not receive registered_by.
// @Tags         Properties
// @Produce      json
// @Security     BearerAuth
// @Param        uuid  path      string                true   "Property UUID"
// @Success      200   {object}  GetPropertyResult     "Property data"
// @Failure      400   {object}  shared.ErrorResponse  "Invalid path parameter"
// @Failure      401   {object}  shared.ErrorResponse  "Missing or invalid authenticated session"
// @Failure      403   {object}  shared.ErrorResponse  "Forbidden"
// @Failure      404   {object}  shared.ErrorResponse  "Property not found"
// @Failure      500   {object}  shared.ErrorResponse  "Internal error"
// @Router       /api/v1/properties/{uuid} [get]
func (h *Handler) getProperty(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return
	}

	result, err := h.service.GetPropertyForRole(c.Request.Context(), propertyUUID, userID, roleID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "forbidden") {
			shared.Forbidden(c, err.Error())
			return
		}

		log.Printf("get property: %v", err)
		shared.InternalError(c, "could not get property")
		return
	}

	c.JSON(http.StatusOK, result)
}

// getPricesHistory godoc
// @Summary      Get property prices history
// @Description  Returns the complete price history for a property. Requires admin role.
// @Tags         Properties
// @Produce      json
// @Security     BearerAuth
// @Param        uuid  path      string                         true  "Property UUID"
// @Success      200   {object}  GetPropertyPricesHistoryResult "Property prices history"
// @Failure      400   {object}  shared.ErrorResponse           "Invalid path parameter"
// @Failure      401   {object}  shared.ErrorResponse           "Missing or invalid authenticated session"
// @Failure      403   {object}  shared.ErrorResponse           "Forbidden"
// @Failure      404   {object}  shared.ErrorResponse           "Property not found"
// @Failure      500   {object}  shared.ErrorResponse           "Internal error"
// @Router       /api/v1/properties/{uuid}/prices/history [get]
func (h *Handler) getPricesHistory(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	result, err := h.service.GetPricesHistory(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property prices history: %v", err)
		shared.InternalError(c, "could not get property prices history")
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
