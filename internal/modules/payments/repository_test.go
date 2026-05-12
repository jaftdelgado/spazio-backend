package payments

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDBTX struct {
	mock.Mock
}

func (m *MockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgconn.CommandTag), a.Error(1)
}

func (m *MockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	a := m.Called(ctx, query, args)
	if a.Get(0) == nil {
		return nil, a.Error(1)
	}
	return a.Get(0).(pgx.Rows), a.Error(1)
}

func (m *MockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgx.Row)
}

type MockRows struct {
	pgx.Rows
	mock.Mock
}

func (m *MockRows) Next() bool {
	return m.Called().Bool(0)
}
func (m *MockRows) Scan(dest ...interface{}) error {
	return m.Called(dest).Error(0)
}
func (m *MockRows) Close()     {}
func (m *MockRows) Err() error { return nil }

type MockRow struct {
	pgx.Row
	mock.Mock
}

func (m *MockRow) Scan(dest ...interface{}) error {
	return m.Called(dest).Error(0)
}

func TestRepository_Methods(t *testing.T) {
	db := new(MockDBTX)
	repo := &repository{
		db:      nil,
		queries: sqlcgen.New(db),
	}

	ctx := context.Background()

	t.Run("GetPaymentByContract", func(t *testing.T) {
		rows := new(MockRows)
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything).Return(rows, nil).Once()
		rows.On("Next").Return(true).Once()
		rows.On("Scan", mock.Anything).Return(nil).Once()
		rows.On("Next").Return(false).Once()
		_, _ = repo.GetPaymentByContract(ctx, 1, 1)
	})

	t.Run("CreatePayment", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything).Return(nil).Once()
		_, _ = repo.CreatePayment(ctx, sqlcgen.CreatePaymentParams{})
	})

	t.Run("GetContractForPayment", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything).Return(nil).Once()
		_, _ = repo.GetContractForPayment(ctx, 1)
	})

	t.Run("GetContractForPaymentWithLock", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything).Return(nil).Once()
		_, _ = repo.GetContractForPaymentWithLock(ctx, 1)
	})

	t.Run("GetPaymentByUUID", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything).Return(nil).Once()
		_, _ = repo.GetPaymentByUUID(ctx, [16]byte{})
	})

	t.Run("GetPaymentByGatewayID", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything).Return(nil).Once()
		_, _ = repo.GetPaymentByGatewayID(ctx, "id")
	})

	t.Run("GetLastPaidPeriod", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything).Return(nil).Once()
		_, _ = repo.GetLastPaidPeriod(ctx, 1)
	})

	t.Run("GetPendingPayments", func(t *testing.T) {
		rows := new(MockRows)
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything).Return(rows, nil).Once()
		rows.On("Next").Return(false).Once()
		_, _ = repo.GetPendingPayments(ctx, 1)
	})

	t.Run("UpdatePaymentStatus", func(t *testing.T) {
		db.On("Exec", ctx, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil).Once()
		_ = repo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{})
	})

	t.Run("ListPayments", func(t *testing.T) {
		rows := new(MockRows)
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(rows, nil).Once()
		rows.On("Next").Return(true).Once()
		rows.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		rows.On("Next").Return(false).Once()
		_, _ = repo.ListPayments(ctx, 1, 1, ListPaymentsInput{})
	})

	t.Run("GetPaymentByID", func(t *testing.T) {
		row := new(MockRow)
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything).Return(row).Once()
		row.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		_, _ = repo.GetPaymentByID(ctx, 1)
	})

	t.Run("Misc Wrapper", func(t *testing.T) {
		_ = repo.WithTx(nil)
	})

	t.Run("Pointers", func(t *testing.T) {
		int4FromPointer(nil)
		v := int32(1)
		int4FromPointer(&v)

		dateFromPointer(nil)
		dateFromPointer(&time.Time{})

		textPointer(pgtype.Text{Valid: false})
		textPointer(pgtype.Text{Valid: true, String: "s"})

		timestamptzPointer(pgtype.Timestamptz{Valid: false})
		timestamptzPointer(pgtype.Timestamptz{Valid: true})

		formatDate(time.Now())
	})
}

func TestNewModule(t *testing.T) {
	m := NewModule(&pgxpool.Pool{}, "token", "secret")
	assert.NotNil(t, m)
	r := gin.New()
	m.RegisterRoutes(r.Group("/test"), r.Group("/test"))
}
