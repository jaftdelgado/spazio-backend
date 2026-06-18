//go:build integration

package properties

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

func TestIntegration_PropertiesRepository_ListAndGet(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	var propertyID int32
	var propertyUUID string
	var title string
	if err := pool.QueryRow(ctx, `
		SELECT property_id, property_uuid::text, title
		FROM properties
		WHERE deleted_at IS NULL
		LIMIT 1
	`).Scan(&propertyID, &propertyUUID, &title); err != nil {
		t.Fatalf("query integration property: %v", err)
	}

	getResult, err := repo.GetPropertyByUUID(ctx, propertyUUID)
	if err != nil {
		t.Fatalf("GetPropertyByUUID() error = %v", err)
	}
	if getResult.Data.PropertyUUID != propertyUUID {
		t.Fatalf("property uuid mismatch: got %q want %q", getResult.Data.PropertyUUID, propertyUUID)
	}
	if getResult.Data.Title != title {
		t.Fatalf("title mismatch: got %q want %q", getResult.Data.Title, title)
	}
	if getResult.Data.Location == nil {
		t.Fatal("expected location data")
	}

	query := title[:minPropertyInt(len(title), 4)]
	items, total, err := repo.ListProperties(ctx, ListPropertiesInput{
		Page:     1,
		PageSize: 10,
		Query:    query,
	})
	if err != nil {
		t.Fatalf("ListProperties() error = %v", err)
	}
	if total == 0 || len(items) == 0 {
		t.Fatalf("expected property list results for query %q", query)
	}
	found := false
	for _, item := range items {
		if item.PropertyUUID == propertyUUID {
			found = true
			if !strings.Contains(strings.ToLower(item.Title), strings.ToLower(query)) {
				t.Fatalf("expected title %q to contain query %q", item.Title, query)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected property_uuid %q for property_id %d in list results", propertyUUID, propertyID)
	}
}

func minPropertyInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestIntegration_PropertiesRepository_UpdatePropertyPrices_AddsNewRentPeriod(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	propertyUUID := "550e8400-e29b-41d4-a716-446655445000"

	if _, err := pool.Exec(ctx, `
		DELETE FROM rent_prices WHERE property_id = 500;
	`); err != nil {
		t.Fatalf("cleanup rent prices: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO rent_prices (
			property_id, period_id, rent_price, deposit, currency, is_negotiable,
			is_current, valid_from, changed_by_user_id
		) VALUES
			(500, 3, 8000, 16000, 'MXN', false, true, NOW(), 200)
	`); err != nil {
		t.Fatalf("seed active monthly rent price: %v", err)
	}

	err := repo.UpdatePropertyPrices(ctx, propertyUUID, UpdatePropertyPricesInput{
		Actor: ActorContext{UserID: 200, RoleID: RoleAdminID},
		RentPrices: []UpdateRentPriceInput{
			{PeriodID: 4, RentPrice: 90000, IsNegotiable: true},
		},
	})
	if err != nil {
		t.Fatalf("UpdatePropertyPrices() adding annual period error = %v", err)
	}

	var (
		rentPrice    float64
		currency     string
		isNegotiable bool
		isCurrent    bool
	)
	err = pool.QueryRow(ctx, `
		SELECT rent_price::float8, currency, is_negotiable, is_current
		FROM rent_prices
		WHERE property_id = 500
		  AND period_id = 4
		ORDER BY valid_from DESC
		LIMIT 1
	`).Scan(&rentPrice, &currency, &isNegotiable, &isCurrent)
	if err != nil {
		if err == pgx.ErrNoRows {
			t.Fatal("expected annual rent price row to be created")
		}
		t.Fatalf("query annual rent price: %v", err)
	}

	if rentPrice != 90000 {
		t.Fatalf("annual rent price mismatch: got %v want %v", rentPrice, 90000.0)
	}
	if currency != "MXN" {
		t.Fatalf("annual rent currency mismatch: got %q want %q", currency, "MXN")
	}
	if !isNegotiable {
		t.Fatal("expected annual rent price to be negotiable")
	}
	if !isCurrent {
		t.Fatal("expected annual rent price to be current")
	}
}
