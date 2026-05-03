package properties

import (
	"context"
	"fmt"

	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) GetPropertyStorageKeys(ctx context.Context, propertyID int32) ([]string, error) {
	rows, err := r.queries.GetPropertyStorageKeys(ctx, propertyID)
	if err != nil {
		return nil, fmt.Errorf("get property storage keys: %w", err)
	}

	return rows, nil
}

func (r *repository) DeleteProperty(ctx context.Context, propertyID int32, changedByUserID int32) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := sqlcgen.New(tx)

	if err := queries.ClearPropertyCoverPhotoURL(ctx, propertyID); err != nil {
		return fmt.Errorf("clear property cover photo url: %w", err)
	}

	if err := queries.DeletePropertyPhotos(ctx, propertyID); err != nil {
		return fmt.Errorf("delete property photos: %w", err)
	}

	if err := queries.SoftDeleteProperty(ctx, propertyID); err != nil {
		return fmt.Errorf("soft delete property: %w", err)
	}

	if err := queries.InsertPropertyStatusHistory(ctx, sqlcgen.InsertPropertyStatusHistoryParams{
		PropertyID:       propertyID,
		PreviousStatusID: StatusAvailable,
		NewStatusID:      StatusDeleted,
		ChangedByUserID:  changedByUserID,
	}); err != nil {
		return fmt.Errorf("insert property status history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
