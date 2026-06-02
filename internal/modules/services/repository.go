package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	queries *sqlcgen.Queries
}

// NewRepository builds a services repository implementation.
func NewRepository(db *pgxpool.Pool) ServicesRepository {
	return &repository{queries: sqlcgen.New(db)}
}

func (r *repository) ListPopularServices(ctx context.Context, input ListPopularInput) ([]Service, int64, error) {
	offset := (input.Page - 1) * input.PageSize

	rows, err := r.queries.ListPopularServices(ctx, sqlcgen.ListPopularServicesParams{
		CategoryID: input.CategoryID,
		PageSize:   input.PageSize,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list popular services: %w", err)
	}

	services := make([]Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, Service{
			ServiceID:    row.ServiceID,
			Code:         row.Code,
			Icon:         row.Icon,
			CategoryCode: row.CategoryCode,
		})
	}

	if len(rows) == 0 {
		return services, 0, nil
	}

	return services, rows[0].TotalCount, nil
}

func (r *repository) SearchServices(ctx context.Context, input SearchInput) ([]Service, int64, error) {
	offset := (input.Page - 1) * input.PageSize
	textQuery := pgtype.Text{String: input.Query, Valid: true}

	rows, err := r.queries.SearchServices(ctx, sqlcgen.SearchServicesParams{
		Query:      textQuery,
		CategoryID: input.CategoryID,
		PageSize:   input.PageSize,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search services: %w", err)
	}

	services := make([]Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, Service{
			ServiceID:    row.ServiceID,
			Code:         row.Code,
			Icon:         row.Icon,
			CategoryCode: row.CategoryCode,
		})
	}

	if len(rows) == 0 {
		return services, 0, nil
	}

	return services, rows[0].TotalCount, nil
}
