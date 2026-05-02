package properties

import (
	"context"
	"errors"
)

const (
	CategoryResidential = "residential"
	CategoryCommercial  = "commercial"
	CategoryLand        = "land"
	CategoryOther       = "other"

	ClauseValueTypeBoolean = 1
	ClauseValueTypeRange   = 2
	ClauseValueTypeInteger = 3
)

// CreatePropertyInput is the request payload required to register a property.
type CreatePropertyInput struct {
	OwnerID        int32                       `json:"owner_id" example:"1"`
	Category       string                      `json:"category" example:"residential"`
	Title          string                      `json:"title" example:"Casa en Xalapa"`
	Description    string                      `json:"description" example:"Spacious residential property near downtown"`
	PropertyTypeID int32                       `json:"property_type_id" example:"1"`
	ModalityID     int32                       `json:"modality_id" example:"1"`
	LotArea        float64                     `json:"lot_area" example:"200"`
	IsFeatured     bool                        `json:"is_featured" example:"false"`
	Residential    *CreateResidentialInput     `json:"residential,omitempty"`
	Commercial     *CreateCommercialInput      `json:"commercial,omitempty"`
	Location       *CreateLocationInput        `json:"location"`
	SalePrice      *CreateSalePriceInput       `json:"sale_price,omitempty"`
	RentPrices     []CreateRentPriceInput      `json:"rent_prices,omitempty"`
	Services       []int32                     `json:"services,omitempty" example:"1,3,7"`
	Clauses        []CreatePropertyClauseInput `json:"clauses,omitempty"`
}

// CreateResidentialInput contains residential details for a property.
type CreateResidentialInput struct {
	Bedrooms         *int16   `json:"bedrooms" example:"3"`
	Bathrooms        *int16   `json:"bathrooms" example:"2"`
	Beds             *int16   `json:"beds" example:"4"`
	Floors           *int16   `json:"floors" example:"2"`
	ParkingSpots     *int16   `json:"parking_spots" example:"1"`
	BuiltArea        *float64 `json:"built_area" example:"120"`
	ConstructionYear *int16   `json:"construction_year" example:"2010"`
	OrientationID    *int32   `json:"orientation_id" example:"2"`
	IsFurnished      *bool    `json:"is_furnished" example:"false"`
}

// CreateCommercialInput contains commercial details for a property.
type CreateCommercialInput struct {
	CeilingHeight   *float64 `json:"ceiling_height" example:"4.5"`
	LoadingDocks    *int16   `json:"loading_docks" example:"1"`
	InternalOffices *int16   `json:"internal_offices" example:"2"`
	ThreePhasePower *bool    `json:"three_phase_power" example:"true"`
	LandUse         *string  `json:"land_use" example:"Retail"`
}

// CreateLocationInput contains the address and coordinates for a property.
type CreateLocationInput struct {
	CityID          int32    `json:"city_id" example:"1"`
	Neighborhood    string   `json:"neighborhood" example:"Centro"`
	Street          string   `json:"street" example:"Av. Principal"`
	ExteriorNumber  string   `json:"exterior_number" example:"45"`
	InteriorNumber  *string  `json:"interior_number,omitempty" example:"A"`
	PostalCode      string   `json:"postal_code" example:"91000"`
	Latitude        *float64 `json:"latitude" example:"19.5438"`
	Longitude       *float64 `json:"longitude" example:"-96.9102"`
	IsPublicAddress *bool    `json:"is_public_address" example:"true"`
}

// CreateSalePriceInput contains the sale pricing details.
type CreateSalePriceInput struct {
	SalePrice    *float64 `json:"sale_price" example:"1500000"`
	Currency     string   `json:"currency" example:"MXN"`
	IsNegotiable *bool    `json:"is_negotiable" example:"true"`
}

// CreateRentPriceInput contains a rent price entry for a period.
type CreateRentPriceInput struct {
	PeriodID     int32    `json:"period_id" example:"3"`
	RentPrice    *float64 `json:"rent_price" example:"8000"`
	Deposit      *float64 `json:"deposit,omitempty" example:"16000"`
	Currency     string   `json:"currency" example:"MXN"`
	IsNegotiable *bool    `json:"is_negotiable" example:"false"`
}

// CreatePropertyClauseInput contains a clause selection and its value payload.
type CreatePropertyClauseInput struct {
	ClauseID     int32    `json:"clause_id" example:"1"`
	BooleanValue *bool    `json:"boolean_value,omitempty" example:"true"`
	IntegerValue *int32   `json:"integer_value,omitempty" example:"2"`
	MinValue     *float64 `json:"min_value,omitempty" example:"1"`
	MaxValue     *float64 `json:"max_value,omitempty" example:"3"`
}

// GetPropertyClausesResult is the response returned by the property clauses list endpoint.
type GetPropertyClausesResult struct {
	Data []PropertyClauseData `json:"data"`
}

// PropertyClauseData represents a linked clause with its stored value payload.
type PropertyClauseData struct {
	ClauseID     int32    `json:"clause_id" example:"1"`
	BooleanValue *bool    `json:"boolean_value,omitempty" example:"true"`
	IntegerValue *int32   `json:"integer_value,omitempty" example:"2"`
	MinValue     *float64 `json:"min_value,omitempty" example:"1"`
	MaxValue     *float64 `json:"max_value,omitempty" example:"3"`
}

// UpdatePropertyClausesInput is the request payload used to replace the linked clauses of a property.
type UpdatePropertyClausesInput struct {
	Clauses []CreatePropertyClauseInput `json:"clauses,omitempty"`
}

// GetPropertyServicesResult is the response returned by the property services list endpoint.
type GetPropertyServicesResult struct {
	Data GetPropertyServicesData `json:"data"`
}

// GetPropertyServicesData contains the linked service identifiers.
type GetPropertyServicesData struct {
	ServiceIDs []int32 `json:"service_ids" example:"1,3,7"`
}

// UpdatePropertyServicesInput is the request payload used to replace the linked services of a property.
type UpdatePropertyServicesInput struct {
	ServiceIDs []int32 `json:"service_ids"`
}

// GetPropertyPricesResult is the response returned by the property prices list endpoint.
type GetPropertyPricesResult struct {
	Data GetPropertyPricesData `json:"data"`
}

// GetPropertyPricesData contains the active prices of the property.
type GetPropertyPricesData struct {
	SalePrice  *ActiveSalePriceData  `json:"sale_price"`
	RentPrices []ActiveRentPriceData `json:"rent_prices"`
}

// ActiveSalePriceData represents the active sale price.
type ActiveSalePriceData struct {
	SalePrice    float64 `json:"sale_price" example:"1500000"`
	Currency     string  `json:"currency" example:"MXN"`
	IsNegotiable bool    `json:"is_negotiable" example:"true"`
}

// ActiveRentPriceData represents an active rent price.
type ActiveRentPriceData struct {
	PeriodID     int32    `json:"period_id" example:"3"`
	RentPrice    float64  `json:"rent_price" example:"8000"`
	Deposit      *float64 `json:"deposit" example:"16000"`
	Currency     string   `json:"currency" example:"MXN"`
	IsNegotiable bool     `json:"is_negotiable" example:"false"`
}

// UpdatePropertyPricesInput is the request payload used to update property prices.
type UpdatePropertyPricesInput struct {
	SalePrice  *UpdateSalePriceInput  `json:"sale_price,omitempty"`
	RentPrices []UpdateRentPriceInput `json:"rent_prices,omitempty"`
}

// UpdateSalePriceInput contains the editable fields of a sale price.
type UpdateSalePriceInput struct {
	SalePrice    float64 `json:"sale_price" example:"1500000"`
	IsNegotiable bool    `json:"is_negotiable" example:"true"`
}

// UpdateRentPriceInput contains the editable fields of a rent price.
type UpdateRentPriceInput struct {
	PeriodID     int32    `json:"period_id" example:"3"`
	RentPrice    float64  `json:"rent_price" example:"8000"`
	Deposit      *float64 `json:"deposit,omitempty" example:"16000"`
	IsNegotiable bool     `json:"is_negotiable" example:"false"`
}

// CreatePropertyResult is the response returned after creating a property.
type CreatePropertyResult struct {
	Data CreatePropertyResultData `json:"data"`
}

// CreatePropertyResultData contains the identifier returned after property creation.
type CreatePropertyResultData struct {
	PropertyUUID string `json:"property_uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// ValidationError represents a client-side validation problem for property creation.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// ErrPropertyNotFound is returned when a property UUID does not exist or has been deleted.
var ErrPropertyNotFound = errors.New("property not found")

// PropertyRepository defines persistence operations for properties.
type PropertyRepository interface {
	GetModalityName(ctx context.Context, modalityID int32) (string, error)
	GetAllowedPeriods(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error)
	GetClauseValueTypes(ctx context.Context, clauseIDs []int32) (map[int32]int32, error)
	CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
	GetPropertyClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error)
	UpdatePropertyClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error
	GetPropertyServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error)
	UpdatePropertyServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error
	GetPropertyPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error)
	UpdatePropertyPrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error
}

// PropertyService defines application logic operations for properties.
type PropertyService interface {
	CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
	GetClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error)
	UpdateClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error
	GetServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error)
	UpdateServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error
	GetPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error)
	UpdatePrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error
}
