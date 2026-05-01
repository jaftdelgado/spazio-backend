package uploads

import (
	"context"
	"errors"
	"io"
)

// ErrPropertyNotFound is returned when the property UUID does not exist or has been deleted.
var ErrPropertyNotFound = errors.New("property not found")

type UploadPhotoInput struct {
	PropertyUUID string
	MimeType     string
	Label        *string
	AltText      *string
	SortOrder    int32
	IsCover      bool
	File         io.Reader
}

type UploadPhotoResult struct {
	PhotoID    int32  `json:"photo_id"`
	StorageKey string `json:"storage_key"`
	URL        string `json:"url"`
}

type SavePhotoInput struct {
	PropertyUUID string
	StorageKey   string
	MimeType     string
	Label        *string
	AltText      *string
	SortOrder    int32
	IsCover      bool
}

type UploadsRepository interface {
	SavePropertyPhoto(ctx context.Context, input SavePhotoInput) (int32, error)
}

type photoStorage interface {
	Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error
	Delete(ctx context.Context, storageKey string) error
	PublicURL(ctx context.Context, storageKey string) (string, error)
}

type UploadsService interface {
	UploadPropertyPhoto(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error)
}
