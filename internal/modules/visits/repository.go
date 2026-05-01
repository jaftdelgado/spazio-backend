package visits

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type Repository interface {
	GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error)
	GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error)
	GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error)
	GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error)
	CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error)
	GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error)
	GetUserRole(ctx context.Context, userID int32) (int32, error)
	ListVisits(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error)
	UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error
	CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error
	GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error)
	CheckUserActive(ctx context.Context, userID int32) (int32, error)
	// Transaction support
	WithTx(tx pgx.Tx) Repository
	Begin(ctx context.Context) (pgx.Tx, error)
}

type repository struct {
	db      sqlcgen.DBTX
	queries *sqlcgen.Queries
	pool    *pgxpool.Pool // To start transactions
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db:      db,
		queries: sqlcgen.New(db),
		pool:    db,
	}
}

func (r *repository) WithTx(tx pgx.Tx) Repository {
	return &repository{
		db:      tx,
		queries: r.queries.WithTx(tx),
		pool:    r.pool,
	}
}

func (r *repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *repository) GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	return r.queries.GetPrimaryAgentForProperty(ctx, propertyID)
}

func (r *repository) GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
	return r.queries.GetAgentSchedule(ctx, agentID)
}

func (r *repository) GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
	return r.queries.GetPropertyExceptions(ctx, sqlcgen.GetPropertyExceptionsParams{
		PropertyID:      propertyID,
		ExceptionDate:   pgtype.Date{Time: start, Valid: true},
		ExceptionDate_2: pgtype.Date{Time: end, Valid: true},
	})
}

func (r *repository) GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error) {
	pgTimes, err := r.queries.GetOccupiedVisits(ctx, sqlcgen.GetOccupiedVisitsParams{
		AgentID:     pgtype.Int4{Int32: agentID, Valid: true},
		VisitDate:   pgtype.Timestamptz{Time: start, Valid: true},
		VisitDate_2: pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	times := make([]time.Time, len(pgTimes))
	for i, pt := range pgTimes {
		times[i] = pt.Time
	}
	return times, nil
}

func (r *repository) CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
	return r.queries.CreateVisit(ctx, arg)
}

func (r *repository) GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error) {
	return r.queries.GetVisitByUUID(ctx, pgtype.UUID{Bytes: visitUUID, Valid: true})
}

func (r *repository) GetUserRole(ctx context.Context, userID int32) (int32, error) {
	return r.queries.GetUserRole(ctx, userID)
}

func (r *repository) ListVisits(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
	return r.queries.ListVisits(ctx, arg)
}

func (r *repository) UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error {
	return r.queries.UpdateVisitStatus(ctx, sqlcgen.UpdateVisitStatusParams{
		VisitID:  visitID,
		StatusID: statusID,
	})
}

func (r *repository) CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error {
	return r.queries.CreateVisitStatusHistory(ctx, arg)
}

func (r *repository) GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
	return r.queries.GetPropertyStatusAndCheckDeleted(ctx, propertyID)
}

func (r *repository) CheckUserActive(ctx context.Context, userID int32) (int32, error) {
	return r.queries.CheckUserActive(ctx, userID)
}
