package payments

import (
	"context"
	"errors"
	"fmt"
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
	GetUserRole(ctx context.Context, userID int32) (int32, error)
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

func (r *repository) GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error) {
	return r.queries.GetPaymentByUUID(ctx, pgtype.UUID{Bytes: paymentUUID, Valid: true})
}

func (r *repository) GetPaymentByGatewayID(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error) {
	return r.queries.GetPaymentByGatewayID(ctx, pgtype.Text{String: gatewayID, Valid: true})
}

func (r *repository) GetLastPaidPeriod(ctx context.Context, contractID int32) (pgtype.Date, error) {
	return r.queries.GetLastPaidPeriod(ctx, contractID)
}

func (r *repository) GetPendingPayments(ctx context.Context, contractID int32) ([]sqlcgen.GetPendingPaymentsRow, error) {
	return r.queries.GetPendingPayments(ctx, contractID)
}

func (r *repository) UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error {
	return r.queries.UpdatePaymentStatus(ctx, arg)
}

func (r *repository) ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error) {
	rows, err := r.queries.ListPayments(ctx, sqlcgen.ListPaymentsParams{
		PropertyID: int4FromPointer(input.PropertyID),
		StatusID:   int4FromPointer(input.StatusID),
		DateFrom:   dateFromPointer(input.DateFrom),
		DateTo:     dateFromPointer(input.DateTo),
		RoleID:     roleID,
		UserID:     userID,
		PageOffset: input.Offset,
		PageLimit:  input.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}

	items := make([]PaymentListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, PaymentListItem{
			PaymentID:     row.PaymentID,
			ContractID:    row.ContractID,
			PropertyID:    row.PropertyID,
			BillingPeriod: formatDate(row.BillingPeriod.Time),
			DueDate:       formatDate(row.DueDate.Time),
			Amount:        row.Amount,
			Currency:      row.Currency,
			PaymentMethod: row.PaymentMethod,
			Gateway:       textPointer(row.Gateway),
			Status:        row.Status,
			PaymentDate:   timestamptzPointer(row.PaymentDate),
			TotalCount:    row.TotalCount,
		})
	}

	return items, nil
}

func (r *repository) GetPaymentByID(ctx context.Context, paymentID int32) (PaymentDetail, error) {
	row, err := r.queries.GetPaymentByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentDetail{}, ErrPaymentNotFound
		}
		return PaymentDetail{}, fmt.Errorf("get payment by id: %w", err)
	}

	return PaymentDetail{
		PaymentID:       row.PaymentID,
		ContractID:      row.ContractID,
		PropertyID:      row.PropertyID,
		TransactionID:   row.TransactionID,
		TransactionType: row.TransactionType,
		BillingPeriod:   formatDate(row.BillingPeriod.Time),
		DueDate:         formatDate(row.DueDate.Time),
		AgreedAmount:    row.AgreedAmount,
		Amount:          row.Amount,
		Currency:        row.Currency,
		PaymentMethod:   row.PaymentMethod,
		Gateway:         textPointer(row.Gateway),
		Status:          row.Status,
		PaymentDate:     timestamptzPointer(row.PaymentDate),
		ClientID:        row.ClientID,
		AgentID:         row.AgentID,
	}, nil
}

func (r *repository) GetUserRole(ctx context.Context, userID int32) (int32, error) {
	roleID, err := r.queries.GetUserRole(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user role: %w", err)
	}

	return roleID, nil
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
