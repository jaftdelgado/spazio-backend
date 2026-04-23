// Package repository provides sqlc-backed data access helpers.
package repository

import (
	"context"

	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

// PropertyRepository defines the persistence contract for properties.
type PropertyRepository interface {
	CreateProperty(ctx context.Context, arg sqlcgen.CreatePropertyParams) (sqlcgen.CreatePropertyRow, error)
}

// SQLCPropertyRepository adapts sqlc queries to the repository contract.
type SQLCPropertyRepository struct {
	queries *sqlcgen.Queries
}

// NewPropertyRepository builds a repository backed by sqlc queries.
func NewPropertyRepository(queries *sqlcgen.Queries) *SQLCPropertyRepository {
	return &SQLCPropertyRepository{queries: queries}
}

// CreateProperty inserts a property through sqlc.
func (r *SQLCPropertyRepository) CreateProperty(ctx context.Context, arg sqlcgen.CreatePropertyParams) (sqlcgen.CreatePropertyRow, error) {
	return r.queries.CreateProperty(ctx, arg)
}
