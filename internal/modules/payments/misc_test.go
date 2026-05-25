package payments

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		errContains string
	}{
		{
			name:        "pgError 23503",
			err:         &pgconn.PgError{Code: "23503"},
			errContains: "el contrato o método de pago seleccionado no existe",
		},
		{
			name:        "other pgError",
			err:         &pgconn.PgError{Code: "99999"},
			errContains: "99999",
		},
		{
			name:        "other error",
			err:         errors.New("other error"),
			errContains: "other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translated := translateError(tt.err)
			if !strings.Contains(translated.Error(), tt.errContains) {
				t.Errorf("expected %v to contain %v", translated.Error(), tt.errContains)
			}
		})
	}
}

func TestHandler_ResolveHelpers(t *testing.T) {
	h := NewHandler(&mockPaymentService{})
	rec, ctx := newHandlerTestContext(http.MethodPost, "/payments")

	// Call processPayment without auth context
	h.processPayment(ctx)

	if http.StatusUnauthorized != rec.Code {
		t.Errorf("expected %v, got %v", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_ResolveListPaymentsInput_Errors(t *testing.T) {
	// property_id invalid
	_, err := resolveOptionalInt("abc", "property_id")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// date invalid
	_, err = resolveOptionalDate("abc", "date_from")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// date valid
	date, err := resolveOptionalDate("2024-01-01", "date_from")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) != *date {
		t.Errorf("expected %v, got %v", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), *date)
	}

	// limit invalid
	_, err = resolveLimit("abc")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// offset invalid
	_, err = resolveOffset("abc")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// limit out of bounds
	err = validateListPaymentsRequest(101, 0, nil, nil)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	err = validateListPaymentsRequest(-1, 0, nil, nil)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	err = validateListPaymentsRequest(10, -1, nil, nil)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
