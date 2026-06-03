package visits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestHandler_ScheduleVisit(t *testing.T) {
	tests := []struct {
		name         string
		reqBody      interface{}
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on empty body",
			reqBody:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "400 Bad Request on invalid request (not future)",
			reqBody: CreateVisitRequest{
				PropertyID: 1,
				VisitDate:  time.Now().Add(-1 * time.Hour), // Past
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "400 Bad Request on service failure",
			reqBody: CreateVisitRequest{
				PropertyID: 1,
				VisitDate:  time.Now().Add(72 * time.Hour).Truncate(time.Hour),
			},
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					scheduleVisitFunc: func(ctx context.Context, uid, pid int32, vd time.Time) (VisitResponse, error) {
						return VisitResponse{}, errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "201 Created on success",
			reqBody: CreateVisitRequest{
				PropertyID: 1,
				VisitDate:  time.Now().Add(72 * time.Hour).Truncate(time.Hour),
			},
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					scheduleVisitFunc: func(ctx context.Context, uid, pid int32, vd time.Time) (VisitResponse, error) {
						return VisitResponse{}, nil
					},
				}
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVisitsService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodPost, "/visits")
			setAuthenticatedContext(ctx, 10, 3)

			if tt.reqBody != nil {
				body, _ := json.Marshal(tt.reqBody)
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			} else {
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer([]byte{}))
			}

			h.scheduleVisit(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandler_RescheduleVisit(t *testing.T) {
	tests := []struct {
		name         string
		uuidParam    string
		reqBody      interface{}
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on invalid uuid",
			uuidParam:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 Bad Request on invalid request body",
			uuidParam:  uuid.New().String(),
			reqBody:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "400 Bad Request on service failure",
			uuidParam: uuid.New().String(),
			reqBody: CreateVisitRequest{
				PropertyID: 1,
				VisitDate:  time.Now().Add(72 * time.Hour).Truncate(time.Hour),
			},
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					rescheduleVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID, nd time.Time) (VisitResponse, error) {
						return VisitResponse{}, errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "201 Created on success",
			uuidParam: uuid.New().String(),
			reqBody: CreateVisitRequest{
				PropertyID: 1,
				VisitDate:  time.Now().Add(72 * time.Hour).Truncate(time.Hour),
			},
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					rescheduleVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID, nd time.Time) (VisitResponse, error) {
						return VisitResponse{}, nil
					},
				}
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVisitsService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodPatch, "/visits/"+tt.uuidParam+"/reschedule")
			setAuthenticatedContext(ctx, 10, 3)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.uuidParam}}

			if tt.reqBody != nil {
				body, _ := json.Marshal(tt.reqBody)
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			} else {
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer([]byte{}))
			}

			h.rescheduleVisit(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}
