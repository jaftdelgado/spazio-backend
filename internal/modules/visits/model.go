package visits

import (
	"time"

	"github.com/google/uuid"
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
