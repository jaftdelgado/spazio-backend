package uploads

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/google/uuid"
	"github.com/nickalie/go-webpbin"
)

type service struct {
	repository   UploadsRepository
	r2Client     photoStorage
	encodeToWebP func(UploadPhotoInput) ([]byte, error)
}

func NewService(repository UploadsRepository, r2Client photoStorage) UploadsService {
	return &service{
		repository:   repository,
		r2Client:     r2Client,
		encodeToWebP: convertToWebP,
	}
}

func (s *service) UploadPropertyPhoto(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
	webpData, err := s.encodeToWebP(input)
	if err != nil {
		return UploadPhotoResult{}, fmt.Errorf("convert to webp: %w", err)
	}

	storageKey := fmt.Sprintf("properties/%s/photos/%s.webp", input.PropertyUUID, uuid.New().String())

	if err := s.r2Client.Upload(ctx, storageKey, "image/webp", bytes.NewReader(webpData)); err != nil {
		return UploadPhotoResult{}, fmt.Errorf("upload to r2: %w", err)
	}

	photoID, err := s.repository.SavePropertyPhoto(ctx, SavePhotoInput{
		PropertyUUID: input.PropertyUUID,
		StorageKey:   storageKey,
		MimeType:     "image/webp",
		Label:        input.Label,
		AltText:      input.AltText,
		SortOrder:    input.SortOrder,
		IsCover:      input.IsCover,
	})
	if err != nil {
		if deleteErr := s.r2Client.Delete(ctx, storageKey); deleteErr != nil {
			log.Printf("delete orphaned upload %s: %v", storageKey, deleteErr)
		}
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

// convertToWebP decodes any supported image format and encodes it as WebP.
// Supported inputs: image/jpeg, image/png, image/webp.
// Extracted as a standalone function so it can be reused for profile photos
// or any other upload flow in the future.
func convertToWebP(input UploadPhotoInput) ([]byte, error) {
	img, _, err := image.Decode(input.File)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	var buf bytes.Buffer
	if err := webpbin.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}

	return buf.Bytes(), nil
}
