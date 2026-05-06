package contracts

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockContractService struct {
	res CreateContractResult
	err error
}

func (m *mockContractService) GenerateContract(ctx context.Context, userID int32, input CreateContractInput) (CreateContractResult, error) {
	return m.res, m.err
}

func (m *mockContractService) ListContracts(ctx context.Context, userID int32, filter ListContractsFilter) ([]ContractListItem, error) {
	return []ContractListItem{{ContractUUID: "uuid-123"}}, m.err
}

func (m *mockContractService) GetContractDetail(ctx context.Context, userID int32, contractUUID uuid.UUID) (ContractDetail, error) {
	return ContractDetail{ContractUUID: "uuid-123"}, m.err
}

func TestGenerateContractHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        any
		userIDHeader   string
		mock           *mockContractService
		wantStatusCode int
	}{
		{
			name: "success",
			payload: CreateContractInput{
				TransactionID: 100,
				Currency:      "MXN",
				AgreedAmount:  1500,
				StartDate:     time.Now(),
			},
			userIDHeader:   "102",
			mock:           &mockContractService{res: CreateContractResult{ContractUUID: "uuid-123"}},
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "missing user id",
			payload:        CreateContractInput{TransactionID: 100},
			userIDHeader:   "",
			mock:           &mockContractService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid transaction id",
			payload:        CreateContractInput{TransactionID: 0},
			userIDHeader:   "102",
			mock:           &mockContractService{},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			
			body, _ := json.Marshal(tt.payload)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/contracts", bytes.NewReader(body))
			ctx.Request.Header.Set("X-User-ID", tt.userIDHeader)
			
			handler := NewHandler(tt.mock)
			handler.generateContract(ctx)
			
			if recorder.Code != tt.wantStatusCode {
				t.Errorf("handler.generateContract() status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestListContractsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name           string
		userIDHeader   string
		mock           *mockContractService
		wantStatusCode int
	}{
		{
			name:           "success",
			userIDHeader:   "1",
			mock:           &mockContractService{},
			wantStatusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/contracts", nil)
			ctx.Request.Header.Set("X-User-ID", tt.userIDHeader)
			handler := NewHandler(tt.mock)
			handler.listContracts(ctx)
			if recorder.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestGetContractHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contractUUID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	tests := []struct {
		name           string
		userIDHeader   string
		contractUUID   string
		mock           *mockContractService
		wantStatusCode int
	}{
		{
			name:           "success",
			userIDHeader:   "1",
			contractUUID:   contractUUID,
			mock:           &mockContractService{},
			wantStatusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/contracts/"+tt.contractUUID, nil)
			ctx.Request.Header.Set("X-User-ID", tt.userIDHeader)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.contractUUID}}
			handler := NewHandler(tt.mock)
			handler.getContract(ctx)
			if recorder.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}
