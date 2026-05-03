package properties

import (
	"context"
	"errors"
	"time"
)

const (
	SubtypeResidential = "residential"
	SubtypeCommercial  = "commercial"
	SubtypeOther       = "other"

	ClauseValueTypeBoolean = 1
	ClauseValueTypeRange   = 2
	ClauseValueTypeInteger = 3
)

// CreatePropertyInput is the request payload required to register a property.
type CreatePropertyInput struct {
	OwnerID        int32                       `json:"owner_id" example:"1"`
	Subtype        string                      `json:"subtype" example:"residential"`
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

// ListPropertiesInput defines filters for listing property cards.
type ListPropertiesInput struct {
	Page           int32
	PageSize       int32
	Query          string
	StatusIDs      []int32
	PropertyTypeID int32
	ModalityID     int32
	CountryID      int32
	StateID        int32
	CityID         int32
	Sort           string
	Order          string
}

// ListPropertiesResult is the response payload returned by the properties list endpoint.
type ListPropertiesResult struct {
	Data []PropertyCardData `json:"data"`
	Meta ListPropertiesMeta `json:"meta"`
}

// ListPropertiesMeta contains pagination metadata for property cards.
type ListPropertiesMeta struct {
	TotalCount  int64 `json:"total_count"`
	TotalPages  int32 `json:"total_pages"`
	CurrentPage int32 `json:"current_page"`
	PageSize    int32 `json:"page_size"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
}

// PropertyCardData represents a property in card format.
type PropertyCardData struct {
	PropertyUUID  string                   `json:"property_uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Title         string                   `json:"title" example:"Apartment in Downtown"`
	CoverPhotoURL *string                  `json:"cover_photo_url" example:"https://cdn.example.com/properties/cover.jpg"`
	PropertyType  PropertyCardTypeData     `json:"property_type"`
	Modality      PropertyCardModalityData `json:"modality"`
	Status        PropertyCardStatusData   `json:"status"`
	Price         *PropertyCardPriceData   `json:"price"`
}

// PropertyCardTypeData contains the serialized property type used in cards.
type PropertyCardTypeData struct {
	PropertyTypeID int32   `json:"property_type_id" example:"1"`
	Name           string  `json:"name" example:"Apartment"`
	Icon           *string `json:"icon,omitempty" example:"/icons/apartment.svg"`
}

// PropertyCardModalityData contains the serialized modality used in cards.
type PropertyCardModalityData struct {
	ModalityID int32  `json:"modality_id" example:"2"`
	Name       string `json:"name" example:"Rent"`
}

// PropertyCardStatusData contains the serialized status used in cards.
type PropertyCardStatusData struct {
	StatusID int32  `json:"status_id" example:"2"`
	Name     string `json:"name" example:"Available"`
}

// PropertyCardPriceData contains the selected price shown in property cards.
type PropertyCardPriceData struct {
	Amount     float64 `json:"amount" example:"25000"`
	Currency   string  `json:"currency" example:"MXN"`
	PriceType  string  `json:"price_type" example:"rent"`
	PeriodName *string `json:"period_name" example:"Monthly"`
}

// GetPropertyClausesResult is the response returned by the property clauses list endpoint.
type GetPropertyClausesResult struct {
	Data []PropertyClauseData `json:"data"`
}

// GetPropertyResult es la respuesta del GET /properties/:uuid.
type GetPropertyResult struct {
	Data GetPropertyData `json:"data"`
}

// GetPropertyFullResult is the response returned by GET /properties/:uuid?full=true.
type GetPropertyFullResult struct {
	Data GetPropertyFullData `json:"data"`
}

// GetPropertyData contiene los datos base, subtipo y ubicación de la propiedad.
type GetPropertyData struct {
	PropertyUUID   string           `json:"property_uuid"`
	OwnerID        int32            `json:"owner_id"`
	Subtype        string           `json:"subtype"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	PropertyTypeID int32            `json:"property_type_id"`
	ModalityID     int32            `json:"modality_id"`
	LotArea        float64          `json:"lot_area"`
	IsFeatured     bool             `json:"is_featured"`
	Residential    *ResidentialData `json:"residential"`
	Commercial     *CommercialData  `json:"commercial"`
	Location       *LocationData    `json:"location"`
}

// GetPropertyFullData contains the base property data plus related resources.
type GetPropertyFullData struct {
	PropertyUUID   string                 `json:"property_uuid"`
	OwnerID        int32                  `json:"owner_id"`
	Subtype        string                 `json:"subtype"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	PropertyTypeID int32                  `json:"property_type_id"`
	ModalityID     int32                  `json:"modality_id"`
	LotArea        float64                `json:"lot_area"`
	IsFeatured     bool                   `json:"is_featured"`
	Residential    *ResidentialData       `json:"residential"`
	Commercial     *CommercialData        `json:"commercial"`
	Location       *LocationData          `json:"location"`
	Prices         PropertyFullPricesData `json:"prices"`
	Photos         []PropertyPhotoData    `json:"photos"`
	Services       []int32                `json:"services"`
	Clauses        []PropertyClauseData   `json:"clauses"`
}

// PropertyFullPricesData contains current prices and timeline entries for the property.
type PropertyFullPricesData struct {
	Current PropertyCurrentPricesData  `json:"current"`
	History []PropertyPriceHistoryData `json:"history"`
}

// PropertyCurrentPricesData contains the current prices grouped by sale and rent.
type PropertyCurrentPricesData struct {
	Sale *CurrentSalePriceDetailData  `json:"sale"`
	Rent []CurrentRentPriceDetailData `json:"rent"`
}

// CurrentSalePriceDetailData contains the current sale price details.
type CurrentSalePriceDetailData struct {
	Amount       float64    `json:"amount" example:"1500000"`
	Currency     string     `json:"currency" example:"MXN"`
	IsNegotiable bool       `json:"is_negotiable" example:"true"`
	ValidFrom    time.Time  `json:"valid_from" example:"2026-05-02T10:00:00Z"`
	ValidUntil   *time.Time `json:"valid_until,omitempty" example:"2026-06-02T10:00:00Z"`
	IsCurrent    bool       `json:"is_current" example:"true"`
}

// CurrentRentPriceDetailData contains a current rent price detail.
type CurrentRentPriceDetailData struct {
	Amount       float64    `json:"amount" example:"25000"`
	Currency     string     `json:"currency" example:"MXN"`
	PeriodName   *string    `json:"period_name" example:"Monthly"`
	IsNegotiable bool       `json:"is_negotiable" example:"false"`
	Deposit      *float64   `json:"deposit,omitempty" example:"50000"`
	ValidFrom    time.Time  `json:"valid_from" example:"2026-05-02T10:00:00Z"`
	ValidUntil   *time.Time `json:"valid_until,omitempty" example:"2026-06-02T10:00:00Z"`
	IsCurrent    bool       `json:"is_current" example:"true"`
}

// PropertyPriceHistoryData contains one historical price entry.
type PropertyPriceHistoryData struct {
	PriceType    string     `json:"price_type" example:"sale"`
	Amount       float64    `json:"amount" example:"1500000"`
	Currency     string     `json:"currency" example:"MXN"`
	PeriodName   *string    `json:"period_name,omitempty" example:"Monthly"`
	IsNegotiable bool       `json:"is_negotiable" example:"true"`
	Deposit      *float64   `json:"deposit,omitempty" example:"50000"`
	ValidFrom    time.Time  `json:"valid_from" example:"2026-05-02T10:00:00Z"`
	ValidUntil   *time.Time `json:"valid_until,omitempty" example:"2026-06-02T10:00:00Z"`
	IsCurrent    bool       `json:"is_current" example:"true"`
}

// ResidentialData contiene los campos del subtipo residencial.
type ResidentialData struct {
	Bedrooms         int16   `json:"bedrooms"`
	Bathrooms        int16   `json:"bathrooms"`
	Beds             int16   `json:"beds"`
	Floors           int16   `json:"floors"`
	ParkingSpots     int16   `json:"parking_spots"`
	BuiltArea        float64 `json:"built_area"`
	ConstructionYear int16   `json:"construction_year"`
	OrientationID    int32   `json:"orientation_id"`
	IsFurnished      bool    `json:"is_furnished"`
}

// CommercialData contiene los campos del subtipo comercial.
type CommercialData struct {
	CeilingHeight   float64 `json:"ceiling_height"`
	LoadingDocks    int16   `json:"loading_docks"`
	InternalOffices int16   `json:"internal_offices"`
	ThreePhasePower bool    `json:"three_phase_power"`
	LandUse         string  `json:"land_use"`
}

// LocationData contiene los campos de ubicación de la propiedad.
type LocationData struct {
	CityID          int32   `json:"city_id"`
	Neighborhood    string  `json:"neighborhood"`
	Street          string  `json:"street"`
	ExteriorNumber  string  `json:"exterior_number"`
	InteriorNumber  *string `json:"interior_number"`
	PostalCode      string  `json:"postal_code"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	IsPublicAddress bool    `json:"is_public_address"`
}

// UpdatePropertyInput es el payload del PATCH /properties/:uuid.
type UpdatePropertyInput struct {
	Title       *string                 `json:"title,omitempty"`
	Description *string                 `json:"description,omitempty"`
	LotArea     *float64                `json:"lot_area,omitempty"`
	IsFeatured  *bool                   `json:"is_featured,omitempty"`
	Residential *UpdateResidentialInput `json:"residential,omitempty"`
	Commercial  *UpdateCommercialInput  `json:"commercial,omitempty"`
	Location    *UpdateLocationInput    `json:"location,omitempty"`
}

// UpdateResidentialInput contiene los campos editables del subtipo residencial.
type UpdateResidentialInput struct {
	Bedrooms         *int16   `json:"bedrooms"`
	Bathrooms        *int16   `json:"bathrooms"`
	Beds             *int16   `json:"beds"`
	Floors           *int16   `json:"floors"`
	ParkingSpots     *int16   `json:"parking_spots"`
	BuiltArea        *float64 `json:"built_area"`
	ConstructionYear *int16   `json:"construction_year"`
	OrientationID    *int32   `json:"orientation_id"`
	IsFurnished      *bool    `json:"is_furnished"`
}

// UpdateCommercialInput contiene los campos editables del subtipo comercial.
type UpdateCommercialInput struct {
	CeilingHeight   *float64 `json:"ceiling_height"`
	LoadingDocks    *int16   `json:"loading_docks"`
	InternalOffices *int16   `json:"internal_offices"`
	ThreePhasePower *bool    `json:"three_phase_power"`
	LandUse         *string  `json:"land_use"`
}

// UpdateLocationInput contiene los campos editables de ubicación.
type UpdateLocationInput struct {
	CityID          *int32   `json:"city_id"`
	Neighborhood    *string  `json:"neighborhood"`
	Street          *string  `json:"street"`
	ExteriorNumber  *string  `json:"exterior_number"`
	InteriorNumber  *string  `json:"interior_number"`
	PostalCode      *string  `json:"postal_code"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	IsPublicAddress *bool    `json:"is_public_address"`
}

// UpdatePropertyResult es la respuesta del PATCH /properties/:uuid.
type UpdatePropertyResult struct {
	Message string `json:"message"`
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

// GetPropertyPhotosResult is the response returned by the property photos list endpoint.
type GetPropertyPhotosResult struct {
	Data []PropertyPhotoData `json:"data"`
}

// PropertyPhotoData represents the metadata of a linked photo.
type PropertyPhotoData struct {
	PhotoID    int32   `json:"photo_id" example:"12"`
	StorageKey string  `json:"storage_key" example:"properties/123/front.jpg"`
	MimeType   string  `json:"mime_type" example:"image/jpeg"`
	SortOrder  int16   `json:"sort_order" example:"0"`
	IsCover    bool    `json:"is_cover" example:"true"`
	Label      *string `json:"label,omitempty" example:"Front facade"`
	AltText    *string `json:"alt_text,omitempty" example:"Front facade of the property"`
}

// UpdatePropertyPhotosInput is the request payload used to replace the linked photo metadata of a property.
type UpdatePropertyPhotosInput struct {
	Photos []UpdatePhotoMetadataInput `json:"photos,omitempty"`
}

// UpdatePhotoMetadataInput contains the editable fields of a linked photo.
type UpdatePhotoMetadataInput struct {
	PhotoID   int32   `json:"photo_id" example:"12"`
	SortOrder int16   `json:"sort_order" example:"0"`
	IsCover   bool    `json:"is_cover" example:"true"`
	Label     *string `json:"label,omitempty" example:"Front facade"`
	AltText   *string `json:"alt_text,omitempty" example:"Front facade of the property"`
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
	GetPropertySubtype(ctx context.Context, propertyTypeID int32) (string, error)
	GetClauseValueTypes(ctx context.Context, clauseIDs []int32) (map[int32]int32, error)
	CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
	ListProperties(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error)
	GetPropertyClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error)
	UpdatePropertyClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error
	GetPropertyPhotos(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error)
	UpdatePropertyPhotos(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error
	GetPropertyServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error)
	UpdatePropertyServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error
	GetPropertyPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error)
	UpdatePropertyPrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error

	// New persistence operations for GET / PATCH
	GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error)
	GetFullProperty(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error)
	UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error)
}

// PropertyService defines application logic operations for properties.
type PropertyService interface {
	CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
	ListProperties(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error)
	GetClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error)
	UpdateClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error
	GetPhotos(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error)
	UpdatePhotos(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error
	GetServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error)
	UpdateServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error
	GetPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error)
	UpdatePrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error
	// New endpoints
	GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error)
	GetFullProperty(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error)
	UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error)
}
