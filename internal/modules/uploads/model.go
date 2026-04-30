package uploads

import (
	"context"
	"io"
)

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

type UploadsService interface {
	UploadPropertyPhoto(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error)
}
