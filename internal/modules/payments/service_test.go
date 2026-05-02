package payments

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetPaymentByContract(ctx context.Context, contractID int32, statusID int32) ([]sqlcgen.Payment, error) {
	args := m.Called(ctx, contractID, statusID)
	return args.Get(0).([]sqlcgen.Payment), args.Error(1)
}

func (m *MockRepository) CreatePayment(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlcgen.Payment), args.Error(1)
}

func (m *MockRepository) GetContractForPayment(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error) {
	args := m.Called(ctx, contractID)
	return args.Get(0).(sqlcgen.GetContractForPaymentRow), args.Error(1)
}

func (m *MockRepository) GetContractForPaymentWithLock(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
	args := m.Called(ctx, contractID)
	return args.Get(0).(sqlcgen.GetContractForPaymentWithLockRow), args.Error(1)
}

func (m *MockRepository) GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.Payment, error) {
	args := m.Called(ctx, paymentUUID)
	return args.Get(0).(sqlcgen.Payment), args.Error(1)
}

func (m *MockRepository) UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockRepository) WithTx(tx pgx.Tx) Repository {
	return m
}

func (m *MockRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(pgx.Tx), args.Error(1)
}

type MockTx struct {
	pgx.Tx
	mock.Mock
}

func (m *MockTx) Rollback(ctx context.Context) error { return nil }
func (m *MockTx) Commit(ctx context.Context) error   { return nil }

func TestProcessPayment_Success(t *testing.T) {
	repo := new(MockRepository)
	service := NewService(repo)
	ctx := context.Background()
	clientID := int32(10)
	tx := new(MockTx)

	req := RegisterPaymentRequest{
		ContractID:      1,
		PaymentMethodID: 1,
		Amount:          1500.00,
	}

	repo.On("Begin", ctx).Return(tx, nil)
	repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
		ContractID:   1,
		ClientID:     clientID,
		AgreedAmount: pgtype.Numeric{Int: bigNewInt(150000), Exp: -2, Valid: true},
		StatusID:     1,
	}, nil)

	repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil)
	repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusPending)).Return([]sqlcgen.Payment{}, nil)

	repo.On("CreatePayment", mock.Anything, mock.Anything).Return(sqlcgen.Payment{
		PaymentUuid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
		StatusID:         PaymentStatusCompleted,
		GatewayPaymentID: pgtype.Text{String: "MOCK-123", Valid: true},
	}, nil)

	res, err := service.ProcessPayment(ctx, clientID, req)

	assert.NoError(t, err)
	assert.Equal(t, "Success", res.Status)
}

func TestProcessPayment_ContractCancelled(t *testing.T) {
	repo := new(MockRepository)
	service := NewService(repo)
	ctx := context.Background()
	tx := new(MockTx)

	req := RegisterPaymentRequest{ContractID: 1}

	repo.On("Begin", ctx).Return(tx, nil)
	repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
		StatusID: ContractStatusCancelled,
	}, nil)

	_, err := service.ProcessPayment(ctx, 10, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ya está cancelado")
}

func bigNewInt(v int64) *big.Int {
	return big.NewInt(v)
}

func TestConfirmPendingPayment_Success(t *testing.T) {
	repo := new(MockRepository)
	service := NewService(repo)
	ctx := context.Background()
	pUUID := uuid.New()
	clientID := int32(100)

	repo.On("GetPaymentByUUID", ctx, pUUID).Return(sqlcgen.Payment{
		PaymentID: 100,
		ClientID:  clientID,
		StatusID:  PaymentStatusPending,
		DueDate:   pgtype.Date{Time: time.Now().Add(1 * time.Hour), Valid: true},
	}, nil)

	repo.On("UpdatePaymentStatus", ctx, mock.Anything).Return(nil)

	err := service.ConfirmPendingPayment(ctx, clientID, pUUID)

	assert.NoError(t, err)
}
