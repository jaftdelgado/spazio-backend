//go:build integration

package locations

import (
	"context"
	"strings"
	"testing"

	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

func TestIntegration_LocationsRepository(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	t.Run("list countries returns active countries", func(t *testing.T) {
		items, err := repo.ListCountries(ctx)
		if err != nil {
			t.Fatalf("ListCountries() error = %v", err)
		}
		if len(items) == 0 {
			t.Fatal("expected countries, got none")
		}
	})

	t.Run("list states filters by country and search", func(t *testing.T) {
		var countryID int32
		var stateName string
		err := pool.QueryRow(ctx, `
			SELECT s.country_id, s.name
			FROM states s
			JOIN countries c ON c.country_id = s.country_id
			WHERE c.is_active = true AND s.is_active = true
			LIMIT 1
		`).Scan(&countryID, &stateName)
		if err != nil {
			t.Fatalf("query active state: %v", err)
		}

		search := stateName[:minInt(len(stateName), 3)]
		items, err := repo.ListStates(ctx, ListStatesInput{CountryID: countryID, Search: search})
		if err != nil {
			t.Fatalf("ListStates() error = %v", err)
		}
		if len(items) == 0 {
			t.Fatalf("expected states for country_id %d search %q", countryID, search)
		}
		for _, item := range items {
			if !strings.Contains(strings.ToLower(item.Name), strings.ToLower(search)) {
				t.Fatalf("state %q does not match search %q", item.Name, search)
			}
		}
	})

	t.Run("list cities supports pagination and search", func(t *testing.T) {
		var stateID int32
		var cityName string
		err := pool.QueryRow(ctx, `
			SELECT c.state_id, c.name
			FROM cities c
			LIMIT 1
		`).Scan(&stateID, &cityName)
		if err != nil {
			t.Fatalf("query city: %v", err)
		}

		search := cityName[:minInt(len(cityName), 3)]
		items, total, err := repo.ListCities(ctx, ListCitiesInput{
			StateID:  stateID,
			Page:     1,
			PageSize: 10,
			Search:   search,
		})
		if err != nil {
			t.Fatalf("ListCities() error = %v", err)
		}
		if total == 0 || len(items) == 0 {
			t.Fatalf("expected city results for state_id %d search %q", stateID, search)
		}
		for _, item := range items {
			if !strings.Contains(strings.ToLower(item.Name), strings.ToLower(search)) {
				t.Fatalf("city %q does not match search %q", item.Name, search)
			}
		}
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
