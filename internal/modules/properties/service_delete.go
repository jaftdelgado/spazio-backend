package properties

import (
	"context"
	"errors"
	"fmt"
	"log"
)

func (s *service) DeleteProperty(ctx context.Context, propertyUUID string, input DeletePropertyInput) error {
	if err := requireAdminActor(input.Actor); err != nil {
		return err
	}

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

	if err := s.repository.DeleteProperty(ctx, propertyID, input.ChangedByUserID); err != nil {
		return errors.New("could not complete deletion: database transaction failed")
	}

	for _, storageKey := range storageKeys {
		if err := s.r2Client.Delete(ctx, storageKey); err != nil {
			log.Printf("property delete storage cleanup failed for property_id=%d storage_key=%s: %v", propertyID, storageKey, err)
		}
	}

	return nil
}
