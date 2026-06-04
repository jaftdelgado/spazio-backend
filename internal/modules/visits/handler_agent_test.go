package visits

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestHandler_ConfirmVisit(t *testing.T) {
	tests := []struct {
		name         string
		uuidParam    string
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on invalid uuid",
			uuidParam:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "500 Internal Error on service failure",
			uuidParam: uuid.New().String(),
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					confirmVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID) error {
						return errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "200 OK on success",
			uuidParam: uuid.New().String(),
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					confirmVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID) error {
						return nil
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
			rec, ctx := newHandlerTestContext(http.MethodPatch, "/visits/"+tt.uuidParam+"/confirm")
			setAuthenticatedContext(ctx, 10, 3)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.uuidParam}}

			h.confirmVisit(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandler_CompleteVisit(t *testing.T) {
	tests := []struct {
		name         string
		uuidParam    string
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on invalid uuid",
			uuidParam:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "500 Internal Error on service failure",
			uuidParam: uuid.New().String(),
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					completeVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID) error {
						return errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "200 OK on success",
			uuidParam: uuid.New().String(),
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					completeVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID) error {
						return nil
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
			rec, ctx := newHandlerTestContext(http.MethodPatch, "/visits/"+tt.uuidParam+"/complete")
			setAuthenticatedContext(ctx, 10, 3)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.uuidParam}}

			h.completeVisit(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandler_CancelVisit(t *testing.T) {
	tests := []struct {
		name         string
		uuidParam    string
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on invalid uuid",
			uuidParam:  "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "500 Internal Error on service failure",
			uuidParam: uuid.New().String(),
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					cancelVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID) error {
						return errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "200 OK on success",
			uuidParam: uuid.New().String(),
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					cancelVisitFunc: func(ctx context.Context, uid, rid int32, vid uuid.UUID) error {
						return nil
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
			rec, ctx := newHandlerTestContext(http.MethodPatch, "/visits/"+tt.uuidParam+"/cancel")
			setAuthenticatedContext(ctx, 10, 3)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.uuidParam}}

			h.cancelVisit(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}
