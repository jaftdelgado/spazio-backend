package catalogs

import "context"

// Modality is a catalog item exposed by the modalities endpoint.
type Modality struct {
	ModalityID int32  `json:"modality_id"`
	Name       string `json:"name"`
}

// ListModalitiesResult is the response payload returned by the modalities use case.
type ListModalitiesResult struct {
	Data []Modality `json:"data"`
}

// CatalogsRepository defines persistence operations for the catalogs module.
type CatalogsRepository interface {
	ListModalities(ctx context.Context) ([]Modality, error)
}

// CatalogsService defines business operations for the catalogs module.
type CatalogsService interface {
	ListModalities(ctx context.Context) (ListModalitiesResult, error)
}
