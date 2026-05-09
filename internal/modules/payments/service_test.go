package payments

import (
	"context"
	"errors"
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

func (m *MockRepository) GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error) {
	args := m.Called(ctx, paymentUUID)
	return args.Get(0).(sqlcgen.GetPaymentByUUIDRow), args.Error(1)
}

func (m *MockRepository) UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockRepository) ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error) {
	args := m.Called(ctx, userID, roleID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]PaymentListItem), args.Error(1)
}

func (m *MockRepository) GetPaymentByID(ctx context.Context, paymentID int32) (PaymentDetail, error) {
	args := m.Called(ctx, paymentID)
	return args.Get(0).(PaymentDetail), args.Error(1)
}

func (m *MockRepository) GetUserRole(ctx context.Context, userID int32) (int32, error) {
	args := m.Called(ctx, userID)
	return int32(args.Int(0)), args.Error(1)
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

// --- UC-16 & UC-17 Tests (Processing) ---

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
		AgreedAmount: pgtype.Numeric{Int: big.NewInt(150000), Exp: -2, Valid: true},
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

<<<<<<< HEAD
func bigNewInt(v int64) *big.Int {
	return big.NewInt(v)
}

=======
>>>>>>> origin/main
func TestConfirmPendingPayment_Success(t *testing.T) {
	repo := new(MockRepository)
	service := NewService(repo)
	ctx := context.Background()
	pUUID := uuid.New()
	clientID := int32(100)

	repo.On("GetPaymentByUUID", ctx, pUUID).Return(sqlcgen.GetPaymentByUUIDRow{
		PaymentID: 100,
		ClientID:  clientID,
		StatusID:  PaymentStatusPending,
		DueDate:   pgtype.Date{Time: time.Now().Add(1 * time.Hour), Valid: true},
	}, nil)

	repo.On("UpdatePaymentStatus", ctx, mock.Anything).Return(nil)

	err := service.ConfirmPendingPayment(ctx, clientID, pUUID)

	assert.NoError(t, err)
}

// --- UC-17 Tests (Consulting) ---

func TestService_ListPayments_GetUserRoleFails(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(0, errors.New("role lookup failed"))

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list payments:")
}

func TestService_ListPayments_UnsupportedRole(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(99, nil)

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	assert.ErrorIs(t, err, ErrUnsupportedRole)
}

func TestService_ListPayments_RepositoryListFails(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	input := ListPaymentsInput{Limit: 20, Offset: 0}

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(int(roleAdminID), nil)
	repo.On("ListPayments", mock.Anything, int32(7), roleAdminID, input).Return(nil, errors.New("db down"))

	_, err := svc.ListPayments(context.Background(), 7, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list payments:")
}

func TestService_ListPayments_EmptyResult(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	input := ListPaymentsInput{Limit: 20, Offset: 0}

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(int(roleAdminID), nil)
	repo.On("ListPayments", mock.Anything, int32(7), roleAdminID, input).Return([]PaymentListItem{}, nil)

	result, err := svc.ListPayments(context.Background(), 7, input)
	assert.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.Pagination.Total)
}

func TestService_ListPayments_TotalFromFirstItem(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	input := ListPaymentsInput{Limit: 20, Offset: 0}

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(int(roleAdminID), nil)
	repo.On("ListPayments", mock.Anything, int32(7), roleAdminID, input).Return([]PaymentListItem{
		{PaymentID: 1, TotalCount: 84},
		{PaymentID: 2, TotalCount: 84},
	}, nil)

	result, err := svc.ListPayments(context.Background(), 7, input)
	assert.NoError(t, err)
	assert.Equal(t, int64(84), result.Pagination.Total)
}

func TestService_GetPaymentByID_PaymentNotFound(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)

	repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{}, ErrPaymentNotFound)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	assert.ErrorIs(t, err, ErrPaymentNotFound)
}

func TestService_GetPaymentByID_AdminAccessesForeignPayment(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)

	repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 99}, nil)
	repo.On("GetUserRole", mock.Anything, int32(1)).Return(int(roleAdminID), nil)

	result, err := svc.GetPaymentByID(context.Background(), 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), result.PaymentID)
}

func TestService_GetPaymentByID_AgentAccessesForeignPayment(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)

	repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1, ClientID: 3, AgentID: 99}, nil)
	repo.On("GetUserRole", mock.Anything, int32(7)).Return(int(roleAgentID), nil)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	assert.ErrorIs(t, err, ErrPaymentForbidden)
}
