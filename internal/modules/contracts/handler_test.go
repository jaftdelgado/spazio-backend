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
	rentResult     CreateContractResult
	saleResult     CreateContractResult
	listResult     []ContractListItem
	detailResult   ContractDetail
	err            error
	receivedFilter ListContractsFilter
}

func (m *mockContractService) GenerateRentContract(ctx context.Context, userID int32, input CreateRentContractInput) (CreateContractResult, error) {
	return m.rentResult, m.err
}

func (m *mockContractService) GenerateSaleContract(ctx context.Context, userID int32, input CreateSaleContractInput) (CreateContractResult, error) {
	return m.saleResult, m.err
}

func (m *mockContractService) ListContracts(ctx context.Context, userID int32, roleID int32, filter ListContractsFilter) ([]ContractListItem, error) {
	m.receivedFilter = filter

	if m.listResult != nil {
		return m.listResult, m.err
	}

	return []ContractListItem{{ContractUUID: "uuid-123"}}, m.err
}

func (m *mockContractService) GetContractDetail(ctx context.Context, userID int32, roleID int32, contractUUID uuid.UUID) (ContractDetail, error) {
	if m.detailResult.ContractUUID != "" {
		return m.detailResult, m.err
	}

	return ContractDetail{ContractUUID: "uuid-123"}, m.err
}

func TestCreateRentContractHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	tests := []struct {
		name           string
		body           string
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
			mock: &mockContractService{
				rentResult: CreateContractResult{ContractUUID: "uuid-123"},
			},
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
			name:           "invalid json",
			body:           `{"transaction_id":`,
			mock:           &mockContractService{},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
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
		{
			name:           "bad request service error",
			payload:        CreateRentContractInput{TransactionID: 100},
			mock:           &mockContractService{err: errors.New("ya existe un contrato generado para esta transacción")},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "internal service error",
			payload:        CreateRentContractInput{TransactionID: 100},
			mock:           &mockContractService{err: errors.New("storage unavailable")},
			withAuth:       true,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			body := []byte(tt.body)
			if tt.body == "" {
				body, _ = json.Marshal(tt.payload)
			}

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
		body           string
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
			mock: &mockContractService{
				saleResult: CreateContractResult{ContractUUID: "uuid-456"},
			},
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
			name:           "invalid json",
			body:           `{"transaction_id":`,
			mock:           &mockContractService{},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
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
		{
			name:           "forbidden service error",
			payload:        CreateSaleContractInput{TransactionID: 200},
			mock:           &mockContractService{err: errors.New("no tiene permiso para generar el contrato de esta venta")},
			withAuth:       true,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "internal service error",
			payload:        CreateSaleContractInput{TransactionID: 200},
			mock:           &mockContractService{err: errors.New("pdf generation failed")},
			withAuth:       true,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			body := []byte(tt.body)
			if tt.body == "" {
				body, _ = json.Marshal(tt.payload)
			}

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
		withAuth       bool
		wantStatusCode int
	}{
		{
			name:           "success",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts",
			withAuth:       true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "success with valid filters",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?page=2&limit=20&transaction_type=rent&status_id=1&owner_id=5&start_date=2026-01-01T00:00:00Z&end_date=2026-12-31T00:00:00Z&search=casa",
			withAuth:       true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "missing auth context",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts",
			withAuth:       false,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid page",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?page=0",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid page text",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?page=abc",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid limit zero",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?limit=0",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid limit too high",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?limit=101",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid transaction type",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?transaction_type=lease",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid status id",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?status_id=abc",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid owner id",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?owner_id=abc",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid start date",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?start_date=bad-date",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid end date",
			mock:           &mockContractService{},
			url:            "/api/v1/contracts?end_date=bad-date",
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "internal service error",
			mock:           &mockContractService{err: errors.New("database unavailable")},
			url:            "/api/v1/contracts",
			withAuth:       true,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			ctx.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)

			if tt.withAuth {
				setContractAuth(ctx, 1, 1)
			}

			handler := NewHandler(tt.mock)
			handler.listContracts(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("listContracts() status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestGetContractHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"

	tests := []struct {
		name           string
		contractUUID   string
		mock           *mockContractService
		withAuth       bool
		wantStatusCode int
	}{
		{
			name:           "success",
			contractUUID:   validUUID,
			mock:           &mockContractService{},
			withAuth:       true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "missing auth context",
			contractUUID:   validUUID,
			mock:           &mockContractService{},
			withAuth:       false,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid uuid",
			contractUUID:   "invalid-uuid",
			mock:           &mockContractService{},
			withAuth:       true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "forbidden service error",
			contractUUID:   validUUID,
			mock:           &mockContractService{err: errors.New("no tiene permiso para ver este contrato")},
			withAuth:       true,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "not found service error",
			contractUUID:   validUUID,
			mock:           &mockContractService{err: errors.New("contract not found")},
			withAuth:       true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "internal service error",
			contractUUID:   validUUID,
			mock:           &mockContractService{err: errors.New("database unavailable")},
			withAuth:       true,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/contracts/"+tt.contractUUID, nil)
			ctx.Params = []gin.Param{{Key: "uuid", Value: tt.contractUUID}}

			if tt.withAuth {
				setContractAuth(ctx, 1, 1)
			}

			handler := NewHandler(tt.mock)
			handler.getContract(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("getContract() status = %d, want %d", recorder.Code, tt.wantStatusCode)
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