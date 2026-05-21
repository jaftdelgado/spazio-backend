package visits

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestPgTimeToHM(t *testing.T) {
	pt := pgtype.Time{Microseconds: 15*3600*1e6 + 30*60*1e6, Valid: true}
	h, m := pgTimeToHM(pt)
	assert.Equal(t, 15, h)
	assert.Equal(t, 30, m)
}

func TestHandler_ResolveIdentity_Errors(t *testing.T) {
	// If context doesn't have the keys, resolveAuthenticatedIdentity returns false
	svc := &mockVisitsService{}
	h := NewHandler(svc)
	rec, ctx := newHandlerTestContext(http.MethodGet, "/visits")

	// Call listVisits without setAuthenticatedContext
	h.listVisits(ctx)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
