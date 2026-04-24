package properties

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	db *pgxpool.Pool
}

// NewRepository builds a property repository implementation.
func NewRepository(db *pgxpool.Pool) PropertyRepository {
	return &repository{db: db}
}

func (r *repository) CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	_ = input

	if err := ctx.Err(); err != nil {
		return CreatePropertyResult{}, fmt.Errorf("create property cancelled: %w", err)
	}

	return CreatePropertyResult{}, fmt.Errorf("create property query not implemented")
}
