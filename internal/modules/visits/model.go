package visits

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

const (
	StatusPending       = 1
	StatusWaitingAgent  = 2
	StatusWaitingClient = 3
	StatusConfirmed     = 4
	StatusCancelled     = 5
	StatusCompleted     = 6

	PropertyStatusAvailable = 2
)

// AvailabilityRequest represents the query for available slots
type AvailabilityRequest struct {
	PropertyID int32     `form:"property_id" binding:"required" example:"5"`
	Date       time.Time `form:"date" binding:"required" time_format:"2006-01-02" example:"2025-01-20"`
}

// TimeSlot represents an available 1-hour window
type TimeSlot struct {
	StartTime time.Time `json:"start_time" example:"2025-01-20T10:00:00Z"`
	EndTime   time.Time `json:"end_time" example:"2025-01-20T11:00:00Z"`
	Available bool      `json:"available" example:"true"`
}

// CreateVisitRequest represents the body to schedule a visit
type CreateVisitRequest struct {
	PropertyID int32     `json:"property_id" binding:"required" example:"5"`
	VisitDate  time.Time `json:"visit_date" binding:"required" example:"2025-01-20T10:00:00Z"`
}

// VisitResponse represents the public info of a visit
type VisitResponse struct {
	VisitUUID     uuid.UUID `json:"visit_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	PropertyID    int32     `json:"property_id" example:"5"`
	PropertyTitle string    `json:"property_title" example:"Casa en Centro"`
	AgentID       int32     `json:"agent_id" example:"2"`
	AgentName     string    `json:"agent_name" example:"Juan Perez"`
	AgentPhone    string    `json:"agent_phone" example:"5551234567"`
	VisitDate     time.Time `json:"visit_date" example:"2025-01-20T10:00:00Z"`
	Status        string    `json:"status" example:"Pending"`
	CreatedAt     time.Time `json:"created_at" example:"2025-01-15T08:00:00Z"`
	ClientName    string    `json:"client_name" example:"Maria Gomez"`
	ClientPhone   string    `json:"client_phone" example:"5559876543"`
	CityName      string    `json:"city_name" example:"Ciudad de México"`
	Address       string    `json:"address" example:"Av. Reforma 123"`
}

type ListVisitsFilter struct {
	Date       *time.Time
	StatusID   *int32
	PropertyID *int32
}

type Repository interface {
	GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error)
	GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error)
	GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error)
	GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error)
	CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error)
	GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error)
	ListVisits(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error)
	UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error
	CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error
	GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error)
	CheckUserActive(ctx context.Context, userID int32) (int32, error)
	WithTx(tx pgx.Tx) Repository
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service defines the business logic for the visits module.
type Service interface {
	GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error)
	ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error)
	ListUserVisits(ctx context.Context, userID int32, roleID int32, filter ListVisitsFilter) ([]VisitResponse, error)
	ConfirmVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
	RescheduleVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error)
	CompleteVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
	CancelVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
}
