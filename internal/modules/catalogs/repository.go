package catalogs

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	queries *sqlcgen.Queries
}

// NewRepository builds a catalogs repository implementation.
func NewRepository(db *pgxpool.Pool) CatalogsRepository {
	return &repository{queries: sqlcgen.New(db)}
}

func (r *repository) ListModalities(ctx context.Context) ([]Modality, error) {
	rows, err := r.queries.ListModalities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list modalities: %w", err)
	}

	modalities := make([]Modality, 0, len(rows))
	for _, row := range rows {
		modalities = append(modalities, Modality{
			ModalityID: row.ModalityID,
			Name:       row.Name,
		})
	}

	return modalities, nil
}

func (r *repository) ListPropertyTypes(ctx context.Context) ([]PropertyType, error) {
	rows, err := r.queries.ListPropertyTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list property types: %w", err)
	}

	propertyTypes := make([]PropertyType, 0, len(rows))
	for _, row := range rows {
		var icon *string
		if row.Icon.Valid {
			icon = &row.Icon.String
		}

		propertyTypes = append(propertyTypes, PropertyType{
			PropertyTypeID: row.PropertyTypeID,
			Name:           row.Name,
			Icon:           icon,
		})
	}

	return propertyTypes, nil
}
