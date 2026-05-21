package payments

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHandler_ConfirmPendingPayment(t *testing.T) {
	tests := []struct {
		name         string
		setupService func() *mockPaymentService
		uuidParam    string
		wantStatus   int
	}{
		{
			name:       "400 Bad Request when uuid is invalid",
			uuidParam:  "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "404 Not Found (F6 Mapping)",
			uuidParam: uuid.New().String(),
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					confirmPendingPaymentFunc: func(ctx context.Context, uid int32, pid uuid.UUID) error {
						return errors.New("pago no encontrado")
					},
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "200 OK on success",
			uuidParam: uuid.New().String(),
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					confirmPendingPaymentFunc: func(ctx context.Context, uid int32, pid uuid.UUID) error {
						return nil
					},
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPaymentService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodPatch, "/confirm")
			setAuthenticatedContext(ctx, 10, 3)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.uuidParam}}

			h.confirmPendingPayment(ctx)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
