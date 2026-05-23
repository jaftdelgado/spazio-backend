package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) CountCompletedPaymentsForContract(ctx context.Context, contractID int32) (int64, error) {
	return r.queries.CountCompletedPaymentsForContract(ctx, contractID)
}

func (r *repository) UpdateTransactionStatusByContract(ctx context.Context, contractID int32, statusID int32) error {
	return r.queries.UpdateTransactionStatusByContract(ctx, sqlcgen.UpdateTransactionStatusByContractParams{
		ContractID: contractID,
		StatusID:   statusID,
	})
}

func (r *repository) UpdatePropertyStatusByContract(ctx context.Context, contractID int32, statusID int32) error {
	return r.queries.UpdatePropertyStatusByContract(ctx, sqlcgen.UpdatePropertyStatusByContractParams{
		ContractID: contractID,
		StatusID:   statusID,
	})
}

func (r *repository) UpdateContractStatus(ctx context.Context, contractID int32, statusID int32) error {
	return r.queries.UpdateContractStatus(ctx, sqlcgen.UpdateContractStatusParams{
		ContractID: contractID,
		StatusID:   statusID,
	})
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

	return mapListPaymentsRows(rows), nil
}

func mapListPaymentsRows(rows []sqlcgen.ListPaymentsRow) []PaymentListItem {
	items := make([]PaymentListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, PaymentListItem{
			PaymentUUID:   row.PaymentUuid.Bytes,
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
	return items
}

func (r *repository) GetPaymentDetailByUUID(ctx context.Context, paymentUUID uuid.UUID) (PaymentDetailRecord, error) {
	row, err := r.queries.GetPaymentDetailByUUID(ctx, pgtype.UUID{Bytes: paymentUUID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentDetailRecord{}, ErrPaymentNotFound
		}
		return PaymentDetailRecord{}, fmt.Errorf("get payment detail by uuid: %w", err)
	}

	return mapPaymentDetailRow(row), nil
}

func mapPaymentDetailRow(row sqlcgen.GetPaymentDetailByUUIDRow) PaymentDetailRecord {
	return PaymentDetailRecord{
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
	}
}
