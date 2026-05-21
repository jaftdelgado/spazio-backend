package payments

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

// mockPaymentRepository is a manual mock implementation of the Repository interface.
type mockPaymentRepository struct {
	getPaymentByContractFunc           func(ctx context.Context, contractID int32, statusID int32) ([]sqlcgen.Payment, error)
	createPaymentFunc                  func(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error)
	getContractForPaymentFunc          func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error)
	getContractForPaymentWithLockFunc  func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error)
	getPaymentByUUIDFunc               func(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error)
	getPaymentByGatewayIDFunc          func(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error)
	getLastPaidPeriodFunc              func(ctx context.Context, contractID int32) (pgtype.Date, error)
	getPendingPaymentsFunc             func(ctx context.Context, contractID int32) ([]sqlcgen.GetPendingPaymentsRow, error)
	updatePaymentStatusFunc            func(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error
	withTxFunc                         func(tx pgx.Tx) Repository
	beginFunc                          func(ctx context.Context) (pgx.Tx, error)
	listPaymentsFunc                   func(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error)
	getPaymentByIDFunc                 func(ctx context.Context, paymentID int32) (PaymentDetail, error)
	countCompletedPaymentsForContractFunc func(ctx context.Context, contractID int32) (int64, error)
	updateTransactionStatusByContractFunc func(ctx context.Context, contractID int32, statusID int32) error
	updatePropertyStatusByContractFunc    func(ctx context.Context, contractID int32, statusID int32) error
	updateContractStatusFunc              func(ctx context.Context, contractID int32, statusID int32) error
}

func (m *mockPaymentRepository) GetPaymentByContract(ctx context.Context, contractID int32, statusID int32) ([]sqlcgen.Payment, error) {
	if m.getPaymentByContractFunc != nil {
		return m.getPaymentByContractFunc(ctx, contractID, statusID)
	}
	return nil, nil
}

func (m *mockPaymentRepository) CreatePayment(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error) {
	if m.createPaymentFunc != nil {
		return m.createPaymentFunc(ctx, arg)
	}
	return sqlcgen.Payment{}, nil
}

func (m *mockPaymentRepository) GetContractForPayment(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error) {
	if m.getContractForPaymentFunc != nil {
		return m.getContractForPaymentFunc(ctx, contractID)
	}
	return sqlcgen.GetContractForPaymentRow{}, nil
}

func (m *mockPaymentRepository) GetContractForPaymentWithLock(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
	if m.getContractForPaymentWithLockFunc != nil {
		return m.getContractForPaymentWithLockFunc(ctx, contractID)
	}
	return sqlcgen.GetContractForPaymentWithLockRow{}, nil
}

func (m *mockPaymentRepository) GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error) {
	if m.getPaymentByUUIDFunc != nil {
		return m.getPaymentByUUIDFunc(ctx, paymentUUID)
	}
	return sqlcgen.GetPaymentByUUIDRow{}, nil
}

func (m *mockPaymentRepository) GetPaymentByGatewayID(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error) {
	if m.getPaymentByGatewayIDFunc != nil {
		return m.getPaymentByGatewayIDFunc(ctx, gatewayID)
	}
	return sqlcgen.GetPaymentByGatewayIDRow{}, nil
}

func (m *mockPaymentRepository) GetLastPaidPeriod(ctx context.Context, contractID int32) (pgtype.Date, error) {
	if m.getLastPaidPeriodFunc != nil {
		return m.getLastPaidPeriodFunc(ctx, contractID)
	}
	return pgtype.Date{}, nil
}

func (m *mockPaymentRepository) GetPendingPayments(ctx context.Context, contractID int32) ([]sqlcgen.GetPendingPaymentsRow, error) {
	if m.getPendingPaymentsFunc != nil {
		return m.getPendingPaymentsFunc(ctx, contractID)
	}
	return nil, nil
}

func (m *mockPaymentRepository) UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error {
	if m.updatePaymentStatusFunc != nil {
		return m.updatePaymentStatusFunc(ctx, arg)
	}
	return nil
}

func (m *mockPaymentRepository) WithTx(tx pgx.Tx) Repository {
	if m.withTxFunc != nil {
		return m.withTxFunc(tx)
	}
	return m
}

func (m *mockPaymentRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.beginFunc != nil {
		return m.beginFunc(ctx)
	}
	return nil, nil
}

func (m *mockPaymentRepository) ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error) {
	if m.listPaymentsFunc != nil {
		return m.listPaymentsFunc(ctx, userID, roleID, input)
	}
	return nil, nil
}

func (m *mockPaymentRepository) GetPaymentByID(ctx context.Context, paymentID int32) (PaymentDetail, error) {
	if m.getPaymentByIDFunc != nil {
		return m.getPaymentByIDFunc(ctx, paymentID)
	}
	return PaymentDetail{}, nil
}

func (m *mockPaymentRepository) CountCompletedPaymentsForContract(ctx context.Context, contractID int32) (int64, error) {
	if m.countCompletedPaymentsForContractFunc != nil {
		return m.countCompletedPaymentsForContractFunc(ctx, contractID)
	}
	return 0, nil
}

func (m *mockPaymentRepository) UpdateTransactionStatusByContract(ctx context.Context, contractID int32, statusID int32) error {
	if m.updateTransactionStatusByContractFunc != nil {
		return m.updateTransactionStatusByContractFunc(ctx, contractID, statusID)
	}
	return nil
}

func (m *mockPaymentRepository) UpdatePropertyStatusByContract(ctx context.Context, contractID int32, statusID int32) error {
	if m.updatePropertyStatusByContractFunc != nil {
		return m.updatePropertyStatusByContractFunc(ctx, contractID, statusID)
	}
	return nil
}

func (m *mockPaymentRepository) UpdateContractStatus(ctx context.Context, contractID int32, statusID int32) error {
	if m.updateContractStatusFunc != nil {
		return m.updateContractStatusFunc(ctx, contractID, statusID)
	}
	return nil
}

type mockTx struct {
	pgx.Tx
	commitFunc   func(ctx context.Context) error
	rollbackFunc func(ctx context.Context) error
}

func (m *mockTx) Commit(ctx context.Context) error {
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}
	return nil
}

func (m *mockTx) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}
	return nil
}
