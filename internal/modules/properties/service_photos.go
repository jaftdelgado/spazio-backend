package properties

import (
	"context"
	"fmt"
)

func (s *service) GetPhotos(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error) {
	result, err := s.repository.GetPropertyPhotos(ctx, propertyUUID)
	if err != nil {
		return GetPropertyPhotosResult{}, fmt.Errorf("get property photos: %w", err)
	}

	return result, nil
}

func (s *service) UpdatePhotos(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error {
	if err := validatePhotoMetadataInputs(input.Photos); err != nil {
		return ValidationError{Message: err.Error()}
	}

	if err := s.repository.UpdatePropertyPhotos(ctx, propertyUUID, input); err != nil {
		return fmt.Errorf("update property photos: %w", err)
	}

	return nil
}

func validatePhotoMetadataInputs(photos []UpdatePhotoMetadataInput) error {
	if len(photos) == 0 {
		return nil
	}

	coverCount := 0
	seen := make(map[int32]struct{}, len(photos))
	for _, photo := range photos {
		if photo.PhotoID <= 0 {
			return fmt.Errorf("photo_id must be greater than 0")
		}

		if _, ok := seen[photo.PhotoID]; ok {
			return fmt.Errorf("photo_id %d is duplicated", photo.PhotoID)
		}
		seen[photo.PhotoID] = struct{}{}

		if photo.IsCover {
			coverCount++
		}
	}

	if coverCount != 1 {
		return fmt.Errorf("exactly one photo must have is_cover = true")
	}

	return nil
}
