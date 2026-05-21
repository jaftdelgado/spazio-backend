package properties

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service PropertyService
}

func NewHandler(service PropertyService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	properties := r.Group("/api/v1/properties")

	adminOnly := properties.Group("")
	adminOnly.Use(middleware.RequireRole("admin"))
	{
		adminOnly.POST("", h.createProperty)
		adminOnly.PATCH("/:uuid", h.updateProperty)
		adminOnly.DELETE("/:uuid", h.deleteProperty)
		adminOnly.PUT("/:uuid/clauses", h.updateClauses)
		adminOnly.PUT("/:uuid/photos", h.updatePhotos)
		adminOnly.PUT("/:uuid/services", h.updateServices)
		adminOnly.PUT("/:uuid/prices", h.updatePrices)
		adminOnly.GET("/:uuid/history", h.getPropertyHistory)
		adminOnly.GET("/:uuid/prices/history", h.getPricesHistory)
	}

	adminOrAgent := properties.Group("")
	adminOrAgent.Use(middleware.RequireRole("admin", "agent"))
	{
		adminOrAgent.GET("", h.listProperties)
		adminOrAgent.GET("/:uuid", h.getProperty)
		adminOrAgent.GET("/:uuid/photos", h.getPhotos)
		adminOrAgent.GET("/:uuid/services", h.getServices)
		adminOrAgent.GET("/:uuid/clauses", h.getClauses)
		adminOrAgent.GET("/:uuid/prices", h.getPrices)
	}
}

// createProperty godoc
// @Summary      Register a new property
// @Description  Registers a property and all related records in a single database transaction. The backend generates the property UUID and stores subtype, location, pricing, services, and clauses atomically. The authenticated user is set as the owner.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      CreatePropertyInput   true  "Property payload"
// @Success      201      {object}  CreatePropertyResult  "Property created"
// @Failure      400      {object}  shared.ErrorResponse  "Invalid input"
// @Failure      500      {object}  shared.ErrorResponse  "Internal error"
// @Router       /api/v1/properties [post]
func (h *Handler) createProperty(c *gin.Context) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return
	}

	if err := rejectForbiddenPayloadFields(c, "category", "subtype", "owner_id"); err != nil {
		shared.BadRequest(c, err)
		return
	}

	var req CreatePropertyInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	req = sanitizeCreatePropertyInput(req)

	if err := validateCreatePropertyRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.CreateProperty(c.Request.Context(), userID, req)
	if err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		log.Printf("create property: %v", err)
		shared.InternalError(c, "could not create property")
		return
	}

	c.JSON(http.StatusCreated, result)
}

func sanitizeCreatePropertyInput(input CreatePropertyInput) CreatePropertyInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Location != nil {
		input.Location.Neighborhood = strings.TrimSpace(input.Location.Neighborhood)
		input.Location.Street = strings.TrimSpace(input.Location.Street)
		input.Location.ExteriorNumber = strings.TrimSpace(input.Location.ExteriorNumber)
		input.Location.InteriorNumber = trimOptionalString(input.Location.InteriorNumber)
		input.Location.PostalCode = strings.TrimSpace(input.Location.PostalCode)
	}

	if input.Commercial != nil {
		input.Commercial.LandUse = trimOptionalString(input.Commercial.LandUse)
	}

	if input.SalePrice != nil {
		input.SalePrice.Currency = strings.TrimSpace(input.SalePrice.Currency)
	}

	for i := range input.RentPrices {
		input.RentPrices[i].Currency = strings.TrimSpace(input.RentPrices[i].Currency)
	}

	return input
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func validateCreatePropertyRequest(req CreatePropertyInput) error {
	if err := shared.Validate([]shared.ValidationRule{
		{Fail: req.Title == "", Msg: "title is required"},
		{Fail: req.PropertyTypeID <= 0, Msg: "property_type_id must be greater than 0"},
		{Fail: req.ModalityID <= 0, Msg: "modality_id must be greater than 0"},
	}); err != nil {
		return err
	}

	if req.Location == nil {
		return errors.New("location is required")
	}

	if err := shared.Validate([]shared.ValidationRule{
		{Fail: req.Location.CityID <= 0, Msg: "location.city_id must be greater than 0"},
		{Fail: req.Location.Street == "", Msg: "location.street is required"},
		{Fail: req.Location.ExteriorNumber == "", Msg: "location.exterior_number is required"},
		{Fail: req.Location.Latitude == nil, Msg: "location.latitude is required"},
		{Fail: req.Location.Longitude == nil, Msg: "location.longitude is required"},
		{Fail: req.Location.IsPublicAddress == nil, Msg: "location.is_public_address is required"},
	}); err != nil {
		return err
	}

	if err := validateCoordinates(*req.Location.Latitude, *req.Location.Longitude); err != nil {
		return err
	}

	if err := validateCreateSubtypeDomain(req); err != nil {
		return err
	}

	if err := validateOptionalPrices(req); err != nil {
		return err
	}

	if err := validateCollections(req); err != nil {
		return err
	}

	return nil
}

func validateSubtypePayload(req CreatePropertyInput) error {
	switch req.Subtype {
	case SubtypeResidential:
		if req.Residential == nil {
			return errors.New("residential is required when subtype is residential")
		}
		if req.Commercial != nil {
			return errors.New("commercial must be omitted when subtype is residential")
		}

		return shared.Validate([]shared.ValidationRule{
			{Fail: req.Residential.Bedrooms == nil, Msg: "residential.bedrooms is required"},
			{Fail: req.Residential.Bathrooms == nil, Msg: "residential.bathrooms is required"},
			{Fail: req.Residential.Beds == nil, Msg: "residential.beds is required"},
			{Fail: req.Residential.Floors == nil, Msg: "residential.floors is required"},
			{Fail: req.Residential.ParkingSpots == nil, Msg: "residential.parking_spots is required"},
			{Fail: req.Residential.BuiltArea == nil, Msg: "residential.built_area is required"},
			{Fail: req.Residential.ConstructionYear == nil, Msg: "residential.construction_year is required"},
			{Fail: req.Residential.OrientationID == nil || *req.Residential.OrientationID <= 0, Msg: "residential.orientation_id must be greater than 0"},
			{Fail: req.Residential.IsFurnished == nil, Msg: "residential.is_furnished is required"},
		})
	case SubtypeCommercial:
		if req.Commercial == nil {
			return errors.New("commercial is required when subtype is commercial")
		}
		if req.Residential != nil {
			return errors.New("residential must be omitted when subtype is commercial")
		}

		return shared.Validate([]shared.ValidationRule{
			{Fail: req.Commercial.CeilingHeight == nil, Msg: "commercial.ceiling_height is required"},
			{Fail: req.Commercial.LoadingDocks == nil, Msg: "commercial.loading_docks is required"},
			{Fail: req.Commercial.InternalOffices == nil, Msg: "commercial.internal_offices is required"},
			{Fail: req.Commercial.ThreePhasePower == nil, Msg: "commercial.three_phase_power is required"},
			{Fail: req.Commercial.LandUse == nil || *req.Commercial.LandUse == "", Msg: "commercial.land_use is required"},
		})
	case SubtypeOther:
		if req.Residential != nil {
			return errors.New("residential must be omitted for the selected subtype")
		}
		if req.Commercial != nil {
			return errors.New("commercial must be omitted for the selected subtype")
		}
	default:
		return errors.New("subtype must be one of residential, commercial, or other")
	}

	return nil
}

func validateOptionalPrices(req CreatePropertyInput) error {
	if req.SalePrice != nil {
		if err := shared.Validate([]shared.ValidationRule{
			{Fail: req.SalePrice.SalePrice == nil, Msg: "sale_price.sale_price is required"},
			{Fail: req.SalePrice.Currency == "", Msg: "sale_price.currency is required"},
			{Fail: req.SalePrice.IsNegotiable == nil, Msg: "sale_price.is_negotiable is required"},
		}); err != nil {
			return err
		}
	}

	for i, rentPrice := range req.RentPrices {
		if err := shared.Validate([]shared.ValidationRule{
			{Fail: rentPrice.PeriodID <= 0, Msg: "rent_prices[" + indexString(i) + "].period_id must be greater than 0"},
			{Fail: rentPrice.RentPrice == nil, Msg: "rent_prices[" + indexString(i) + "].rent_price is required"},
			{Fail: rentPrice.Currency == "", Msg: "rent_prices[" + indexString(i) + "].currency is required"},
			{Fail: rentPrice.IsNegotiable == nil, Msg: "rent_prices[" + indexString(i) + "].is_negotiable is required"},
		}); err != nil {
			return err
		}

		if *rentPrice.RentPrice <= 0 {
			return errors.New("rent_prices[" + indexString(i) + "].rent_price must be greater than 0")
		}
		if rentPrice.Deposit != nil && *rentPrice.Deposit < 0 {
			return errors.New("rent_prices[" + indexString(i) + "].deposit must be greater than or equal to 0")
		}
	}

	return nil
}

func validateCreateSubtypeDomain(req CreatePropertyInput) error {
	if req.Residential != nil {
		if *req.Residential.Bedrooms < 0 || *req.Residential.Bathrooms < 0 || *req.Residential.Beds < 0 || *req.Residential.Floors < 0 || *req.Residential.ParkingSpots < 0 {
			return errors.New("residential numeric fields must be greater than or equal to 0")
		}
		if *req.Residential.BuiltArea <= 0 {
			return errors.New("residential.built_area must be greater than 0")
		}
	}

	if req.Commercial != nil {
		if *req.Commercial.CeilingHeight <= 0 {
			return errors.New("commercial.ceiling_height must be greater than 0")
		}
		if *req.Commercial.LoadingDocks < 0 || *req.Commercial.InternalOffices < 0 {
			return errors.New("commercial numeric fields must be greater than or equal to 0")
		}
	}

	return nil
}

func validateCollections(req CreatePropertyInput) error {
	if err := validateServiceIDs(req.Services); err != nil {
		return err
	}

	if err := validateClauseInputs(req.Clauses); err != nil {
		return err
	}

	return nil
}

func indexString(index int) string {
	return strconv.Itoa(index)
}

func resolveAuthenticatedActor(c *gin.Context) (int32, int32, bool) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, 0, false
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, 0, false
	}

	return userID, roleID, true
}

func attachActorContext(c *gin.Context, actor *ActorContext) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		return
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		return
	}

	actor.UserID = userID
	actor.RoleID = roleID
}

func rejectForbiddenPayloadFields(c *gin.Context, fields ...string) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	if len(body) == 0 {
		return nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	for _, field := range fields {
		if _, exists := payload[field]; exists {
			return errors.New(field + " is not allowed")
		}
	}

	return nil
}
