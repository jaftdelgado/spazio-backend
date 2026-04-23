package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

// CreatePropertyInput represents the payload required to create a property.
type CreatePropertyInput struct {
	OwnerID        int32  `json:"owner_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	PropertyTypeID int32  `json:"property_type_id"`
	ModalityID     int32  `json:"modality_id"`
	StatusID       int32  `json:"status_id"`
	CoverPhotoURL  string `json:"cover_photo_url"`
}

// CreatePropertyResult represents the basic property response returned by the service.
type CreatePropertyResult struct {
	PropertyID int32     `json:"property_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

type propertyRepository interface {
	CreateProperty(ctx context.Context, arg sqlcgen.CreatePropertyParams) (sqlcgen.CreatePropertyRow, error)
}

// PropertyService contains property-related application logic.
type PropertyService struct {
	repository propertyRepository
}

// NewPropertyService builds a property service.
func NewPropertyService(repository propertyRepository) *PropertyService {
	return &PropertyService{repository: repository}
}

// CreateProperty stores a new property and returns the created summary.
func (s *PropertyService) CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	row, err := s.repository.CreateProperty(ctx, sqlcgen.CreatePropertyParams{
		OwnerID:        input.OwnerID,
		Title:          input.Title,
		Description:    input.Description,
		PropertyTypeID: input.PropertyTypeID,
		ModalityID:     input.ModalityID,
		StatusID:       input.StatusID,
		CoverPhotoUrl:  input.CoverPhotoURL,
	})
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("create property: %w", err)
	}

	return CreatePropertyResult{
		PropertyID: row.PropertyID,
		Title:      row.Title,
		CreatedAt:  row.CreatedAt,
	}, nil
}
