package payments

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type Repository interface {
	GetPaymentByContract(ctx context.Context, contractID int32, statusID int32) ([]sqlcgen.Payment, error)
	CreatePayment(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error)
	GetContractForPayment(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error)
	GetContractForPaymentWithLock(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error)
	GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.Payment, error)
	UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error
	// Transaction support
	WithTx(tx pgx.Tx) Repository
	Begin(ctx context.Context) (pgx.Tx, error)
}

type repository struct {
	db      sqlcgen.DBTX
	queries *sqlcgen.Queries
	pool    *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db:      db,
		queries: sqlcgen.New(db),
		pool:    db,
	}
}

func (r *repository) WithTx(tx pgx.Tx) Repository {
	return &repository{
		db:      tx,
		queries: r.queries.WithTx(tx),
		pool:    r.pool,
	}
}

func (r *repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *repository) GetPaymentByContract(ctx context.Context, contractID int32, statusID int32) ([]sqlcgen.Payment, error) {
	return r.queries.GetPaymentByContract(ctx, sqlcgen.GetPaymentByContractParams{
		ContractID: contractID,
		StatusID:   statusID,
	})
}

func (r *repository) CreatePayment(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error) {
	return r.queries.CreatePayment(ctx, arg)
}

func (r *repository) GetContractForPayment(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error) {
	return r.queries.GetContractForPayment(ctx, contractID)
}

func (r *repository) GetContractForPaymentWithLock(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
	return r.queries.GetContractForPaymentWithLock(ctx, contractID)
}

func (r *repository) GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.Payment, error) {
	return r.queries.GetPaymentByUUID(ctx, pgtype.UUID{Bytes: paymentUUID, Valid: true})
}

func (r *repository) UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error {
	return r.queries.UpdatePaymentStatus(ctx, arg)
}
