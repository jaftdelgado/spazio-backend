//go:build integration

package properties

import (
	"context"
	"strings"
	"testing"

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
