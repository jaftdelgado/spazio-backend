package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (m *MockRepository) GetPaymentByGatewayID(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error) {
	args := m.Called(ctx, gatewayID)
	return args.Get(0).(sqlcgen.GetPaymentByGatewayIDRow), args.Error(1)
}

func (m *MockRepository) GetLastPaidPeriod(ctx context.Context, contractID int32) (pgtype.Date, error) {
	args := m.Called(ctx, contractID)
	return args.Get(0).(pgtype.Date), args.Error(1)
}

func (m *MockRepository) GetPendingPayments(ctx context.Context, contractID int32) ([]sqlcgen.GetPendingPaymentsRow, error) {
	args := m.Called(ctx, contractID)
	return args.Get(0).([]sqlcgen.GetPendingPaymentsRow), args.Error(1)
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
	return args.Get(0).(int32), args.Error(1)
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
func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestProcessPayment_Full(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()
	tx := new(MockTx)
	clientID := int32(10)

	baseReq := RegisterPaymentRequest{
		ContractID:      1,
		PaymentMethodID: 1,
		Amount:          1000.0,
		Currency:        "MXN",
		PayerEmail:      "test@test.com",
		GatewayMethodID: "visa",
	}

	t.Run("Begin Error", func(t *testing.T) {
		svc := NewService(repo, "T", "S")
		repo.On("Begin", ctx).Return(nil, errors.New("fail")).Once()
		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
	})

	t.Run("Get Contract Error", func(t *testing.T) {
		svc := NewService(repo, "T", "S")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{}, errors.New("fail")).Once()
		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
	})

	t.Run("Rejected by Gateway", func(t *testing.T) {
		svc := NewService(repo, "TEST-REJECTED", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rechazado por la pasarela")
	})

	t.Run("Success Pending (OXXO)", func(t *testing.T) {
		svc := NewService(repo, "TEST-PENDING", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()

		repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(p sqlcgen.CreatePaymentParams) bool {
			return p.StatusID == PaymentStatusPending
		})).Return(sqlcgen.Payment{
			PaymentUuid:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
			StatusID:         PaymentStatusPending,
			GatewayPaymentID: pgtype.Text{String: "123", Valid: true},
		}, nil).Once()
		tx.On("Commit", ctx).Return(nil).Once()

		res, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.NoError(t, err)
		assert.Equal(t, "Pending", res.Status)
		assert.NotNil(t, res.ReferenceNumber)
	})

	t.Run("Translate DB Error (FK)", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()

		repo.On("CreatePayment", mock.Anything, mock.Anything).Return(sqlcgen.Payment{}, &pgconn.PgError{Code: "23503"}).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Equal(t, "el contrato o método de pago seleccionado no existe", err.Error())
	})

	t.Run("Translate DB Error (Other)", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "S")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()
		repo.On("CreatePayment", mock.Anything, mock.Anything).Return(sqlcgen.Payment{}, &pgconn.PgError{Code: "99"}).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
	})

	t.Run("Contract Blocked", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			StatusID: ContractStatusBlocked,
		}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "está bloqueado")
	})

	t.Run("Contract Terminated", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			StatusID: ContractStatusTerminated,
		}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ha sido terminado")
	})

	t.Run("Contract Expired", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			EndDate: pgtype.Date{Time: time.Now().Add(-24 * time.Hour), Valid: true},
		}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vencimiento del contrato ha pasado")
	})

	t.Run("Unauthorized User", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: 999, // Different
		}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no pertenece al usuario")
	})

	t.Run("Currency Mismatch", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "USD",
		}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no coincide con la moneda del contrato")
	})

	t.Run("Already Paid Period (Rent)", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN", TransactionType: "rent",
		}, nil).Once()

		billingPeriod := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		repo.On("GetLastPaidPeriod", mock.Anything, int32(1)).Return(pgtype.Date{Valid: false}, nil).Once() // No prev payments, current month calculated

		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{
			{BillingPeriod: pgtype.Date{Time: billingPeriod, Valid: true}, PaymentUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true}},
		}, nil).Once()

		res, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.NoError(t, err)
		assert.Equal(t, "Already Paid", res.Status)
	})

	t.Run("Cancel Pending Prev Success", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()

		// Pending exists
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{
			{PaymentID: 55},
		}, nil).Once()
		repo.On("UpdatePaymentStatus", mock.Anything, mock.MatchedBy(func(p sqlcgen.UpdatePaymentStatusParams) bool {
			return p.PaymentID == 55 && p.StatusID == PaymentStatusFailed
		})).Return(nil).Once()

		repo.On("CreatePayment", mock.Anything, mock.Anything).Return(sqlcgen.Payment{
			PaymentUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true},
			StatusID:    PaymentStatusCompleted,
		}, nil).Once()
		tx.On("Commit", ctx).Return(nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.NoError(t, err)
	})

	t.Run("Amount Mismatch", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "SECRET")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(500000), Exp: -2, Valid: true}, // 5000.00
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq) // req has 1000.0
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no coincide con el monto pactado")
	})

	t.Run("Commit Error", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "S")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()
		repo.On("CreatePayment", mock.Anything, mock.Anything).Return(sqlcgen.Payment{}, nil).Once()
		tx.On("Commit", ctx).Return(errors.New("fail")).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.Error(t, err)
	})

	t.Run("Rent Next Period", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "S")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN", TransactionType: "rent",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
		}, nil).Once()

		lastPaid := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		repo.On("GetLastPaidPeriod", mock.Anything, int32(1)).Return(pgtype.Date{Time: lastPaid, Valid: true}, nil).Once()

		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()
		repo.On("CreatePayment", mock.Anything, mock.MatchedBy(func(p sqlcgen.CreatePaymentParams) bool {
			return p.BillingPeriod.Time.Month() == time.February
		})).Return(sqlcgen.Payment{StatusID: 2}, nil).Once()
		tx.On("Commit", ctx).Return(nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.NoError(t, err)
	})

	t.Run("Contract Valid Not Expired", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", "S")
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("GetContractForPaymentWithLock", mock.Anything, int32(1)).Return(sqlcgen.GetContractForPaymentWithLockRow{
			ClientID: clientID, Currency: "MXN",
			AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
			EndDate:      pgtype.Date{Time: time.Now().Add(100 * time.Hour), Valid: true},
		}, nil).Once()
		repo.On("GetPaymentByContract", mock.Anything, int32(1), int32(PaymentStatusCompleted)).Return([]sqlcgen.Payment{}, nil).Once()
		repo.On("GetPendingPayments", mock.Anything, int32(1)).Return([]sqlcgen.GetPendingPaymentsRow{}, nil).Once()
		repo.On("CreatePayment", mock.Anything, mock.Anything).Return(sqlcgen.Payment{StatusID: 2}, nil).Once()
		tx.On("Commit", ctx).Return(nil).Once()

		_, err := svc.ProcessPayment(ctx, clientID, baseReq)
		assert.NoError(t, err)
	})
}

func calculateTestSig(requestID, ts, body, secret string) string {
	manifest := fmt.Sprintf("id:%s;ts:%s;", requestID, ts)
	signedString := manifest + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedString))
	v1 := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("ts=%s,v1=%s", ts, v1)
}

func TestHandleWebhook_Full(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()
	secret := "TEST-SECRET"

	t.Run("Invalid Signature", func(t *testing.T) {
		svc := NewService(repo, "TOKEN", secret)
		err := svc.HandleWebhook(ctx, "ts=123,v1=wrong", "id", []byte(`{}`))
		assert.Error(t, err)
		assert.Equal(t, "invalid signature", err.Error())
	})

	t.Run("Invalid Signature Format", func(t *testing.T) {
		svc := NewService(repo, "TOKEN", secret)
		err := svc.HandleWebhook(ctx, "wrong-format", "id", []byte(`{}`))
		assert.Error(t, err)
	})

	t.Run("Ignore non-payment type", func(t *testing.T) {
		svc := NewService(repo, "TOKEN", secret)
		body := `{"type":"plan"}`
		sig := calculateTestSig("req1", "123", body, secret)
		err := svc.HandleWebhook(ctx, sig, "req1", []byte(body))
		assert.NoError(t, err)
	})

	t.Run("Payment Not Found in DB", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", secret)
		body := `{"type":"payment","data":{"id":"123"}}`
		sig := calculateTestSig("req1", "123", body, secret)

		repo.On("GetPaymentByGatewayID", mock.Anything, "123").Return(sqlcgen.GetPaymentByGatewayIDRow{}, errors.New("not found")).Once()

		err := svc.HandleWebhook(ctx, sig, "req1", []byte(body))
		assert.NoError(t, err)
	})

	t.Run("Already Processed", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", secret)
		body := `{"type":"payment","data":{"id":"123"}}`
		sig := calculateTestSig("req1", "123", body, secret)

		repo.On("GetPaymentByGatewayID", mock.Anything, "123").Return(sqlcgen.GetPaymentByGatewayIDRow{
			StatusID: PaymentStatusCompleted,
		}, nil).Once()

		err := svc.HandleWebhook(ctx, sig, "req1", []byte(body))
		assert.NoError(t, err)
	})

	t.Run("Status Approved", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", secret)
		body := `{"type":"payment","data":{"id":"123"}}`
		sig := calculateTestSig("req1", "123", body, secret)

		repo.On("GetPaymentByGatewayID", mock.Anything, "123").Return(sqlcgen.GetPaymentByGatewayIDRow{
			PaymentID: 1, StatusID: PaymentStatusPending,
		}, nil).Once()

		repo.On("UpdatePaymentStatus", mock.Anything, mock.MatchedBy(func(p sqlcgen.UpdatePaymentStatusParams) bool {
			return p.StatusID == PaymentStatusCompleted
		})).Return(nil).Once()

		err := svc.HandleWebhook(ctx, sig, "req1", []byte(body))
		assert.NoError(t, err)
	})

	t.Run("Status Refunded", func(t *testing.T) {
		svc := NewService(repo, "TEST-REFUNDED", secret)
		body := `{"type":"payment","data":{"id":"123"}}`
		sig := calculateTestSig("req1", "123", body, secret)

		repo.On("GetPaymentByGatewayID", mock.Anything, "123").Return(sqlcgen.GetPaymentByGatewayIDRow{
			PaymentID: 1, StatusID: PaymentStatusCompleted,
		}, nil).Once()

		repo.On("UpdatePaymentStatus", mock.Anything, mock.MatchedBy(func(p sqlcgen.UpdatePaymentStatusParams) bool {
			return p.StatusID == PaymentStatusRefunded
		})).Return(nil).Once()

		err := svc.HandleWebhook(ctx, sig, "req1", []byte(body))
		assert.NoError(t, err)
	})

	t.Run("Status Rejected/Cancelled", func(t *testing.T) {
		svc := NewService(repo, "TEST-REJECTED", secret)
		body := `{"type":"payment","data":{"id":"123"}}`
		sig := calculateTestSig("req1", "123", body, secret)

		repo.On("GetPaymentByGatewayID", mock.Anything, "123").Return(sqlcgen.GetPaymentByGatewayIDRow{
			PaymentID: 1, StatusID: PaymentStatusPending,
		}, nil).Once()

		repo.On("UpdatePaymentStatus", mock.Anything, mock.MatchedBy(func(p sqlcgen.UpdatePaymentStatusParams) bool {
			return p.StatusID == PaymentStatusFailed
		})).Return(nil).Once()

		err := svc.HandleWebhook(ctx, sig, "req1", []byte(body))
		assert.NoError(t, err)
	})

	t.Run("Body Unmarshal Error", func(t *testing.T) {
		svc := NewService(repo, "TOKEN", secret)
		sig := calculateTestSig("r", "1", "invalid", secret)
		err := svc.HandleWebhook(ctx, sig, "r", []byte("invalid"))
		assert.Error(t, err)
	})

	t.Run("Update Status Error", func(t *testing.T) {
		svc := NewService(repo, "TEST-TOKEN", secret)
		body := `{"type":"payment","data":{"id":"123"}}`
		sig := calculateTestSig("r", "1", body, secret)
		repo.On("GetPaymentByGatewayID", mock.Anything, "123").Return(sqlcgen.GetPaymentByGatewayIDRow{PaymentID: 1}, nil).Once()
		repo.On("UpdatePaymentStatus", mock.Anything, mock.Anything).Return(errors.New("fail")).Once()
		err := svc.HandleWebhook(ctx, sig, "r", []byte(body))
		assert.Error(t, err)
	})
}

func TestConfirmPendingPayment_Extended(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TOKEN", "SECRET")
	ctx := context.Background()
	uID := uuid.New()
	clientID := int32(10)

	t.Run("Not Found", func(t *testing.T) {
		repo.On("GetPaymentByUUID", ctx, uID).Return(sqlcgen.GetPaymentByUUIDRow{}, errors.New("not found")).Once()
		err := svc.ConfirmPendingPayment(ctx, clientID, uID)
		assert.Error(t, err)
		assert.Equal(t, "pago no encontrado", err.Error())
	})

	t.Run("Forbidden", func(t *testing.T) {
		repo.On("GetPaymentByUUID", ctx, uID).Return(sqlcgen.GetPaymentByUUIDRow{ClientID: 999}, nil).Once()
		err := svc.ConfirmPendingPayment(ctx, clientID, uID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no te pertenece")
	})

	t.Run("Not Pending", func(t *testing.T) {
		repo.On("GetPaymentByUUID", ctx, uID).Return(sqlcgen.GetPaymentByUUIDRow{ClientID: clientID, StatusID: PaymentStatusCompleted}, nil).Once()
		err := svc.ConfirmPendingPayment(ctx, clientID, uID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "estado pendiente")
	})

	t.Run("Expired", func(t *testing.T) {
		repo.On("GetPaymentByUUID", ctx, uID).Return(sqlcgen.GetPaymentByUUIDRow{
			ClientID: clientID, StatusID: PaymentStatusPending,
			DueDate: pgtype.Date{Time: time.Now().Add(-1 * time.Hour), Valid: true},
		}, nil).Once()
		err := svc.ConfirmPendingPayment(ctx, clientID, uID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ha expirado")
	})
}

func TestService_ListPayments_GetUserRoleFails(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(int32(0), errors.New("role lookup failed"))

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list payments:")
}

func TestService_ListPayments_UnsupportedRole(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(int32(99), nil)

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	assert.ErrorIs(t, err, ErrUnsupportedRole)
}

func TestService_ListPayments_RepositoryListFails(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")
	input := ListPaymentsInput{Limit: 20, Offset: 0}

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(roleAdminID, nil)
	repo.On("ListPayments", mock.Anything, int32(7), roleAdminID, input).Return(nil, errors.New("db down"))

	_, err := svc.ListPayments(context.Background(), 7, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list payments:")
}

func TestService_ListPayments_EmptyResult(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")
	input := ListPaymentsInput{Limit: 20, Offset: 0}

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(roleAdminID, nil)
	repo.On("ListPayments", mock.Anything, int32(7), roleAdminID, input).Return([]PaymentListItem{}, nil)

	result, err := svc.ListPayments(context.Background(), 7, input)
	assert.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.Pagination.Total)
}

func TestService_ListPayments_TotalFromFirstItem(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")
	input := ListPaymentsInput{Limit: 20, Offset: 0}

	repo.On("GetUserRole", mock.Anything, int32(7)).Return(roleAdminID, nil)
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
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")

	repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{}, ErrPaymentNotFound)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	assert.ErrorIs(t, err, ErrPaymentNotFound)
}

func TestService_GetPaymentByID_Roles(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo, "TEST-TOKEN", "TEST-SECRET")
	ctx := context.Background()

	t.Run("Admin Success", func(t *testing.T) {
		repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 99}, nil).Once()
		repo.On("GetUserRole", mock.Anything, int32(1)).Return(roleAdminID, nil).Once()
		_, err := svc.GetPaymentByID(ctx, 1, 1)
		assert.NoError(t, err)
	})

	t.Run("Agent Forbidden", func(t *testing.T) {
		repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 99}, nil).Once()
		repo.On("GetUserRole", mock.Anything, int32(2)).Return(roleAgentID, nil).Once()
		_, err := svc.GetPaymentByID(ctx, 2, 1) // ID 2 != 99
		assert.ErrorIs(t, err, ErrPaymentForbidden)
	})

	t.Run("Client Forbidden", func(t *testing.T) {
		repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 99}, nil).Once()
		repo.On("GetUserRole", mock.Anything, int32(3)).Return(roleClientID, nil).Once()
		_, err := svc.GetPaymentByID(ctx, 3, 1) // ID 3 != 7
		assert.ErrorIs(t, err, ErrPaymentForbidden)
	})

	t.Run("Unsupported Role", func(t *testing.T) {
		repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1}, nil).Once()
		repo.On("GetUserRole", mock.Anything, int32(4)).Return(int32(99), nil).Once()
		_, err := svc.GetPaymentByID(ctx, 4, 1)
		assert.ErrorIs(t, err, ErrUnsupportedRole)
	})

	t.Run("Role Lookup Error", func(t *testing.T) {
		repo.On("GetPaymentByID", mock.Anything, int32(1)).Return(PaymentDetail{PaymentID: 1}, nil).Once()
		repo.On("GetUserRole", mock.Anything, int32(1)).Return(int32(0), errors.New("fail")).Once()
		_, err := svc.GetPaymentByID(ctx, 1, 1)
		assert.Error(t, err)
	})
}
