package catalogs

import "context"

// Modality is a catalog item exposed by the modalities endpoint.
type Modality struct {
	ModalityID int32  `json:"modality_id" example:"1"`
	Name       string `json:"name" example:"Rent"`
}

// PropertyType is a catalog item exposed by the property-types endpoint.
type PropertyType struct {
	PropertyTypeID int32   `json:"property_type_id" example:"1"`
	Name           string  `json:"name" example:"Apartment"`
	Icon           *string `json:"icon,omitempty" example:"/icons/apartment.svg"`
}

// RentPeriod is a catalog item exposed by the rent-periods endpoint.
type RentPeriod struct {
	PeriodID int32  `json:"period_id" example:"1"`
	Name     string `json:"name" example:"Monthly"`
}

// Orientation is a catalog item exposed by the orientations endpoint.
type Orientation struct {
	OrientationID int32  `json:"orientation_id" example:"1"`
	Name          string `json:"name" example:"North"`
}

// ListModalitiesResult is the response payload returned by the modalities use case.
type ListModalitiesResult struct {
	Data []Modality `json:"data"`
}

// ListPropertyTypesResult is the response payload returned by the property-types use case.
type ListPropertyTypesResult struct {
	Data []PropertyType `json:"data"`
}

// ListRentPeriodsResult is the response payload returned by the rent-periods use case.
type ListRentPeriodsResult struct {
	Data []RentPeriod `json:"data"`
}

// ListOrientationsResult is the response payload returned by the orientations use case.
type ListOrientationsResult struct {
	Data []Orientation `json:"data"`
}

// CatalogsRepository defines persistence operations for the catalogs module.
type CatalogsRepository interface {
	ListModalities(ctx context.Context) ([]Modality, error)
	ListPropertyTypes(ctx context.Context) ([]PropertyType, error)
	ListRentPeriodsByPropertyType(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error)
	ListOrientations(ctx context.Context) ([]Orientation, error)
}

// CatalogsService defines business operations for the catalogs module.
type CatalogsService interface {
	ListModalities(ctx context.Context) (ListModalitiesResult, error)
	ListPropertyTypes(ctx context.Context) (ListPropertyTypesResult, error)
	ListRentPeriods(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error)
	ListOrientations(ctx context.Context) (ListOrientationsResult, error)
}
