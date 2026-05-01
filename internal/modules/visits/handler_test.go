package visits

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockService struct {
	availableSlots []TimeSlot
	scheduleResult VisitResponse
	listResult     []VisitResponse
	err            error
	called         map[string]bool
}

func (m *mockService) GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error) {
	m.called["GetAvailableSlots"] = true
	return m.availableSlots, m.err
}

func (m *mockService) ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	m.called["ScheduleVisit"] = true
	return m.scheduleResult, m.err
}

func (m *mockService) ListUserVisits(ctx context.Context, userID int32, filter ListVisitsFilter) ([]VisitResponse, error) {
	m.called["ListUserVisits"] = true
	return m.listResult, m.err
}

func (m *mockService) ConfirmVisit(ctx context.Context, userID int32, visitUUID uuid.UUID) error {
	m.called["ConfirmVisit"] = true
	return m.err
}

func (m *mockService) RescheduleVisit(ctx context.Context, userID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error) {
	m.called["RescheduleVisit"] = true
	return m.scheduleResult, m.err
}

func (m *mockService) CompleteVisit(ctx context.Context, userID int32, visitUUID uuid.UUID) error {
	m.called["CompleteVisit"] = true
	return m.err
}

func TestScheduleVisitHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	futureDate := time.Now().Add(72 * time.Hour).Truncate(time.Hour)

	tests := []struct {
		name             string
		payload          any
		userIDHeader     string
		mock             *mockService
		wantStatusCode   int
		wantBodyContains string
	}{
		{
			name: "successful schedule",
			payload: CreateVisitRequest{
				PropertyID: 1,
				VisitDate:  futureDate,
			},
			userIDHeader: "3",
			mock: &mockService{
				called: make(map[string]bool),
				scheduleResult: VisitResponse{VisitUUID: uuid.New(), Status: "Pending"},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "invalid user id",
			payload:        CreateVisitRequest{PropertyID: 1, VisitDate:  futureDate},
			userIDHeader:   "abc",
			mock:           &mockService{called: make(map[string]bool)},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			body, _ := json.Marshal(tt.payload)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/visits", bytes.NewReader(body))
			ctx.Request.Header.Set("X-User-ID", tt.userIDHeader)
			handler := NewHandler(tt.mock)
			handler.scheduleVisit(ctx)
			if recorder.Code != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestGetAvailabilityHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		propertyID string
		dateQuery  string
		mock       *mockService
		wantStatusCode int
	}{
		{
			name:       "successful",
			propertyID: "1",
			dateQuery:  "2026-05-15",
			mock: &mockService{
				called: make(map[string]bool),
				availableSlots: []TimeSlot{{Available: true}},
			},
			wantStatusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/properties/"+tt.propertyID+"/availability?date="+tt.dateQuery, nil)
			ctx.Params = []gin.Param{{Key: "id", Value: tt.propertyID}}
			handler := NewHandler(tt.mock)
			handler.getAvailability(ctx)
			if recorder.Code != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestConfirmVisitHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	visitUUID := uuid.New()
	tests := []struct {
		name         string
		visitUUID    string
		userIDHeader string
		mock         *mockService
		wantStatusCode int
	}{
		{
			name:         "successful",
			visitUUID:    visitUUID.String(),
			userIDHeader: "2",
			mock:         &mockService{called: make(map[string]bool)},
			wantStatusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPatch, "/visits/"+tt.visitUUID+"/confirm", nil)
			ctx.Request.Header.Set("X-User-ID", tt.userIDHeader)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.visitUUID}}
			handler := NewHandler(tt.mock)
			handler.confirmVisit(ctx)
			if recorder.Code != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}
