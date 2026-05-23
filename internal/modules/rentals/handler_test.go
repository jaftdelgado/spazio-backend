package rentals

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockRentalService struct {
	previewFunc func(ctx context.Context, auth AuthContext, input RentalPreviewInput) (RentalPreviewResponse, error)
	confirmFunc func(ctx context.Context, auth AuthContext, input RentalConfirmInput) (RentalResponse, error)
}

func (m *mockRentalService) PreviewRental(ctx context.Context, auth AuthContext, input RentalPreviewInput) (RentalPreviewResponse, error) {
	if m.previewFunc != nil {
		return m.previewFunc(ctx, auth, input)
	}
	return RentalPreviewResponse{}, nil
}

func (m *mockRentalService) ConfirmRental(ctx context.Context, auth AuthContext, input RentalConfirmInput) (RentalResponse, error) {
	if m.confirmFunc != nil {
		return m.confirmFunc(ctx, auth, input)
	}
	return RentalResponse{}, nil
}

func TestHandler_PreviewRental(t *testing.T) {
	gin.SetMode(gin.TestMode)

	propertyUUID := uuid.New()
	service := &mockRentalService{
		previewFunc: func(ctx context.Context, auth AuthContext, input RentalPreviewInput) (RentalPreviewResponse, error) {
			return RentalPreviewResponse{
				PropertyUUID: propertyUUID.String(),
				Period:       "Monthly",
				PeriodID:     3,
			}, nil
		},
	}
	handler := NewHandler(service)

	body, _ := json.Marshal(RentalPreviewRequest{
		PropertyUUID: propertyUUID.String(),
		PeriodID:     3,
		StartDate:    "2026-07-01",
		EndDate:      "2026-09-30",
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rentals/preview", bytes.NewReader(body))
	ctx.Request.Header.Set("Authorization", "Bearer token")
	setRentalAuth(ctx, uuid.New(), 7, 3)

	handler.previewRental(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestHandler_PreviewRental_InvalidBody_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(&mockRentalService{})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rentals/preview", bytes.NewReader([]byte(`{"property_uuid":"bad"`)))
	ctx.Request.Header.Set("Authorization", "Bearer token")
	setRentalAuth(ctx, uuid.New(), 7, 3)

	handler.previewRental(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandler_ConfirmRental_ServiceError_Returns403(t *testing.T) {
	gin.SetMode(gin.TestMode)

	propertyUUID := uuid.New()
	clientUUID := uuid.New()
	service := &mockRentalService{
		confirmFunc: func(ctx context.Context, auth AuthContext, input RentalConfirmInput) (RentalResponse, error) {
			return RentalResponse{}, newStatusError(http.StatusForbidden, "forbidden")
		},
	}
	handler := NewHandler(service)

	body, _ := json.Marshal(RentalConfirmRequest{
		PropertyUUID: propertyUUID.String(),
		ClientUUID:   clientUUID.String(),
		PeriodID:     3,
		StartDate:    "2026-07-01",
		EndDate:      "2026-09-30",
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rentals", bytes.NewReader(body))
	ctx.Request.Header.Set("Authorization", "Bearer token")
	setRentalAuth(ctx, clientUUID, 7, 3)

	handler.confirmRental(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func setRentalAuth(ctx *gin.Context, userUUID uuid.UUID, userID int32, roleID int32) {
	ctx.Set("user_id", userID)
	ctx.Set("role_id", roleID)
	ctx.Set("user_uuid", userUUID.String())
}
