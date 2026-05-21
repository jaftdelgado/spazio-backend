package payments

import (
	"context"
	"time"

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
	GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error)
	GetPaymentByGatewayID(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error)
	GetLastPaidPeriod(ctx context.Context, contractID int32) (pgtype.Date, error)
	GetPendingPayments(ctx context.Context, contractID int32) ([]sqlcgen.GetPendingPaymentsRow, error)
	UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error
	WithTx(tx pgx.Tx) Repository
	Begin(ctx context.Context) (pgx.Tx, error)

	ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error)
	GetPaymentByID(ctx context.Context, paymentID int32) (PaymentDetail, error)

	CountCompletedPaymentsForContract(ctx context.Context, contractID int32) (int64, error)
	UpdateTransactionStatusByContract(ctx context.Context, contractID int32, statusID int32) error
	UpdatePropertyStatusByContract(ctx context.Context, contractID int32, statusID int32) error
	UpdateContractStatus(ctx context.Context, contractID int32, statusID int32) error
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

func int4FromPointer(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func dateFromPointer(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: value.UTC(), Valid: true}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func timestamptzPointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time.UTC()
	return &timestamp
}

func formatDate(value time.Time) string {
	return value.UTC().Format("2006-01-02")
}
