package contracts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func (m *mockContractService) GenerateRentContract(ctx context.Context, userID int32, input CreateRentContractInput) (CreateContractResult, error) {
	return m.res, m.err
}

func (m *mockContractService) GenerateSaleContract(ctx context.Context, userID int32, input CreateSaleContractInput) (CreateContractResult, error) {
	return m.res, m.err
}

func (m *mockContractService) ListContracts(ctx context.Context, userID int32, roleID int32, filter ListContractsFilter) ([]ContractListItem, error) {
	return []ContractListItem{{ContractUUID: "uuid-123"}}, m.err
}

func (m *mockContractService) GetContractDetail(ctx context.Context, userID int32, roleID int32, contractUUID uuid.UUID) (ContractDetail, error) {
	return ContractDetail{ContractUUID: "uuid-123"}, m.err
}

func TestCreateRentContractHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	tests := []struct {
		name           string
		payload        any
		mock           *mockContractService
		withAuth       bool
		wantStatusCode int
	}{
		{
			name: "success",
			payload: CreateRentContractInput{
				TransactionID: 100,
				PeriodID:      3,
				Currency:      "MXN",
				AgreedAmount:  1500,
				StartDate:     startDate,
				EndDate:       endDate,
			},
			mock:           &mockContractService{res: CreateContractResult{ContractUUID: "uuid-123"}},
			withAuth:       true,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "missing auth context",
			payload:        CreateRentContractInput{TransactionID: 100},
			mock:           &mockContractService{},
			withAuth:       false,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid transaction id",
			payload:        CreateRentContractInput{TransactionID: 0},
			mock:           &mockContractService{},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "forbidden service error",
			payload:        CreateRentContractInput{TransactionID: 100},
			mock:           &mockContractService{err: errors.New("no tiene permiso para generar el contrato de esta renta")},
			withAuth:       true,
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			body, _ := json.Marshal(tt.payload)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/contracts/rent", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			if tt.withAuth {
				setContractAuth(ctx, 201, 3)
			}

			handler := NewHandler(tt.mock)
			handler.createRentContract(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("createRentContract() status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestCreateSaleContractHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        any
		mock           *mockContractService
		withAuth       bool
		wantStatusCode int
	}{
		{
			name: "success",
			payload: CreateSaleContractInput{
				TransactionID: 200,
				Currency:      "MXN",
				AgreedAmount:  850000,
			},
			mock:           &mockContractService{res: CreateContractResult{ContractUUID: "uuid-456"}},
			withAuth:       true,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "missing auth context",
			payload:        CreateSaleContractInput{TransactionID: 200},
			mock:           &mockContractService{},
			withAuth:       false,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid transaction id",
			payload:        CreateSaleContractInput{TransactionID: 0},
			mock:           &mockContractService{},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request service error",
			payload:        CreateSaleContractInput{TransactionID: 200},
			mock:           &mockContractService{err: errors.New("la transacción no corresponde a una venta")},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			body, _ := json.Marshal(tt.payload)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/contracts/sale", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			if tt.withAuth {
				setContractAuth(ctx, 102, 2)
			}

			handler := NewHandler(tt.mock)
			handler.createSaleContract(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("createSaleContract() status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestListContractsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mock           *mockContractService
		url            string
		wantStatusCode int
	}{
		{
			name:           "success",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid page",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?page=0",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			ctx.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)
			setContractAuth(ctx, 1, 1)

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
		contractUUID   string
		mock           *mockContractService
		wantStatusCode int
	}{
		{
			name:           "success",
			contractUUID:   contractUUID,
			mock:           &mockContractService{},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid uuid",
			contractUUID:   "invalid-uuid",
			mock:           &mockContractService{},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/contracts/"+tt.contractUUID, nil)
			setContractAuth(ctx, 1, 1)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.contractUUID}}

			handler := NewHandler(tt.mock)
			handler.getContract(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func setContractAuth(ctx *gin.Context, userID int32, roleID int32) {
	ctx.Set("user_id", userID)
	ctx.Set("role_id", roleID)
	ctx.Set("user_role", "admin")
	ctx.Set("user_uuid", "uuid-123")
	ctx.Set("user_email", "user@example.com")
}