package visits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error) {
	args := m.Called(ctx, propertyID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TimeSlot), args.Error(1)
}

func (m *MockService) ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	args := m.Called(ctx, clientID, propertyID, visitDate)
	return args.Get(0).(VisitResponse), args.Error(1)
}

func (m *MockService) ListUserVisits(ctx context.Context, userID int32, roleID int32, filter ListVisitsFilter) ([]VisitResponse, error) {
	args := m.Called(ctx, userID, roleID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]VisitResponse), args.Error(1)
}

func (m *MockService) ConfirmVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID, visitUUID)
	return args.Error(0)
}

func (m *MockService) RescheduleVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error) {
	args := m.Called(ctx, userID, roleID, visitUUID, newDate)
	return args.Get(0).(VisitResponse), args.Error(1)
}

func (m *MockService) CompleteVisit(ctx context.Context, userID int32, roleID int32, visitUUID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID, visitUUID)
	return args.Error(0)
}

func setupHandlerTest(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-Test-User-ID"); userID != "" {
			if parsedUserID, err := strconv.ParseInt(userID, 10, 32); err == nil {
				c.Set("user_id", int32(parsedUserID))
			}
		}
		if roleID := c.GetHeader("X-Test-Role-ID"); roleID != "" {
			if parsedRoleID, err := strconv.ParseInt(roleID, 10, 32); err == nil {
				c.Set("role_id", int32(parsedRoleID))
			}
		}
		if c.Value("user_id") != nil || c.GetHeader("X-Test-User-ID") != "" {
			c.Set("user_role", "client")
			c.Set("user_uuid", "uuid-123")
			c.Set("user_email", "user@example.com")
		}
		c.Next()
	})
	h.RegisterRoutes(r.Group(""))
	return r
}

func TestHandler_GetAvailability(t *testing.T) {
	svc := new(MockService)
	r := setupHandlerTest(svc)

	t.Run("Missing Date (Success with Default)", func(t *testing.T) {
		svc.On("GetAvailableSlots", mock.Anything, int32(1), mock.Anything).Return([]TimeSlot{}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties/1/availability", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid Property ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties/abc/availability?date=2024-10-10", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid Date Format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties/1/availability?date=invalid", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.On("GetAvailableSlots", mock.Anything, int32(1), mock.Anything).Return(nil, errors.New("fail")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties/1/availability?date=2024-10-10", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		dateStr := "2024-10-10"
		date, _ := time.Parse("2006-01-02", dateStr)
		svc.On("GetAvailableSlots", mock.Anything, int32(1), date).Return([]TimeSlot{{Available: true}}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties/1/availability?date="+dateStr, nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_ScheduleVisit(t *testing.T) {
	svc := new(MockService)
	r := setupHandlerTest(svc)

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 1, VisitDate: time.Now().Add(100 * time.Hour).Truncate(time.Hour)})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/visits", bytes.NewReader(body))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Bad JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/visits", bytes.NewReader([]byte("invalid")))
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation Error (Past Date)", func(t *testing.T) {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 1, VisitDate: time.Now().Add(-10 * time.Hour)})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/visits", bytes.NewReader(body))
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		futureDate := time.Now().Add(100 * time.Hour).Truncate(time.Hour)
		svc.On("ScheduleVisit", mock.Anything, int32(1), int32(10), futureDate).Return(VisitResponse{}, errors.New("fail")).Once()
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 10, VisitDate: futureDate})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/visits", bytes.NewReader(body))
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		futureDate := time.Now().Add(100 * time.Hour).Truncate(time.Hour)
		svc.On("ScheduleVisit", mock.Anything, int32(1), int32(10), futureDate).Return(VisitResponse{Status: "Pending"}, nil).Once()
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 10, VisitDate: futureDate})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/visits", bytes.NewReader(body))
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestHandler_ListUserVisits(t *testing.T) {
	svc := new(MockService)
	r := setupHandlerTest(svc)

	t.Run("Missing User ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/visits", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.On("ListUserVisits", mock.Anything, int32(1), int32(3), mock.Anything).Return(nil, errors.New("fail")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/visits", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success with Filters", func(t *testing.T) {
		svc.On("ListUserVisits", mock.Anything, int32(1), int32(3), mock.Anything).Return([]VisitResponse{}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/visits?date=2024-10-10&status_id=1&property_id=10", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_ConfirmVisit(t *testing.T) {
	svc := new(MockService)
	r := setupHandlerTest(svc)
	uID := uuid.New()

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/confirm", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/abc/confirm", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.On("ConfirmVisit", mock.Anything, int32(1), int32(3), uID).Return(errors.New("fail")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/confirm", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		svc.On("ConfirmVisit", mock.Anything, int32(1), int32(3), uID).Return(nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/confirm", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_RescheduleVisit(t *testing.T) {
	svc := new(MockService)
	r := setupHandlerTest(svc)
	uID := uuid.New()

	t.Run("Invalid UUID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/abc/reschedule", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Bad JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/reschedule", bytes.NewReader([]byte("invalid")))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 1, VisitDate: time.Time{}}) // Zero date
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/reschedule", bytes.NewReader(body))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		w := httptest.NewRecorder()
		futureDate := time.Now().Add(100 * time.Hour).Truncate(time.Hour)
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 10, VisitDate: futureDate})
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/reschedule", bytes.NewReader(body))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		futureDate := time.Now().Add(100 * time.Hour).Truncate(time.Hour)
		svc.On("RescheduleVisit", mock.Anything, int32(1), int32(3), uID, futureDate).Return(VisitResponse{}, errors.New("fail")).Once()
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 10, VisitDate: futureDate})
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/reschedule", bytes.NewReader(body))
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		futureDate := time.Now().Add(100 * time.Hour).Truncate(time.Hour)
		svc.On("RescheduleVisit", mock.Anything, int32(1), int32(3), uID, futureDate).Return(VisitResponse{Status: "Pending"}, nil).Once()
		w := httptest.NewRecorder()
		body, _ := json.Marshal(CreateVisitRequest{PropertyID: 10, VisitDate: futureDate})
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/reschedule", bytes.NewReader(body))
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestHandler_CompleteVisit(t *testing.T) {
	svc := new(MockService)
	r := setupHandlerTest(svc)
	uID := uuid.New()

	t.Run("Invalid UUID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/abc/complete", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Unauthorized without auth context", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/complete", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.On("CompleteVisit", mock.Anything, int32(1), int32(3), uID).Return(errors.New("fail")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/complete", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		svc.On("CompleteVisit", mock.Anything, int32(1), int32(3), uID).Return(nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/visits/"+uID.String()+"/complete", nil)
		setVisitAuth(req, 1, 3)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func setVisitAuth(req *http.Request, userID int32, roleID int32) {
	req.Header.Set("X-Test-User-ID", strconv.Itoa(int(userID)))
	req.Header.Set("X-Test-Role-ID", strconv.Itoa(int(roleID)))
}
