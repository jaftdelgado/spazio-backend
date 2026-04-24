package properties

import (
	"context"
	"time"
)

// CreatePropertyInput is the payload required to create a property.
type CreatePropertyInput struct {
	OwnerID        int32  `json:"owner_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	PropertyTypeID int32  `json:"property_type_id"`
	ModalityID     int32  `json:"modality_id"`
	StatusID       int32  `json:"status_id"`
	CoverPhotoURL  string `json:"cover_photo_url"`
}

// CreatePropertyResult is the response returned after creating a property.
type CreatePropertyResult struct {
	PropertyID int32     `json:"property_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

// PropertyRepository defines persistence operations for properties.
type PropertyRepository interface {
	CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
}

// PropertyService defines application logic operations for properties.
type PropertyService interface {
	CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error)
}
