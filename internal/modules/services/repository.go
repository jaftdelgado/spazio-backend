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

func (r *repository) ListPopularServices(ctx context.Context, limit int32) ([]Service, int64, error) {
	total, err := r.queries.CountActiveServices(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count active services: %w", err)
	}

	rows, err := r.queries.ListPopularServices(ctx, limit)
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

	return services, total, nil
}

func (r *repository) SearchServices(ctx context.Context, query string, limit int32) ([]Service, int64, error) {
	textQuery := pgtype.Text{String: query, Valid: true}

	total, err := r.queries.CountSearchServices(ctx, textQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("count search services: %w", err)
	}

	rows, err := r.queries.SearchServices(ctx, sqlcgen.SearchServicesParams{
		Query:       textQuery,
		SearchLimit: limit,
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

	return services, total, nil
}
