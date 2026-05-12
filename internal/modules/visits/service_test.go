package visits

import (
	"context"
	"errors"
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

func (m *MockRepository) GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	args := m.Called(ctx, propertyID)
	return int32(args.Int(0)), args.Error(1)
}

func (m *MockRepository) GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
	args := m.Called(ctx, agentID)
	return args.Get(0).([]sqlcgen.GetAgentScheduleRow), args.Error(1)
}

func (m *MockRepository) GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
	args := m.Called(ctx, propertyID, start, end)
	return args.Get(0).([]sqlcgen.GetPropertyExceptionsRow), args.Error(1)
}

func (m *MockRepository) GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error) {
	args := m.Called(ctx, agentID, start, end)
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *MockRepository) CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlcgen.Visit), args.Error(1)
}

func (m *MockRepository) GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error) {
	args := m.Called(ctx, visitUUID)
	return args.Get(0).(sqlcgen.Visit), args.Error(1)
}

func (m *MockRepository) ListVisits(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]sqlcgen.ListVisitsRow), args.Error(1)
}

func (m *MockRepository) UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error {
	args := m.Called(ctx, visitID, statusID)
	return args.Error(0)
}

func (m *MockRepository) CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockRepository) GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
	args := m.Called(ctx, propertyID)
	return args.Get(0).(sqlcgen.GetPropertyStatusAndCheckDeletedRow), args.Error(1)
}

func (m *MockRepository) CheckUserActive(ctx context.Context, userID int32) (int32, error) {
	args := m.Called(ctx, userID)
	return int32(args.Int(0)), args.Error(1)
}

func (m *MockRepository) WithTx(tx pgx.Tx) Repository {
	args := m.Called(tx)
	return args.Get(0).(Repository)
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

func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestTranslateError(t *testing.T) {
	errProp := &pgconn.PgError{Code: "23503", ConstraintName: "property_id"}
	assert.Equal(t, "la propiedad seleccionada no existe", translateError(errProp).Error())

	errUser := &pgconn.PgError{Code: "23503", ConstraintName: "client_id"}
	assert.Equal(t, "el usuario involucrado no existe", translateError(errUser).Error())

	errOtherConstraint := &pgconn.PgError{Code: "23503", ConstraintName: "other"}
	assert.Equal(t, "recurso relacionado no encontrado", translateError(errOtherConstraint).Error())

	errUnique := &pgconn.PgError{Code: "23505"}
	assert.Equal(t, "ya existe una visita programada para ese horario", translateError(errUnique).Error())

	otherErr := errors.New("other")
	assert.Equal(t, otherErr, translateError(otherErr))
}

func TestGetAvailableSlots(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	date := time.Date(2024, 10, 10, 0, 0, 0, 0, time.UTC)

	t.Run("No Agent", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(0, errors.New("no agent")).Once()
		slots, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.Nil(t, slots)
		assert.Error(t, err)
	})

	t.Run("DB Error on Schedule", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Once()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{}, errors.New("db error")).Once()
		_, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.Error(t, err)
	})

	t.Run("No schedule for day", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Once()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: 1}, // Monday
		}, nil).Once()
		slots, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.NoError(t, err)
		assert.Empty(t, slots)
	})

	t.Run("DB Error on Exceptions", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Once()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: 4, StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, errors.New("fail")).Once()
		_, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.Error(t, err)
	})

	t.Run("DB Error on Occupied", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Once()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: 4, StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, errors.New("fail")).Once()
		_, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.Error(t, err)
	})

	t.Run("Success with exceptions and occupied", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Once()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{
				DayOfWeek: 4,                                          // Thursday
				StartTime: pgtype.Time{Microseconds: 9 * 3600 * 1e6},  // 09:00
				EndTime:   pgtype.Time{Microseconds: 12 * 3600 * 1e6}, // 12:00
			},
		}, nil).Once()

		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{
			{
				StartTime: pgtype.Time{Microseconds: 10 * 3600 * 1e6, Valid: true},
				EndTime:   pgtype.Time{Microseconds: 11 * 3600 * 1e6, Valid: true},
			},
		}, nil).Once()

		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{
			time.Date(2024, 10, 10, 11, 0, 0, 0, time.UTC),
		}, nil).Once()

		slots, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.NoError(t, err)
		assert.Len(t, slots, 2)
		assert.True(t, slots[0].Available)
		assert.False(t, slots[1].Available)
	})

	t.Run("Exception without times", func(t *testing.T) {
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Once()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{
				DayOfWeek: 4,
				StartTime: pgtype.Time{Microseconds: 9 * 3600 * 1e6},
				EndTime:   pgtype.Time{Microseconds: 10 * 3600 * 1e6},
			},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{
			{StartTime: pgtype.Time{Valid: false}},
		}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()

		slots, err := svc.GetAvailableSlots(ctx, 1, date)
		assert.NoError(t, err)
		assert.Len(t, slots, 0)
	})
}

func TestScheduleVisit(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)

	t.Run("Too early", func(t *testing.T) {
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(24*time.Hour))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "48 horas")
	})

	t.Run("Too late", func(t *testing.T) {
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(40*24*time.Hour))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "30 días")
	})

	t.Run("Client inactive", func(t *testing.T) {
		repo.On("CheckUserActive", ctx, int32(1)).Return(0, errors.New("inactive")).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(72*time.Hour))
		assert.Error(t, err)
	})

	t.Run("Property not exists", func(t *testing.T) {
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{}, errors.New("not found")).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(72*time.Hour))
		assert.Error(t, err)
		assert.Equal(t, "la propiedad no existe", err.Error())
	})

	t.Run("Property deleted", func(t *testing.T) {
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{
			DeletedAt: pgtype.Timestamptz{Valid: true},
		}, nil).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(72*time.Hour))
		assert.Error(t, err)
		assert.Equal(t, "la propiedad ya no está disponible (eliminada)", err.Error())
	})

	t.Run("Property not available", func(t *testing.T) {
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{
			StatusID: 3, // Sold
		}, nil).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(72*time.Hour))
		assert.Error(t, err)
	})

	t.Run("No agent", func(t *testing.T) {
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(0, errors.New("no agent")).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, now.Add(72*time.Hour))
		assert.Error(t, err)
	})

	t.Run("No agent in Internal", func(t *testing.T) {
		visitDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(0, errors.New("no agent")).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, visitDate)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tiene un agente asignado disponible")
	})

	t.Run("Slot not available", func(t *testing.T) {
		visitDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{}, nil).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, visitDate)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "el horario seleccionado ya no está disponible")
	})

	t.Run("Create Visit Fails", func(t *testing.T) {
		visitDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: int16(visitDate.Weekday()), StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()
		repo.On("CreateVisit", ctx, mock.Anything).Return(sqlcgen.Visit{}, errors.New("db fail")).Once()
		_, err := svc.ScheduleVisit(ctx, 1, 1, visitDate)
		assert.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		visitDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()

		// Mock GetAvailableSlots success
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: int16(visitDate.Weekday()), StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()

		repo.On("CreateVisit", ctx, mock.Anything).Return(sqlcgen.Visit{
			VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true},
			StatusID:  1,
		}, nil).Once()

		res, err := svc.ScheduleVisit(ctx, 1, 1, visitDate)
		assert.NoError(t, err)
		assert.Equal(t, "Pending", res.Status)
	})
}

func TestListUserVisits(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("Role Error", func(t *testing.T) {
		_, err := svc.ListUserVisits(ctx, 1, 99, ListVisitsFilter{})
		assert.Error(t, err)
	})

	t.Run("Success Admin with nil fields", func(t *testing.T) {
		repo.On("ListVisits", ctx, mock.Anything).Return([]sqlcgen.ListVisitsRow{
			{
				VisitUuid:  pgtype.UUID{Bytes: uuid.New(), Valid: true},
				StatusName: "Confirmed",
			},
		}, nil).Once()
		res, err := svc.ListUserVisits(ctx, 1, 1, ListVisitsFilter{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB list fails", func(t *testing.T) {
		repo.On("ListVisits", ctx, mock.Anything).Return(nil, errors.New("fail")).Once()
		_, err := svc.ListUserVisits(ctx, 1, 1, ListVisitsFilter{})
		assert.Error(t, err)
	})

	t.Run("Agent list Success", func(t *testing.T) {
		repo.On("ListVisits", ctx, mock.Anything).Return([]sqlcgen.ListVisitsRow{
			{
				VisitUuid:  pgtype.UUID{Bytes: uuid.New(), Valid: true},
				StatusName: "Pending",
				AgentPhone: pgtype.Text{String: "123", Valid: true},
				CityName:   pgtype.Text{String: "City", Valid: true},
				ClientName: "Client",
				AgentName:  "Agent",
				Address:    "Address",
			},
		}, nil).Once()
		res, err := svc.ListUserVisits(ctx, 1, 2, ListVisitsFilter{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Client list Success", func(t *testing.T) {
		st := int32(1)
		pr := int32(1)
		repo.On("ListVisits", ctx, mock.Anything).Return([]sqlcgen.ListVisitsRow{}, nil).Once()
		_, err := svc.ListUserVisits(ctx, 1, 3, ListVisitsFilter{StatusID: &st, PropertyID: &pr, Date: &time.Time{}})
		assert.NoError(t, err)
	})
}

func TestConfirmVisit(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	uID := uuid.New()

	t.Run("Visit not found", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{}, errors.New("not found")).Once()
		err := svc.ConfirmVisit(ctx, 1, 1, uID)
		assert.Error(t, err)
	})

	t.Run("Forbidden Client", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{ClientID: 2}, nil).Once()
		err := svc.ConfirmVisit(ctx, 1, 3, uID)
		assert.Error(t, err)
	})

	t.Run("Success Client Pending", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, ClientID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusWaitingAgent)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.ConfirmVisit(ctx, 1, 3, uID)
		assert.NoError(t, err)
	})

	t.Run("Success Client Waiting Client", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, ClientID: 1, StatusID: StatusWaitingClient}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusConfirmed)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.ConfirmVisit(ctx, 1, 3, uID)
		assert.NoError(t, err)
	})

	t.Run("Success Agent Pending", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, AgentID: pgtype.Int4{Int32: 2, Valid: true}, StatusID: StatusPending}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusWaitingClient)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.ConfirmVisit(ctx, 2, 2, uID)
		assert.NoError(t, err)
	})

	t.Run("Forbidden Agent", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{AgentID: pgtype.Int4{Int32: 2, Valid: true}}, nil).Once()
		err := svc.ConfirmVisit(ctx, 3, 2, uID)
		assert.Error(t, err)
	})

	t.Run("Success Agent Waiting Agent", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, AgentID: pgtype.Int4{Int32: 2, Valid: true}, StatusID: StatusWaitingAgent}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusConfirmed)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.ConfirmVisit(ctx, 2, 2, uID)
		assert.NoError(t, err)
	})

	t.Run("Success Admin", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusConfirmed)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.ConfirmVisit(ctx, 10, 1, uID)
		assert.NoError(t, err)
	})

	t.Run("Invalid transition", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, StatusID: StatusConfirmed}, nil).Once()
		err := svc.ConfirmVisit(ctx, 10, 1, uID)
		assert.Error(t, err)
	})

	t.Run("Update fails", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusConfirmed)).Return(errors.New("fail")).Once()
		err := svc.ConfirmVisit(ctx, 10, 1, uID)
		assert.Error(t, err)
	})
}

func TestRescheduleVisit(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	uID := uuid.New()
	tx := new(MockTx)
	now := time.Now().UTC().Truncate(time.Hour)

	t.Run("Begin fails", func(t *testing.T) {
		repo.On("Begin", ctx).Return(nil, errors.New("fail")).Once()
		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, time.Now())
		assert.Error(t, err)
	})

	t.Run("Visit not found", func(t *testing.T) {
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{}, errors.New("not found")).Once()
		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, time.Now())
		assert.Error(t, err)
	})

	t.Run("Already cancelled", func(t *testing.T) {
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{StatusID: StatusCancelled}, nil).Once()
		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, time.Now())
		assert.Error(t, err)
	})

	t.Run("Forbidden", func(t *testing.T) {
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{StatusID: StatusPending, ClientID: 2}, nil).Once()
		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, time.Now())
		assert.Error(t, err)
	})

	t.Run("Update fails", func(t *testing.T) {
		newDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, ClientID: 1, PropertyID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: int16(newDate.Weekday()), StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()
		repo.On("CreateVisit", ctx, mock.Anything).Return(sqlcgen.Visit{VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true}}, nil).Once()

		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCancelled)).Return(errors.New("fail")).Once()
		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, newDate)
		assert.Error(t, err)
	})

	t.Run("History fails", func(t *testing.T) {
		newDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, ClientID: 1, PropertyID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: int16(newDate.Weekday()), StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()
		repo.On("CreateVisit", ctx, mock.Anything).Return(sqlcgen.Visit{VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true}}, nil).Once()

		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCancelled)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(errors.New("fail")).Once()
		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, newDate)
		assert.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		newDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, ClientID: 1, PropertyID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: int16(newDate.Weekday()), StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()
		repo.On("CreateVisit", ctx, mock.Anything).Return(sqlcgen.Visit{VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true}}, nil).Once()

		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCancelled)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		tx.On("Commit", ctx).Return(nil).Once()

		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, newDate)
		assert.NoError(t, err)
	})

	t.Run("Commit fails", func(t *testing.T) {
		newDate := now.Add(100 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour)
		repo.On("Begin", ctx).Return(tx, nil).Once()
		repo.On("WithTx", tx).Return(repo).Once()
		tx.On("Rollback", ctx).Return(nil).Once()
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, ClientID: 1, PropertyID: 1, StatusID: StatusPending}, nil).Once()
		repo.On("CheckUserActive", ctx, int32(1)).Return(1, nil).Once()
		repo.On("GetPropertyStatusAndCheckDeleted", ctx, int32(1)).Return(sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: 2}, nil).Once()
		repo.On("GetPrimaryAgentForProperty", ctx, int32(1)).Return(10, nil).Twice()
		repo.On("GetAgentSchedule", ctx, int32(10)).Return([]sqlcgen.GetAgentScheduleRow{
			{DayOfWeek: int16(newDate.Weekday()), StartTime: pgtype.Time{Microseconds: 0}, EndTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6}},
		}, nil).Once()
		repo.On("GetPropertyExceptions", ctx, int32(1), mock.Anything, mock.Anything).Return([]sqlcgen.GetPropertyExceptionsRow{}, nil).Once()
		repo.On("GetOccupiedVisits", ctx, int32(10), mock.Anything, mock.Anything).Return([]time.Time{}, nil).Once()
		repo.On("CreateVisit", ctx, mock.Anything).Return(sqlcgen.Visit{VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true}}, nil).Once()

		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCancelled)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		tx.On("Commit", ctx).Return(errors.New("fail")).Once()

		_, err := svc.RescheduleVisit(ctx, 1, 3, uID, newDate)
		assert.Error(t, err)
	})
}

func TestCompleteVisit(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	uID := uuid.New()

	t.Run("Visit not found", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{}, errors.New("not found")).Once()
		err := svc.CompleteVisit(ctx, 1, 1, uID)
		assert.Error(t, err)
	})

	t.Run("Forbidden Client", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{}, nil).Once()
		err := svc.CompleteVisit(ctx, 1, 3, uID)
		assert.Error(t, err)
	})

	t.Run("Forbidden Agent", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{AgentID: pgtype.Int4{Int32: 2, Valid: true}}, nil).Once()
		err := svc.CompleteVisit(ctx, 3, 2, uID)
		assert.Error(t, err)
	})

	t.Run("Not confirmed", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{StatusID: StatusPending}, nil).Once()
		err := svc.CompleteVisit(ctx, 1, 1, uID)
		assert.Error(t, err)
	})

	t.Run("Update fails", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, StatusID: StatusConfirmed}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCompleted)).Return(errors.New("fail")).Once()
		err := svc.CompleteVisit(ctx, 10, 1, uID)
		assert.Error(t, err)
	})

	t.Run("Success Agent", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, AgentID: pgtype.Int4{Int32: 2, Valid: true}, StatusID: StatusConfirmed}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCompleted)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.CompleteVisit(ctx, 2, 2, uID)
		assert.NoError(t, err)
	})

	t.Run("Success Admin", func(t *testing.T) {
		repo.On("GetVisitByUUID", ctx, uID).Return(sqlcgen.Visit{VisitID: 1, StatusID: StatusConfirmed}, nil).Once()
		repo.On("UpdateVisitStatus", ctx, int32(1), int32(StatusCompleted)).Return(nil).Once()
		repo.On("CreateVisitStatusHistory", ctx, mock.Anything).Return(nil).Once()
		err := svc.CompleteVisit(ctx, 10, 1, uID)
		assert.NoError(t, err)
	})
}
