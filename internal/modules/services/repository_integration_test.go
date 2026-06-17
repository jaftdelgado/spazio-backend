//go:build integration

package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestIntegration_SearchServices_MultilingualTags(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	shared.WithTransaction(t, pool, func(tx pgx.Tx) {
		repo := &repository{queries: sqlcgen.New(tx)}

		wellnessCategoryID := insertServiceCategory(t, ctx, tx, fmt.Sprintf("wellness_it_%d", suffix), "Wellness IT")
		featuresCategoryID := insertServiceCategory(t, ctx, tx, fmt.Sprintf("features_it_%d", suffix), "Features IT")

		poolServiceID := insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("POOL_IT_%d", suffix),
			icon:       "pool",
			categoryID: wellnessCategoryID,
			sortOrder:  10,
			searchTags: `{"es":["alberca","piscina"],"en":["swimming pool","pool"]}`,
		})
		wifiServiceID := insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("WIFI_IT_%d", suffix),
			icon:       "wifi",
			categoryID: featuresCategoryID,
			sortOrder:  20,
			searchTags: `{"es":["internet inalámbrico","red"],"en":["wireless internet","network"]}`,
		})
		nullTagsServiceID := insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("NULL_TAGS_IT_%d", suffix),
			icon:       "tag",
			categoryID: featuresCategoryID,
			sortOrder:  30,
			searchTags: "",
		})
		yogaServiceID := insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("YOGA_IT_%d", suffix),
			icon:       "yoga",
			categoryID: wellnessCategoryID,
			sortOrder:  40,
			searchTags: `{"es":["yoga","bienestar"],"en":["yoga","wellness"]}`,
		})

		tests := []struct {
			name       string
			input      SearchInput
			wantIDs    []int32
			wantTotal  int64
			wantPage   int32
			wantSize   int32
		}{
			{
				name:      "matches spanish tag",
				input:     SearchInput{Query: "alberca", Page: 1, PageSize: 10},
				wantIDs:   []int32{poolServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "matches english tag",
				input:     SearchInput{Query: "swimming pool", Page: 1, PageSize: 10},
				wantIDs:   []int32{poolServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "matches code case-insensitively",
				input:     SearchInput{Query: "wifi", Page: 1, PageSize: 10},
				wantIDs:   []int32{wifiServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "no match returns empty results",
				input:     SearchInput{Query: "xyznotexist", Page: 1, PageSize: 10},
				wantIDs:   []int32{},
				wantTotal: 0,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "null search tags do not break unrelated search",
				input:     SearchInput{Query: "wireless internet", Page: 1, PageSize: 10},
				wantIDs:   []int32{wifiServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "category filter with tag match",
				input:     SearchInput{Query: "yoga", CategoryID: wellnessCategoryID, Page: 1, PageSize: 10},
				wantIDs:   []int32{yogaServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				items, total, err := repo.SearchServices(ctx, tt.input)
				if err != nil {
					t.Fatalf("SearchServices returned error: %v", err)
				}

				if total != tt.wantTotal {
					t.Fatalf("total mismatch: got %d want %d", total, tt.wantTotal)
				}
				if len(items) != len(tt.wantIDs) {
					t.Fatalf("result length mismatch: got %d want %d", len(items), len(tt.wantIDs))
				}
				for i, item := range items {
					if item.ServiceID != tt.wantIDs[i] {
						t.Fatalf("item %d service_id mismatch: got %d want %d", i, item.ServiceID, tt.wantIDs[i])
					}
				}
			})
		}

		items, total, err := repo.SearchServices(ctx, SearchInput{
			Query:    fmt.Sprintf("NULL_TAGS_IT_%d", suffix),
			Page:     1,
			PageSize: 10,
		})
		if err != nil {
			t.Fatalf("SearchServices with NULL tags returned error: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].ServiceID != nullTagsServiceID {
			t.Fatalf("expected NULL-tagged service to remain searchable by code, got total=%d items=%v", total, items)
		}
	})
}

type serviceSeed struct {
	code       string
	icon       string
	categoryID int32
	sortOrder  int32
	searchTags string
}

func insertServiceCategory(t *testing.T, ctx context.Context, tx pgx.Tx, code string, name string) int32 {
	t.Helper()

	var id int32
	err := tx.QueryRow(ctx, `
		INSERT INTO service_categories (code, name)
		VALUES ($1, $2)
		RETURNING category_id
	`, code, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert service category: %v", err)
	}

	return id
}

func insertService(t *testing.T, ctx context.Context, tx pgx.Tx, seed serviceSeed) int32 {
	t.Helper()

	var id int32
	var err error
	if seed.searchTags == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO services (code, icon, category_id, is_active, is_deprecated, sort_order, search_tags)
			VALUES ($1, $2, $3, true, false, $4, NULL)
			RETURNING service_id
		`, seed.code, seed.icon, seed.categoryID, seed.sortOrder).Scan(&id)
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO services (code, icon, category_id, is_active, is_deprecated, sort_order, search_tags)
			VALUES ($1, $2, $3, true, false, $4, $5::jsonb)
			RETURNING service_id
		`, seed.code, seed.icon, seed.categoryID, seed.sortOrder, seed.searchTags).Scan(&id)
	}
	if err != nil {
		t.Fatalf("insert service: %v", err)
	}

	return id
}
