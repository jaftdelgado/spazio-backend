package catalogs

import "context"

// Modality is a catalog item exposed by the modalities endpoint.
type Modality struct {
	ModalityID int32  `json:"modality_id"`
	Name       string `json:"name"`
}

// PropertyType is a catalog item exposed by the property-types endpoint.
type PropertyType struct {
	PropertyTypeID int32   `json:"property_type_id"`
	Name           string  `json:"name"`
	Icon           *string `json:"icon,omitempty"`
}

// ListModalitiesResult is the response payload returned by the modalities use case.
type ListModalitiesResult struct {
	Data []Modality `json:"data"`
}

// ListPropertyTypesResult is the response payload returned by the property-types use case.
type ListPropertyTypesResult struct {
	Data []PropertyType `json:"data"`
}

// CatalogsRepository defines persistence operations for the catalogs module.
type CatalogsRepository interface {
	ListModalities(ctx context.Context) ([]Modality, error)
	ListPropertyTypes(ctx context.Context) ([]PropertyType, error)
}

// CatalogsService defines business operations for the catalogs module.
type CatalogsService interface {
	ListModalities(ctx context.Context) (ListModalitiesResult, error)
	ListPropertyTypes(ctx context.Context) (ListPropertyTypesResult, error)
}
