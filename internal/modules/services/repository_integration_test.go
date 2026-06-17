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
			searchTags: fmt.Sprintf(`{"es":["alberca-it-%d","piscina-it-%d"],"en":["swimming-pool-it-%d","pool-it-%d"]}`, suffix, suffix, suffix, suffix),
		})
		wifiServiceID := insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("WIFI_IT_%d", suffix),
			icon:       "wifi",
			categoryID: featuresCategoryID,
			sortOrder:  20,
			searchTags: fmt.Sprintf(`{"es":["internet-it-%d","red-it-%d"],"en":["wireless-internet-it-%d","network-it-%d"]}`, suffix, suffix, suffix, suffix),
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
			searchTags: fmt.Sprintf(`{"es":["yoga-it-%d","bienestar-it-%d"],"en":["yoga-it-%d","wellness-it-%d"]}`, suffix, suffix, suffix, suffix),
		})

		tests := []struct {
			name      string
			input     SearchInput
			wantIDs   []int32
			wantTotal int64
			wantPage  int32
			wantSize  int32
		}{
			{
				name:      "matches spanish tag",
				input:     SearchInput{Query: fmt.Sprintf("alberca-it-%d", suffix), Page: 1, PageSize: 10},
				wantIDs:   []int32{poolServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "matches english tag",
				input:     SearchInput{Query: fmt.Sprintf("swimming-pool-it-%d", suffix), Page: 1, PageSize: 10},
				wantIDs:   []int32{poolServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "matches code case-insensitively",
				input:     SearchInput{Query: fmt.Sprintf("wifi_it_%d", suffix), Page: 1, PageSize: 10},
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
				input:     SearchInput{Query: fmt.Sprintf("wireless-internet-it-%d", suffix), Page: 1, PageSize: 10},
				wantIDs:   []int32{wifiServiceID},
				wantTotal: 1,
				wantPage:  1,
				wantSize:  10,
			},
			{
				name:      "category filter with tag match",
				input:     SearchInput{Query: fmt.Sprintf("yoga-it-%d", suffix), CategoryID: wellnessCategoryID, Page: 1, PageSize: 10},
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

func TestIntegration_ListPopularServices_PaginatesAndFiltersByCategory(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	shared.WithTransaction(t, pool, func(tx pgx.Tx) {
		repo := &repository{queries: sqlcgen.New(tx)}

		categoryID := insertServiceCategory(t, ctx, tx, fmt.Sprintf("popular_it_%d", suffix), "Popular IT")
		otherCategoryID := insertServiceCategory(t, ctx, tx, fmt.Sprintf("popular_other_it_%d", suffix), "Popular Other IT")

		firstID := insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("POPULAR_A_%d", suffix),
			icon:       "sparkles",
			categoryID: categoryID,
			sortOrder:  10,
			searchTags: `{"es":["popular"],"en":["popular"]}`,
		})
		_ = insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("POPULAR_B_%d", suffix),
			icon:       "sparkles",
			categoryID: categoryID,
			sortOrder:  20,
			searchTags: `{"es":["popular"],"en":["popular"]}`,
		})
		_ = insertService(t, ctx, tx, serviceSeed{
			code:       fmt.Sprintf("POPULAR_C_%d", suffix),
			icon:       "sparkles",
			categoryID: otherCategoryID,
			sortOrder:  30,
			searchTags: `{"es":["popular"],"en":["popular"]}`,
		})

		items, total, err := repo.ListPopularServices(ctx, ListPopularInput{
			CategoryID: categoryID,
			Page:       1,
			PageSize:   1,
		})
		if err != nil {
			t.Fatalf("ListPopularServices() error = %v", err)
		}
		if total != 2 {
			t.Fatalf("total mismatch: got %d want 2", total)
		}
		if len(items) != 1 {
			t.Fatalf("expected one paginated result, got %d", len(items))
		}
		if items[0].ServiceID != firstID {
			t.Fatalf("expected first paginated item %d, got %d", firstID, items[0].ServiceID)
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
