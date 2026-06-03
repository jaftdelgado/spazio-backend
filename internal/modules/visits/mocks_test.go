package visits

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type mockVisitsRepository struct {
	getPrimaryAgentForPropertyFunc       func(ctx context.Context, propertyID int32) (int32, error)
	getAgentScheduleFunc                 func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error)
	getPropertyExceptionsFunc            func(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error)
	getOccupiedVisitsFunc                func(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error)
	createVisitFunc                      func(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error)
	getVisitByUUIDFunc                   func(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error)
	listVisitsFunc                       func(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error)
	updateVisitStatusFunc                func(ctx context.Context, visitID int32, statusID int32) error
	createVisitStatusHistoryFunc         func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error
	getPropertyStatusAndCheckDeletedFunc func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error)
	checkUserActiveFunc                  func(ctx context.Context, userID int32) (int32, error)
	withTxFunc                           func(tx pgx.Tx) Repository
	beginFunc                            func(ctx context.Context) (pgx.Tx, error)
}

func (m *mockVisitsRepository) GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	if m.getPrimaryAgentForPropertyFunc != nil {
		return m.getPrimaryAgentForPropertyFunc(ctx, propertyID)
	}
	return 0, nil
}

func (m *mockVisitsRepository) GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
	if m.getAgentScheduleFunc != nil {
		return m.getAgentScheduleFunc(ctx, agentID)
	}
	return nil, nil
}

func (m *mockVisitsRepository) GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
	if m.getPropertyExceptionsFunc != nil {
		return m.getPropertyExceptionsFunc(ctx, propertyID, start, end)
	}
	return nil, nil
}

func (m *mockVisitsRepository) GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error) {
	if m.getOccupiedVisitsFunc != nil {
		return m.getOccupiedVisitsFunc(ctx, agentID, start, end)
	}
	return nil, nil
}

func (m *mockVisitsRepository) CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
	if m.createVisitFunc != nil {
		return m.createVisitFunc(ctx, arg)
	}
	return sqlcgen.Visit{}, nil
}

func (m *mockVisitsRepository) GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error) {
	if m.getVisitByUUIDFunc != nil {
		return m.getVisitByUUIDFunc(ctx, visitUUID)
	}
	return sqlcgen.Visit{}, nil
}

func (m *mockVisitsRepository) ListVisits(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
	if m.listVisitsFunc != nil {
		return m.listVisitsFunc(ctx, arg)
	}
	return nil, nil
}

func (m *mockVisitsRepository) UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error {
	if m.updateVisitStatusFunc != nil {
		return m.updateVisitStatusFunc(ctx, visitID, statusID)
	}
	return nil
}

func (m *mockVisitsRepository) CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error {
	if m.createVisitStatusHistoryFunc != nil {
		return m.createVisitStatusHistoryFunc(ctx, arg)
	}
	return nil
}

func (m *mockVisitsRepository) GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
	if m.getPropertyStatusAndCheckDeletedFunc != nil {
		return m.getPropertyStatusAndCheckDeletedFunc(ctx, propertyID)
	}
	return sqlcgen.GetPropertyStatusAndCheckDeletedRow{}, nil
}

func (m *mockVisitsRepository) CheckUserActive(ctx context.Context, userID int32) (int32, error) {
	if m.checkUserActiveFunc != nil {
		return m.checkUserActiveFunc(ctx, userID)
	}
	return 0, nil
}

func (m *mockVisitsRepository) WithTx(tx pgx.Tx) Repository {
	if m.withTxFunc != nil {
		return m.withTxFunc(tx)
	}
	return m
}

func (m *mockVisitsRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.beginFunc != nil {
		return m.beginFunc(ctx)
	}
	return nil, nil
}

type mockTx struct {
	pgx.Tx
	commitErr   error
	rollbackErr error
	commitFunc   func(ctx context.Context) error
	rollbackFunc func(ctx context.Context) error
}

func (m *mockTx) Commit(ctx context.Context) error {
	if m.commitErr != nil {
		return m.commitErr
	}
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}
	return nil
}

func (m *mockTx) Rollback(ctx context.Context) error {
	if m.rollbackErr != nil {
		return m.rollbackErr
	}
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}
	return nil
}
