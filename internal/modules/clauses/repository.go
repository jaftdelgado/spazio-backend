package clauses

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	queries *sqlcgen.Queries
}

// NewRepository builds a clauses repository implementation.
func NewRepository(db *pgxpool.Pool) ClausesRepository {
	return &repository{queries: sqlcgen.New(db)}
}

func (r *repository) ListClauses(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error) {
	rows, err := r.queries.ListClauses(ctx, sqlcgen.ListClausesParams{
		ModalityID: modalityID,
		PageSize:   pageSize,
		PageOffset: pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list clauses: %w", err)
	}

	clauses := make([]Clause, 0, len(rows))
	for _, row := range rows {
		clauses = append(clauses, Clause{
			ClauseID: row.ClauseID,
			Code:     row.Code,
			ValueType: ClauseValueType{
				Code: row.ValueTypeCode,
			},
			SortOrder: row.SortOrder,
		})
	}

	if len(rows) == 0 {
		return clauses, 0, nil
	}

	return clauses, rows[0].TotalCount, nil
}

func (r *repository) SearchClauses(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error) {
	rows, err := r.queries.SearchClauses(ctx, sqlcgen.SearchClausesParams{
		ModalityID: modalityID,
		Query:      query,
		PageSize:   pageSize,
		PageOffset: pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search clauses: %w", err)
	}

	clauses := make([]Clause, 0, len(rows))
	for _, row := range rows {
		clauses = append(clauses, Clause{
			ClauseID: row.ClauseID,
			Code:     row.Code,
			ValueType: ClauseValueType{
				Code: row.ValueTypeCode,
			},
			SortOrder: row.SortOrder,
		})
	}

	if len(rows) == 0 {
		return clauses, 0, nil
	}

	return clauses, rows[0].TotalCount, nil
}
