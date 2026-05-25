package visits

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	agentID, err := r.queries.GetPrimaryAgentForProperty(ctx, propertyID)
	if err != nil {
		return 0, fmt.Errorf("get primary agent for property: %w", err)
	}
	return agentID, nil
}

func (r *repository) GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
	rows, err := r.queries.GetAgentSchedule(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent schedule: %w", err)
	}
	return rows, nil
}

func (r *repository) GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
	rows, err := r.queries.GetPropertyExceptions(ctx, sqlcgen.GetPropertyExceptionsParams{
		PropertyID:      propertyID,
		ExceptionDate:   pgtype.Date{Time: start, Valid: true},
		ExceptionDate_2: pgtype.Date{Time: end, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("get property exceptions: %w", err)
	}
	return rows, nil
}

func (r *repository) GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error) {
	pgTimes, err := r.queries.GetOccupiedVisits(ctx, sqlcgen.GetOccupiedVisitsParams{
		AgentID:     pgtype.Int4{Int32: agentID, Valid: true},
		VisitDate:   pgtype.Timestamptz{Time: start, Valid: true},
		VisitDate_2: pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("get occupied visits: %w", err)
	}

	times := make([]time.Time, 0, len(pgTimes))
	for _, pt := range pgTimes {
		times = append(times, pt.Time)
	}
	return times, nil
}

func (r *repository) CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
	visit, err := r.queries.CreateVisit(ctx, arg)
	if err != nil {
		return sqlcgen.Visit{}, fmt.Errorf("create visit: %w", err)
	}
	return visit, nil
}

func (r *repository) GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error) {
	visit, err := r.queries.GetVisitByUUID(ctx, pgtype.UUID{Bytes: visitUUID, Valid: true})
	if err != nil {
		return sqlcgen.Visit{}, fmt.Errorf("get visit by uuid: %w", err)
	}
	return visit, nil
}

func (r *repository) ListVisits(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
	rows, err := r.queries.ListVisits(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list visits: %w", err)
	}
	return rows, nil
}

func (r *repository) UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error {
	err := r.queries.UpdateVisitStatus(ctx, sqlcgen.UpdateVisitStatusParams{
		VisitID:  visitID,
		StatusID: statusID,
	})
	if err != nil {
		return fmt.Errorf("update visit status: %w", err)
	}
	return nil
}

func (r *repository) CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error {
	err := r.queries.CreateVisitStatusHistory(ctx, arg)
	if err != nil {
		return fmt.Errorf("create visit status history: %w", err)
	}
	return nil
}

func (r *repository) GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
	row, err := r.queries.GetPropertyStatusAndCheckDeleted(ctx, propertyID)
	if err != nil {
		return sqlcgen.GetPropertyStatusAndCheckDeletedRow{}, fmt.Errorf("get property status and check deleted: %w", err)
	}
	return row, nil
}

func (r *repository) CheckUserActive(ctx context.Context, userID int32) (int32, error) {
	active, err := r.queries.CheckUserActive(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("check user active: %w", err)
	}
	return active, nil
}
