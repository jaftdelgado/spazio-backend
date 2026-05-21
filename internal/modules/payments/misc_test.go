package payments

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
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
			assert.Contains(t, translated.Error(), tt.errContains)
		})
	}
}

func TestHandler_ResolveHelpers(t *testing.T) {
	h := NewHandler(&mockPaymentService{})
	rec, ctx := newHandlerTestContext(http.MethodPost, "/payments")

	// Call processPayment without auth context
	h.processPayment(ctx)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_ResolveListPaymentsInput_Errors(t *testing.T) {
	// property_id invalid
	_, err := resolveOptionalInt("abc", "property_id")
	assert.Error(t, err)

	// date invalid
	_, err = resolveOptionalDate("abc", "date_from")
	assert.Error(t, err)

	// date valid
	date, err := resolveOptionalDate("2024-01-01", "date_from")
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), *date)

	// limit invalid
	_, err = resolveLimit("abc")
	assert.Error(t, err)

	// offset invalid
	_, err = resolveOffset("abc")
	assert.Error(t, err)

	// limit out of bounds
	err = validateListPaymentsRequest(101, 0, nil, nil)
	assert.Error(t, err)

	err = validateListPaymentsRequest(-1, 0, nil, nil)
	assert.Error(t, err)

	err = validateListPaymentsRequest(10, -1, nil, nil)
	assert.Error(t, err)
}
