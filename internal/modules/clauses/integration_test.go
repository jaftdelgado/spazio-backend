//go:build integration

package clauses

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestIntegration_ClausesRepository_ListAndSearch(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	shared.WithTransaction(t, pool, func(tx pgx.Tx) {
		repo := &repository{queries: sqlcgen.New(tx)}

		var valueTypeID int32
		if err := tx.QueryRow(ctx, `SELECT value_type_id FROM clause_value_types LIMIT 1`).Scan(&valueTypeID); err != nil {
			t.Fatalf("query clause value type: %v", err)
		}

		var modalityID int32
		if err := tx.QueryRow(ctx, `SELECT modality_id FROM modalities LIMIT 1`).Scan(&modalityID); err != nil {
			t.Fatalf("query modality id: %v", err)
		}

		code := fmt.Sprintf("integration_clause_%d", suffix)
		var clauseID int32
		if err := tx.QueryRow(ctx, `
			INSERT INTO clauses (code, name, value_type_id, is_active, is_deprecated, sort_order, search_tags)
			VALUES ($1, $2, $3, true, false, $4, $5::jsonb)
			RETURNING clause_id
		`, code, "Integration Clause", valueTypeID, 9999, `["integration-search","integration tag"]`).Scan(&clauseID); err != nil {
			t.Fatalf("insert clause: %v", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO clause_modalities (clause_id, modality_id)
			VALUES ($1, $2)
		`, clauseID, modalityID); err != nil {
			t.Fatalf("insert clause modality: %v", err)
		}

		items, total, err := repo.ListClauses(ctx, modalityID, 20, 0)
		if err != nil {
			t.Fatalf("ListClauses() error = %v", err)
		}
		if total == 0 {
			t.Fatal("expected list total > 0")
		}
		found := false
		for _, item := range items {
			if item.ClauseID == clauseID {
				found = true
				if item.Code != code {
					t.Fatalf("code mismatch: got %q want %q", item.Code, code)
				}
				break
			}
		}
		if !found {
			t.Fatalf("expected clause_id %d in list result", clauseID)
		}

		items, total, err = repo.SearchClauses(ctx, modalityID, "integration-search", 20, 0)
		if err != nil {
			t.Fatalf("SearchClauses() error = %v", err)
		}
		if total == 0 || len(items) == 0 {
			t.Fatal("expected search results, got none")
		}
		if items[0].ClauseID != clauseID {
			t.Fatalf("expected first search hit clause_id %d, got %d", clauseID, items[0].ClauseID)
		}
	})
}
