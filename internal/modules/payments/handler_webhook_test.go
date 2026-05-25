package payments

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestHandler_HandleWebhook(t *testing.T) {
	tests := []struct {
		name         string
		setupService func() *mockPaymentService
		headers      map[string]string
		wantStatus   int
	}{
		{
			name: "401 Unauthorized when signature validation fails",
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					handleWebhookFunc: func(ctx context.Context, sig, rid string, body []byte) error {
						return errors.New("invalid signature")
					},
				}
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "200 OK when processed successfully",
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					handleWebhookFunc: func(ctx context.Context, sig, rid string, body []byte) error {
						return nil
					},
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.setupService()
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodPost, "/webhook")
			ctx.Request.Header.Set("x-signature", "fake")
			ctx.Request.Body = http.NoBody
			if tt.headers != nil {
				for k, v := range tt.headers {
					ctx.Request.Header.Set(k, v)
				}
			}

			h.handleWebhook(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}
