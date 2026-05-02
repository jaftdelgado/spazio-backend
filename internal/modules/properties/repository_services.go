package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) GetPropertyServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyServicesResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyServicesResult{}, err
	}

	rows, err := r.queries.ListPropertyServiceIDs(ctx, propertyID)
	if err != nil {
		return GetPropertyServicesResult{}, fmt.Errorf("list property services: %w", err)
	}

	serviceIDs := make([]int32, 0, len(rows))
	for _, serviceID := range rows {
		serviceIDs = append(serviceIDs, serviceID)
	}

	return GetPropertyServicesResult{
		Data: GetPropertyServicesData{ServiceIDs: serviceIDs},
	}, nil
}

func (r *repository) UpdatePropertyServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return fmt.Errorf("parse property uuid: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := sqlcgen.New(tx)

	propertyID, err := getPropertyIDByUUID(ctx, queries, parsedUUID)
	if err != nil {
		return err
	}

	if err := queries.DeletePropertyServices(ctx, propertyID); err != nil {
		return fmt.Errorf("delete property services: %w", err)
	}

	for _, serviceID := range input.ServiceIDs {
		if err := queries.CreatePropertyService(ctx, sqlcgen.CreatePropertyServiceParams{
			PropertyID: propertyID,
			ServiceID:  serviceID,
		}); err != nil {
			return fmt.Errorf("create property service %d: %w", serviceID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
