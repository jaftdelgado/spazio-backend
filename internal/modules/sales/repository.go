package sales

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		pool:    db,
		queries: sqlcgen.New(db),
	}
}

func (r *repository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin sales transaction: %w", err)
	}

	return tx, nil
}

func (r *repository) WithTx(tx pgx.Tx) Repository {
	return &repository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *repository) GetSalePropertyByUUID(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error) {
	row, err := r.queries.GetSalePropertyByUUID(ctx, pgtype.UUID{Bytes: propertyUUID, Valid: true})
	if err != nil {
		return sqlcgen.GetSalePropertyByUUIDRow{}, fmt.Errorf("get sale property by uuid: %w", err)
	}

	return row, nil
}

func (r *repository) GetCurrentSalePriceByPropertyID(ctx context.Context, propertyID int32) (sqlcgen.GetCurrentSalePriceByPropertyIDRow, error) {
	row, err := r.queries.GetCurrentSalePriceByPropertyID(ctx, propertyID)
	if err != nil {
		return sqlcgen.GetCurrentSalePriceByPropertyIDRow{}, fmt.Errorf("get current sale price by property id: %w", err)
	}

	return row, nil
}

func (r *repository) CreateSaleTransaction(ctx context.Context, arg sqlcgen.CreateSaleTransactionParams) (sqlcgen.CreateSaleTransactionRow, error) {
	record, err := r.queries.CreateSaleTransaction(ctx, arg)
	if err != nil {
		return sqlcgen.CreateSaleTransactionRow{}, fmt.Errorf("create sale transaction: %w", err)
	}

	return record, nil
}

func (r *repository) CreateSalePropertyStatusHistory(ctx context.Context, arg sqlcgen.CreateSalePropertyStatusHistoryParams) error {
	if err := r.queries.CreateSalePropertyStatusHistory(ctx, arg); err != nil {
		return fmt.Errorf("create sale property status history: %w", err)
	}

	return nil
}
