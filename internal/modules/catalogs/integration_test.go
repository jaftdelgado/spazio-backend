//go:build integration

package catalogs

import (
	"context"
	"testing"

	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

func TestIntegration_CatalogsRepository(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	t.Run("list modalities returns seeded data", func(t *testing.T) {
		items, err := repo.ListModalities(ctx)
		if err != nil {
			t.Fatalf("ListModalities() error = %v", err)
		}
		if len(items) == 0 {
			t.Fatal("expected modalities, got none")
		}
		if items[0].ModalityID == 0 || items[0].Name == "" {
			t.Fatalf("unexpected modality row: %+v", items[0])
		}
	})

	t.Run("list property types and rent periods returns related data", func(t *testing.T) {
		var propertyTypeID int32
		err := pool.QueryRow(ctx, `
			SELECT ptp.property_type_id
			FROM property_type_periods ptp
			LIMIT 1
		`).Scan(&propertyTypeID)
		if err != nil {
			t.Fatalf("query property_type_id: %v", err)
		}

		propertyTypes, err := repo.ListPropertyTypes(ctx)
		if err != nil {
			t.Fatalf("ListPropertyTypes() error = %v", err)
		}
		if len(propertyTypes) == 0 {
			t.Fatal("expected property types, got none")
		}

		periods, err := repo.ListRentPeriodsByPropertyType(ctx, propertyTypeID)
		if err != nil {
			t.Fatalf("ListRentPeriodsByPropertyType() error = %v", err)
		}
		if len(periods) == 0 {
			t.Fatalf("expected rent periods for property_type_id %d", propertyTypeID)
		}
	})

	t.Run("list orientations returns seeded data", func(t *testing.T) {
		items, err := repo.ListOrientations(ctx)
		if err != nil {
			t.Fatalf("ListOrientations() error = %v", err)
		}
		if len(items) == 0 {
			t.Fatal("expected orientations, got none")
		}
		if items[0].OrientationID == 0 || items[0].Name == "" {
			t.Fatalf("unexpected orientation row: %+v", items[0])
		}
	})
}
