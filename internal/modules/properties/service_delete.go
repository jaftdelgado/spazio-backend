package properties

import (
	"context"
	"errors"
	"fmt"
	"log"
)

func (s *service) DeleteProperty(ctx context.Context, propertyUUID string, input DeletePropertyInput) error {
	property, err := s.repository.GetProperty(ctx, propertyUUID)
	if err != nil {
		return fmt.Errorf("get property: %w", err)
	}

	if property.Data.StatusID != StatusAvailable {
		return ValidationError{Message: "property cannot be deleted: status is not available"}
	}

	propertyID := property.Data.PropertyID

	storageKeys, err := s.repository.GetPropertyStorageKeys(ctx, propertyID)
	if err != nil {
		return fmt.Errorf("get property storage keys: %w", err)
	}

	deletedKeys := make([]string, 0, len(storageKeys))
	for _, storageKey := range storageKeys {
		if err := s.r2Client.Delete(ctx, storageKey); err != nil {
			return errors.New("could not delete property photos from storage")
		}
		deletedKeys = append(deletedKeys, storageKey)
	}

	if err := s.repository.DeleteProperty(ctx, propertyID, input.ChangedByUserID); err != nil {
		for _, storageKey := range deletedKeys {
			log.Printf("orphaned storage key after transaction failure: %s", storageKey)
		}
		return errors.New("could not complete deletion: database transaction failed")
	}

	return nil
}
