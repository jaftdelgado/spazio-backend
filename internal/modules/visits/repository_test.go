package visits

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestRepository_Methods(t *testing.T) {
	db := new(MockDBTX)
	repo := &repository{
		db:      nil,
		queries: sqlcgen.New(db),
	}

	ctx := context.Background()

	t.Run("GetPrimaryAgentForProperty", func(t *testing.T) {
		db.On("QueryRow", ctx, mock.Anything, mock.Anything).Return(new(MockRow)).Once()
		_, _ = repo.GetPrimaryAgentForProperty(ctx, 1)
	})

	t.Run("GetAgentSchedule", func(t *testing.T) {
		db.On("Query", ctx, mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
		_, _ = repo.GetAgentSchedule(ctx, 1)
	})

	t.Run("GetPropertyExceptions", func(t *testing.T) {
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
		_, _ = repo.GetPropertyExceptions(ctx, 1, time.Now(), time.Now())
	})

	t.Run("GetOccupiedVisits", func(t *testing.T) {
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
		_, _ = repo.GetOccupiedVisits(ctx, 1, time.Now(), time.Now())
	})

	t.Run("GetOccupiedVisits_Success", func(t *testing.T) {
		rows := new(MockRows)
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(rows, nil).Once()
		rows.On("Next").Return(true).Once()
		rows.On("Scan", mock.Anything).Return(nil).Once()
		rows.On("Next").Return(false).Once()
		_, _ = repo.GetOccupiedVisits(ctx, 1, time.Now(), time.Now())
	})

	t.Run("CreateVisit", func(t *testing.T) {
		db.On("QueryRow", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(new(MockRow)).Once()
		_, _ = repo.CreateVisit(ctx, sqlcgen.CreateVisitParams{})
	})

	t.Run("GetVisitByUUID", func(t *testing.T) {
		db.On("QueryRow", ctx, mock.Anything, mock.Anything).Return(new(MockRow)).Once()
		_, _ = repo.GetVisitByUUID(ctx, uuid.New())
	})

	t.Run("GetUserRole", func(t *testing.T) {
		db.On("QueryRow", ctx, mock.Anything, mock.Anything).Return(new(MockRow)).Once()
		_, _ = repo.GetUserRole(ctx, 1)
	})

	t.Run("ListVisits", func(t *testing.T) {
		db.On("Query", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
		_, _ = repo.ListVisits(ctx, sqlcgen.ListVisitsParams{})
	})

	t.Run("UpdateVisitStatus", func(t *testing.T) {
		db.On("Exec", ctx, mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil).Once()
		_ = repo.UpdateVisitStatus(ctx, 1, 1)
	})

	t.Run("CreateVisitStatusHistory", func(t *testing.T) {
		db.On("Exec", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil).Once()
		_ = repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{})
	})

	t.Run("GetPropertyStatusAndCheckDeleted", func(t *testing.T) {
		db.On("QueryRow", ctx, mock.Anything, mock.Anything).Return(new(MockRow)).Once()
		_, _ = repo.GetPropertyStatusAndCheckDeleted(ctx, 1)
	})

	t.Run("CheckUserActive", func(t *testing.T) {
		db.On("QueryRow", ctx, mock.Anything, mock.Anything).Return(new(MockRow)).Once()
		_, _ = repo.CheckUserActive(ctx, 1)
	})

	t.Run("Factory and Trans", func(t *testing.T) {
		repo2 := NewRepository(&pgxpool.Pool{})
		assert.NotNil(t, repo2)
		_ = repo2.WithTx(nil)
	})
}

type MockRow struct {
	pgx.Row
}

func (m *MockRow) Scan(dest ...interface{}) error {
	return errors.New("mock scan fail")
}
