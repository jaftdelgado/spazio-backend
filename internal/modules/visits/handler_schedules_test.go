package visits

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHandler_GetAvailability(t *testing.T) {
	tests := []struct {
		name         string
		uuidParam    string
		dateQuery    string
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "400 Bad Request when property ID is invalid",
			uuidParam:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 Bad Request when date is invalid",
			uuidParam:  "1",
			dateQuery:  "?date=invalid-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "500 Internal Error on service failure",
			uuidParam: "1",
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					getAvailableSlotsFunc: func(ctx context.Context, pid int32, d time.Time) ([]TimeSlot, error) {
						return nil, errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "200 OK on success",
			uuidParam: "1",
			dateQuery: "?date=2024-01-01",
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					getAvailableSlotsFunc: func(ctx context.Context, pid int32, d time.Time) ([]TimeSlot, error) {
						return []TimeSlot{}, nil
					},
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVisitsService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodGet, "/properties/1/availability"+tt.dateQuery)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.uuidParam}}

			h.getAvailability(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}
