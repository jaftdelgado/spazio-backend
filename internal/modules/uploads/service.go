package uploads

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/storage"
)

type service struct {
	repository UploadsRepository
	r2Client   *storage.R2Client
}

func NewService(repository UploadsRepository, r2Client *storage.R2Client) UploadsService {
	return &service{
		repository: repository,
		r2Client:   r2Client,
	}
}

func (s *service) UploadPropertyPhoto(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
	ext := "bin"
	switch input.MimeType {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/webp":
		ext = "webp"
	}

	randomUUID := uuid.New().String()
	storageKey := fmt.Sprintf("properties/%s/photos/%s.%s", input.PropertyUUID, randomUUID, ext)

	err := s.r2Client.Upload(ctx, storageKey, input.MimeType, input.File)
	if err != nil {
		return UploadPhotoResult{}, fmt.Errorf("upload to r2: %w", err)
	}

	photoID, err := s.repository.SavePropertyPhoto(ctx, SavePhotoInput{
		PropertyUUID: input.PropertyUUID,
		StorageKey:   storageKey,
		MimeType:     input.MimeType,
		Label:        input.Label,
		AltText:      input.AltText,
		SortOrder:    int32(input.SortOrder),
		IsCover:      input.IsCover,
	})
	if err != nil {
		return UploadPhotoResult{}, fmt.Errorf("save property photo: %w", err)
	}

	url, err := s.r2Client.PublicURL(ctx, storageKey)
	if err != nil {
		return UploadPhotoResult{}, fmt.Errorf("get public url: %w", err)
	}

	return UploadPhotoResult{
		PhotoID:    photoID,
		StorageKey: storageKey,
		URL:        url,
	}, nil
}
