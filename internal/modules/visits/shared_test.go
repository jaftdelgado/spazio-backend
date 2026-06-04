package visits

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockVisitsService struct {
	getAvailableSlotsFunc func(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error)
	scheduleVisitFunc     func(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error)
	listUserVisitsFunc    func(ctx context.Context, userID int32, roleID int32, filter ListVisitsFilter) ([]VisitResponse, error)
	confirmVisitFunc      func(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
	rescheduleVisitFunc   func(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error)
	completeVisitFunc     func(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
	cancelVisitFunc       func(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error
}

func (m *mockVisitsService) GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error) {
	if m.getAvailableSlotsFunc != nil {
		return m.getAvailableSlotsFunc(ctx, propertyID, date)
	}
	return nil, nil
}

func (m *mockVisitsService) ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	if m.scheduleVisitFunc != nil {
		return m.scheduleVisitFunc(ctx, clientID, propertyID, visitDate)
	}
	return VisitResponse{}, nil
}

func (m *mockVisitsService) ListUserVisits(ctx context.Context, userID int32, roleID int32, filter ListVisitsFilter) ([]VisitResponse, error) {
	if m.listUserVisitsFunc != nil {
		return m.listUserVisitsFunc(ctx, userID, roleID, filter)
	}
	return nil, nil
}

func (m *mockVisitsService) ConfirmVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error {
	if m.confirmVisitFunc != nil {
		return m.confirmVisitFunc(ctx, userID, roleID, visitUUID)
	}
	return nil
}

func (m *mockVisitsService) RescheduleVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error) {
	if m.rescheduleVisitFunc != nil {
		return m.rescheduleVisitFunc(ctx, userID, roleID, visitUUID, newDate)
	}
	return VisitResponse{}, nil
}

func (m *mockVisitsService) CompleteVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error {
	if m.completeVisitFunc != nil {
		return m.completeVisitFunc(ctx, userID, roleID, visitUUID)
	}
	return nil
}

func (m *mockVisitsService) CancelVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error {
	if m.cancelVisitFunc != nil {
		return m.cancelVisitFunc(ctx, userID, roleID, visitUUID)
	}
	return nil
}

func newHandlerTestContext(method string, target string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request, _ = http.NewRequest(method, target, nil)
	return recorder, ctx
}

func setAuthenticatedContext(ctx *gin.Context, userID int32, roleID int32) {
	ctx.Set("user_id", userID)
	ctx.Set("role_id", roleID)
	ctx.Set("user_role", "client")
}

func assertErrorResponse(t *testing.T, body []byte, wantMsg string) {
	var resp struct {
		Error string `json:"error"`
	}
	json.Unmarshal(body, &resp)
	if resp.Error == "" {
		t.Errorf("expected error message %q, but got empty body or different structure: %s", wantMsg, string(body))
	}
}
