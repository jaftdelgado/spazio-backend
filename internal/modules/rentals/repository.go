package rentals

import (
	"context"
	"fmt"
	"time"

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

func (r *repository) WithTx(tx pgx.Tx) Repository {
	return &repository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *repository) GetRentalPropertyByUUID(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
	row, err := r.queries.GetRentalPropertyByUUID(ctx, pgtype.UUID{Bytes: propertyUUID, Valid: true})
	if err != nil {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, fmt.Errorf("get rental property by uuid: %w", err)
	}
	return row, nil
}

func (r *repository) GetAllowedRentalPeriods(ctx context.Context, propertyTypeID int32) ([]int32, error) {
	rows, err := r.queries.GetAllowedRentalPeriods(ctx, propertyTypeID)
	if err != nil {
		return nil, fmt.Errorf("get allowed rental periods: %w", err)
	}
	return rows, nil
}

func (r *repository) ListRentalActivePrices(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
	rows, err := r.queries.ListRentalActivePrices(ctx, propertyID)
	if err != nil {
		return nil, fmt.Errorf("list rental active prices: %w", err)
	}
	return rows, nil
}

func (r *repository) ListRentalBlockedDates(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
	rows, err := r.queries.ListRentalBlockedDates(ctx, sqlcgen.ListRentalBlockedDatesParams{
		PropertyID:      propertyID,
		ExceptionDate:   pgtype.Date{Time: startDate, Valid: true},
		ExceptionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("list rental blocked dates: %w", err)
	}
	return rows, nil
}

func (r *repository) GetPrimaryRentalAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	agentID, err := r.queries.GetPrimaryRentalAgentForProperty(ctx, propertyID)
	if err != nil {
		return 0, fmt.Errorf("get primary rental agent for property: %w", err)
	}
	return agentID, nil
}

func (r *repository) CreateRentalTransaction(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error) {
	record, err := r.queries.CreateRentalTransaction(ctx, arg)
	if err != nil {
		return sqlcgen.Transaction{}, fmt.Errorf("create rental transaction: %w", err)
	}
	return record, nil
}

func (r *repository) UpdateRentalPropertyStatus(ctx context.Context, propertyID int32, statusID int32) error {
	if err := r.queries.UpdateRentalPropertyStatus(ctx, sqlcgen.UpdateRentalPropertyStatusParams{
		PropertyID: propertyID,
		StatusID:   statusID,
	}); err != nil {
		return fmt.Errorf("update rental property status: %w", err)
	}
	return nil
}

func (r *repository) CreateRentalPropertyStatusHistory(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error {
	if err := r.queries.CreateRentalPropertyStatusHistory(ctx, arg); err != nil {
		return fmt.Errorf("create rental property status history: %w", err)
	}
	return nil
}

func (r *repository) UpdateRentalTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error {
	if err := r.queries.UpdateRentalTransactionStatus(ctx, sqlcgen.UpdateRentalTransactionStatusParams{
		TransactionID: transactionID,
		StatusID:      statusID,
	}); err != nil {
		return fmt.Errorf("update rental transaction status: %w", err)
	}
	return nil
}
