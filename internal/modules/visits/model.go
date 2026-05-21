package visits

import (
	"context"
	"time"

	"github.com/google/uuid"
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
	PropertyID int32     `form:"property_id" binding:"required"`
	Date       time.Time `form:"date" binding:"required" time_format:"2006-01-02"`
}

// TimeSlot represents an available 1-hour window
type TimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Available bool      `json:"available"`
}

// CreateVisitRequest represents the body to schedule a visit
type CreateVisitRequest struct {
	PropertyID int32     `json:"property_id" binding:"required"`
	VisitDate  time.Time `json:"visit_date" binding:"required"`
}

// VisitResponse represents the public info of a visit
type VisitResponse struct {
	VisitUUID     uuid.UUID `json:"visit_uuid"`
	PropertyID    int32     `json:"property_id"`
	PropertyTitle string    `json:"property_title"`
	AgentID       int32     `json:"agent_id"`
	AgentName     string    `json:"agent_name"`
	AgentPhone    string    `json:"agent_phone"`
	VisitDate     time.Time `json:"visit_date"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ClientName    string    `json:"client_name"`
	ClientPhone   string    `json:"client_phone"`
	CityName      string    `json:"city_name"`
	Address       string    `json:"address"`
}

type ListVisitsFilter struct {
	Date       *time.Time
	StatusID   *int32
	PropertyID *int32
}

// Service defines the business logic for the visits module.
type Service interface {
	GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error)
	ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error)
	ListUserVisits(ctx context.Context, userID int32, roleID int32, filter ListVisitsFilter) ([]VisitResponse, error)
	ConfirmVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
	RescheduleVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error)
	CompleteVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
}
