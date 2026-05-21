package visits

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_ListVisits(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		setupService func() *mockVisitsService
		wantStatus   int
	}{
		{
			name:       "200 OK on empty query",
			query:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "200 OK with query parsing",
			query:      "?status_id=1&property_id=1&date=2024-01-01",
			wantStatus: http.StatusOK,
		},
		{
			name:  "500 Internal Server Error on service failure",
			query: "",
			setupService: func() *mockVisitsService {
				return &mockVisitsService{
					listUserVisitsFunc: func(ctx context.Context, uid, rid int32, filter ListVisitsFilter) ([]VisitResponse, error) {
						return nil, errors.New("service fail")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockVisitsService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodGet, "/visits"+tt.query)
			setAuthenticatedContext(ctx, 10, 3)

			h.listVisits(ctx)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
