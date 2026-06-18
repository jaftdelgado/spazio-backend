//go:build integration

package contracts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	integrationRoleAdminID  int32 = 1
	integrationRoleAgentID  int32 = 2
	integrationRoleClientID int32 = 3

	integrationUserStatusActiveID int32 = 1

	integrationPropertyStatusAvailableID int32 = 2
	integrationPropertyStatusSoldID      int32 = 3

	integrationTransactionStatusPendingID int32 = 1
	integrationTransactionStatusClosedID  int32 = 3

	integrationContractStatusDraftID int32 = 1

	integrationPropertyTypeHouseID int32 = 9001
	integrationModalitySaleID      int32 = 9001
	integrationModalityRentID      int32 = 9002
	integrationCountryID           int32 = 9001
	integrationStateID             int32 = 9001
	integrationCityID              int32 = 9001
	integrationOrientationID       int32 = 9001

	integrationRentPeriodMonthlyID int32 = 3
)

type integrationContractStorage struct {
	uploadedKeys []string
	uploadedPDFs map[string][]byte
	publicURLs   map[string]string
}

func newIntegrationContractStorage() *integrationContractStorage {
	return &integrationContractStorage{
		uploadedKeys: make([]string, 0),
		uploadedPDFs: make(map[string][]byte),
		publicURLs:   make(map[string]string),
	}
}

func (s *integrationContractStorage) Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error {
	if strings.TrimSpace(storageKey) == "" {
		return nil
	}

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, body); err != nil {
		return err
	}

	s.uploadedKeys = append(s.uploadedKeys, storageKey)
	s.uploadedPDFs[storageKey] = buffer.Bytes()
	s.publicURLs[storageKey] = "https://storage.test/" + storageKey

	return nil
}

func (s *integrationContractStorage) PublicURL(ctx context.Context, storageKey string) (string, error) {
	if url, ok := s.publicURLs[storageKey]; ok {
		return url, nil
	}

	return "https://storage.test/" + storageKey, nil
}

type integrationContractFixture struct {
	ownerID       int32
	clientID      int32
	agentID       int32
	propertyID    int32
	propertyUUID  string
	transactionID int32
	amount        float64
	periodID      int32
}

func TestIntegration_Contracts_Setup(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	if repo == nil {
		t.Fatal("expected contracts repository")
	}

	if storage == nil {
		t.Fatal("expected fake contract storage")
	}

	if service == nil {
		t.Fatal("expected contracts service")
	}
}

func TestIntegration_Contracts_CreateRentFixture(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "rent")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	if fixture.ownerID == 0 || fixture.clientID == 0 || fixture.agentID == 0 {
		t.Fatalf("expected fixture users, got owner=%d client=%d agent=%d", fixture.ownerID, fixture.clientID, fixture.agentID)
	}

	if fixture.propertyID == 0 || fixture.propertyUUID == "" {
		t.Fatalf("expected fixture property, got id=%d uuid=%q", fixture.propertyID, fixture.propertyUUID)
	}

	if fixture.transactionID == 0 {
		t.Fatal("expected fixture transaction")
	}

	var transactionType string
	var statusID int32
	var propertyTitle string

	err := pool.QueryRow(ctx, `
		SELECT t.transaction_type::text, t.status_id, p.title
		FROM transactions t
		JOIN properties p ON p.property_id = t.property_id
		WHERE t.transaction_id = $1
	`, fixture.transactionID).Scan(&transactionType, &statusID, &propertyTitle)
	if err != nil {
		t.Fatalf("query fixture transaction: %v", err)
	}

	if transactionType != "rent" {
		t.Fatalf("transaction type: got %q, want rent", transactionType)
	}

	if statusID != integrationTransactionStatusPendingID {
		t.Fatalf("transaction status: got %d, want %d", statusID, integrationTransactionStatusPendingID)
	}

	if !strings.Contains(propertyTitle, "Integration") {
		t.Fatalf("property title: got %q", propertyTitle)
	}
}
func TestIntegration_Contracts_GenerateRentContract(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "rent")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	startDate := time.Now().UTC().AddDate(0, 0, 1)
	endDate := startDate.AddDate(0, 1, 0)

	result, err := service.GenerateRentContract(ctx, fixture.clientID, CreateRentContractInput{
		TransactionID: fixture.transactionID,
		PeriodID:      fixture.periodID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err != nil {
		t.Fatalf("generate rent contract: %v", err)
	}

	if result.ContractID == 0 {
		t.Fatal("expected contract id")
	}

	if strings.TrimSpace(result.ContractUUID) == "" {
		t.Fatal("expected contract uuid")
	}

	contractUUID, err := uuid.Parse(result.ContractUUID)
	if err != nil {
		t.Fatalf("parse contract uuid: %v", err)
	}

	if !strings.HasPrefix(result.StorageKey, "contracts/") {
		t.Fatalf("storage key: got %q, want prefix contracts/", result.StorageKey)
	}

	if !strings.HasSuffix(result.StorageKey, ".pdf") {
		t.Fatalf("storage key: got %q, want pdf suffix", result.StorageKey)
	}

	expectedURL := "https://storage.test/" + result.StorageKey
	if result.PDFUrl != expectedURL {
		t.Fatalf("pdf url: got %q, want %q", result.PDFUrl, expectedURL)
	}

	if len(storage.uploadedKeys) != 1 {
		t.Fatalf("uploaded keys: got %d, want 1", len(storage.uploadedKeys))
	}

	if storage.uploadedKeys[0] != result.StorageKey {
		t.Fatalf("uploaded key: got %q, want %q", storage.uploadedKeys[0], result.StorageKey)
	}

	uploadedPDF := storage.uploadedPDFs[result.StorageKey]
	if len(uploadedPDF) == 0 {
		t.Fatal("expected uploaded pdf bytes")
	}

	var storedPeriodID int32
	var storedCurrency string
	var storedAmount float64
	var storedStorageKey string
	var storedStatusID int32

	err = pool.QueryRow(ctx, `
		SELECT
			period_id,
			currency,
			agreed_amount::float8,
			storage_key,
			status_id
		FROM contracts
		WHERE contract_id = $1
		  AND transaction_id = $2
		  AND deleted_at IS NULL
	`, result.ContractID, fixture.transactionID).Scan(
		&storedPeriodID,
		&storedCurrency,
		&storedAmount,
		&storedStorageKey,
		&storedStatusID,
	)
	if err != nil {
		t.Fatalf("query stored rent contract: %v", err)
	}

	if storedPeriodID != fixture.periodID {
		t.Fatalf("stored period id: got %d, want %d", storedPeriodID, fixture.periodID)
	}

	if storedCurrency != "MXN" {
		t.Fatalf("stored currency: got %q, want MXN", storedCurrency)
	}

	if storedAmount != fixture.amount {
		t.Fatalf("stored amount: got %.2f, want %.2f", storedAmount, fixture.amount)
	}

	if storedStorageKey != result.StorageKey {
		t.Fatalf("stored storage key: got %q, want %q", storedStorageKey, result.StorageKey)
	}

	if storedStatusID != integrationContractStatusDraftID {
		t.Fatalf("stored contract status: got %d, want %d", storedStatusID, integrationContractStatusDraftID)
	}

	var transactionStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM transactions
		WHERE transaction_id = $1
	`, fixture.transactionID).Scan(&transactionStatusID); err != nil {
		t.Fatalf("query transaction status: %v", err)
	}

	if transactionStatusID != integrationTransactionStatusClosedID {
		t.Fatalf("transaction status: got %d, want %d", transactionStatusID, integrationTransactionStatusClosedID)
	}

	var propertyStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM properties
		WHERE property_id = $1
	`, fixture.propertyID).Scan(&propertyStatusID); err != nil {
		t.Fatalf("query property status: %v", err)
	}

	if propertyStatusID != integrationPropertyStatusAvailableID {
		t.Fatalf("property status: got %d, want %d", propertyStatusID, integrationPropertyStatusAvailableID)
	}

	detail, err := service.GetContractDetail(ctx, fixture.clientID, integrationRoleClientID, contractUUID)
	if err != nil {
		t.Fatalf("get rent contract detail: %v", err)
	}

	if detail.ContractID != result.ContractID {
		t.Fatalf("detail contract id: got %d, want %d", detail.ContractID, result.ContractID)
	}

	if detail.ContractUUID != result.ContractUUID {
		t.Fatalf("detail contract uuid: got %q, want %q", detail.ContractUUID, result.ContractUUID)
	}

	if detail.PropertyTitle == "" {
		t.Fatal("expected detail property title")
	}

	if detail.OwnerName == "" {
		t.Fatal("expected detail owner name")
	}

	if detail.ClientName == "" {
		t.Fatal("expected detail client name")
	}

	if detail.AgreedAmount != fixture.amount {
		t.Fatalf("detail agreed amount: got %.2f, want %.2f", detail.AgreedAmount, fixture.amount)
	}

	if detail.Currency != "MXN" {
		t.Fatalf("detail currency: got %q, want MXN", detail.Currency)
	}

	if detail.PDFUrl != expectedURL {
		t.Fatalf("detail pdf url: got %q, want %q", detail.PDFUrl, expectedURL)
	}

	if detail.EndDate == nil {
		t.Fatal("expected detail end date for rent contract")
	}
}
func TestIntegration_Contracts_GenerateSaleContract(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "sale")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	result, err := service.GenerateSaleContract(ctx, fixture.agentID, CreateSaleContractInput{
		TransactionID: fixture.transactionID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
	})
	if err != nil {
		t.Fatalf("generate sale contract: %v", err)
	}

	if result.ContractID == 0 {
		t.Fatal("expected contract id")
	}

	if strings.TrimSpace(result.ContractUUID) == "" {
		t.Fatal("expected contract uuid")
	}

	contractUUID, err := uuid.Parse(result.ContractUUID)
	if err != nil {
		t.Fatalf("parse contract uuid: %v", err)
	}

	if !strings.HasPrefix(result.StorageKey, "contracts/") {
		t.Fatalf("storage key: got %q, want prefix contracts/", result.StorageKey)
	}

	if !strings.HasSuffix(result.StorageKey, ".pdf") {
		t.Fatalf("storage key: got %q, want pdf suffix", result.StorageKey)
	}

	expectedURL := "https://storage.test/" + result.StorageKey
	if result.PDFUrl != expectedURL {
		t.Fatalf("pdf url: got %q, want %q", result.PDFUrl, expectedURL)
	}

	if len(storage.uploadedKeys) != 1 {
		t.Fatalf("uploaded keys: got %d, want 1", len(storage.uploadedKeys))
	}

	if storage.uploadedKeys[0] != result.StorageKey {
		t.Fatalf("uploaded key: got %q, want %q", storage.uploadedKeys[0], result.StorageKey)
	}

	uploadedPDF := storage.uploadedPDFs[result.StorageKey]
	if len(uploadedPDF) == 0 {
		t.Fatal("expected uploaded pdf bytes")
	}

	var storedPeriodID *int32
	var storedCurrency string
	var storedAmount float64
	var storedStorageKey string
	var storedStatusID int32
	var storedEndDate *time.Time

	err = pool.QueryRow(ctx, `
		SELECT
			period_id,
			currency,
			agreed_amount::float8,
			storage_key,
			status_id,
			end_date
		FROM contracts
		WHERE contract_id = $1
		  AND transaction_id = $2
		  AND deleted_at IS NULL
	`, result.ContractID, fixture.transactionID).Scan(
		&storedPeriodID,
		&storedCurrency,
		&storedAmount,
		&storedStorageKey,
		&storedStatusID,
		&storedEndDate,
	)
	if err != nil {
		t.Fatalf("query stored sale contract: %v", err)
	}

	if storedPeriodID != nil {
		t.Fatalf("stored period id: got %v, want nil for sale contract", *storedPeriodID)
	}

	if storedCurrency != "MXN" {
		t.Fatalf("stored currency: got %q, want MXN", storedCurrency)
	}

	if storedAmount != fixture.amount {
		t.Fatalf("stored amount: got %.2f, want %.2f", storedAmount, fixture.amount)
	}

	if storedStorageKey != result.StorageKey {
		t.Fatalf("stored storage key: got %q, want %q", storedStorageKey, result.StorageKey)
	}

	if storedStatusID != integrationContractStatusDraftID {
		t.Fatalf("stored contract status: got %d, want %d", storedStatusID, integrationContractStatusDraftID)
	}

	if storedEndDate != nil {
		t.Fatalf("stored end date: got %v, want nil for sale contract", storedEndDate)
	}

	var transactionStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM transactions
		WHERE transaction_id = $1
	`, fixture.transactionID).Scan(&transactionStatusID); err != nil {
		t.Fatalf("query transaction status: %v", err)
	}

	if transactionStatusID != integrationTransactionStatusClosedID {
		t.Fatalf("transaction status: got %d, want %d", transactionStatusID, integrationTransactionStatusClosedID)
	}

	var propertyStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM properties
		WHERE property_id = $1
	`, fixture.propertyID).Scan(&propertyStatusID); err != nil {
		t.Fatalf("query property status: %v", err)
	}

	if propertyStatusID != integrationPropertyStatusSoldID {
		t.Fatalf("property status: got %d, want %d", propertyStatusID, integrationPropertyStatusSoldID)
	}

	detail, err := service.GetContractDetail(ctx, fixture.agentID, integrationRoleAgentID, contractUUID)
	if err != nil {
		t.Fatalf("get sale contract detail: %v", err)
	}

	if detail.ContractID != result.ContractID {
		t.Fatalf("detail contract id: got %d, want %d", detail.ContractID, result.ContractID)
	}

	if detail.ContractUUID != result.ContractUUID {
		t.Fatalf("detail contract uuid: got %q, want %q", detail.ContractUUID, result.ContractUUID)
	}

	if detail.PropertyTitle == "" {
		t.Fatal("expected detail property title")
	}

	if detail.OwnerName == "" {
		t.Fatal("expected detail owner name")
	}

	if detail.ClientName == "" {
		t.Fatal("expected detail client name")
	}

	if detail.AgreedAmount != fixture.amount {
		t.Fatalf("detail agreed amount: got %.2f, want %.2f", detail.AgreedAmount, fixture.amount)
	}

	if detail.Currency != "MXN" {
		t.Fatalf("detail currency: got %q, want MXN", detail.Currency)
	}

	if detail.PDFUrl != expectedURL {
		t.Fatalf("detail pdf url: got %q, want %q", detail.PDFUrl, expectedURL)
	}

	if detail.EndDate != nil {
		t.Fatalf("expected nil detail end date for sale contract, got %v", detail.EndDate)
	}
}
func TestIntegration_Contracts_GenerateRentContractRejectsUnauthorizedUser(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "rent")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	startDate := time.Now().UTC().AddDate(0, 0, 1)
	endDate := startDate.AddDate(0, 1, 0)

	_, err := service.GenerateRentContract(ctx, fixture.agentID, CreateRentContractInput{
		TransactionID: fixture.transactionID,
		PeriodID:      fixture.periodID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err == nil {
		t.Fatal("expected unauthorized rent contract error")
	}

	if len(storage.uploadedKeys) != 0 {
		t.Fatalf("uploaded keys: got %d, want 0", len(storage.uploadedKeys))
	}

	var contractCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contracts
		WHERE transaction_id = $1
		  AND deleted_at IS NULL
	`, fixture.transactionID).Scan(&contractCount); err != nil {
		t.Fatalf("count contracts: %v", err)
	}

	if contractCount != 0 {
		t.Fatalf("contract count: got %d, want 0", contractCount)
	}

	var transactionStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM transactions
		WHERE transaction_id = $1
	`, fixture.transactionID).Scan(&transactionStatusID); err != nil {
		t.Fatalf("query transaction status: %v", err)
	}

	if transactionStatusID != integrationTransactionStatusPendingID {
		t.Fatalf("transaction status: got %d, want %d", transactionStatusID, integrationTransactionStatusPendingID)
	}

	var propertyStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM properties
		WHERE property_id = $1
	`, fixture.propertyID).Scan(&propertyStatusID); err != nil {
		t.Fatalf("query property status: %v", err)
	}

	if propertyStatusID != integrationPropertyStatusAvailableID {
		t.Fatalf("property status: got %d, want %d", propertyStatusID, integrationPropertyStatusAvailableID)
	}
}

func TestIntegration_Contracts_GenerateSaleContractRejectsUnauthorizedUser(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "sale")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	_, err := service.GenerateSaleContract(ctx, fixture.clientID, CreateSaleContractInput{
		TransactionID: fixture.transactionID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
	})
	if err == nil {
		t.Fatal("expected unauthorized sale contract error")
	}

	if len(storage.uploadedKeys) != 0 {
		t.Fatalf("uploaded keys: got %d, want 0", len(storage.uploadedKeys))
	}

	var contractCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contracts
		WHERE transaction_id = $1
		  AND deleted_at IS NULL
	`, fixture.transactionID).Scan(&contractCount); err != nil {
		t.Fatalf("count contracts: %v", err)
	}

	if contractCount != 0 {
		t.Fatalf("contract count: got %d, want 0", contractCount)
	}

	var transactionStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM transactions
		WHERE transaction_id = $1
	`, fixture.transactionID).Scan(&transactionStatusID); err != nil {
		t.Fatalf("query transaction status: %v", err)
	}

	if transactionStatusID != integrationTransactionStatusPendingID {
		t.Fatalf("transaction status: got %d, want %d", transactionStatusID, integrationTransactionStatusPendingID)
	}

	var propertyStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM properties
		WHERE property_id = $1
	`, fixture.propertyID).Scan(&propertyStatusID); err != nil {
		t.Fatalf("query property status: %v", err)
	}

	if propertyStatusID != integrationPropertyStatusAvailableID {
		t.Fatalf("property status: got %d, want %d", propertyStatusID, integrationPropertyStatusAvailableID)
	}
}
func TestIntegration_Contracts_GenerateRentContractRejectsDuplicateContract(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "rent")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	startDate := time.Now().UTC().AddDate(0, 0, 1)
	endDate := startDate.AddDate(0, 1, 0)

	firstResult, err := service.GenerateRentContract(ctx, fixture.clientID, CreateRentContractInput{
		TransactionID: fixture.transactionID,
		PeriodID:      fixture.periodID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err != nil {
		t.Fatalf("generate first rent contract: %v", err)
	}

	if firstResult.ContractID == 0 {
		t.Fatal("expected first contract id")
	}

	_, err = service.GenerateRentContract(ctx, fixture.clientID, CreateRentContractInput{
		TransactionID: fixture.transactionID,
		PeriodID:      fixture.periodID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err == nil {
		t.Fatal("expected duplicate rent contract error")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "contrato") &&
		!strings.Contains(strings.ToLower(err.Error()), "contract") {
		t.Fatalf("duplicate error: got %q, want contract-related error", err.Error())
	}

	var contractCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contracts
		WHERE transaction_id = $1
		  AND deleted_at IS NULL
	`, fixture.transactionID).Scan(&contractCount); err != nil {
		t.Fatalf("count contracts: %v", err)
	}

	if contractCount != 1 {
		t.Fatalf("contract count: got %d, want 1", contractCount)
	}

	if len(storage.uploadedKeys) != 1 {
		t.Fatalf("uploaded keys: got %d, want 1", len(storage.uploadedKeys))
	}
}

func TestIntegration_Contracts_GenerateSaleContractRejectsDuplicateContract(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	fixture := createContractIntegrationFixture(t, ctx, pool, "sale")
	defer cleanupContractIntegrationFixture(t, ctx, pool, fixture)

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	firstResult, err := service.GenerateSaleContract(ctx, fixture.agentID, CreateSaleContractInput{
		TransactionID: fixture.transactionID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
	})
	if err != nil {
		t.Fatalf("generate first sale contract: %v", err)
	}

	if firstResult.ContractID == 0 {
		t.Fatal("expected first contract id")
	}

	_, err = service.GenerateSaleContract(ctx, fixture.agentID, CreateSaleContractInput{
		TransactionID: fixture.transactionID,
		Currency:      "MXN",
		AgreedAmount:  fixture.amount,
	})
	if err == nil {
		t.Fatal("expected duplicate sale contract error")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "contrato") &&
		!strings.Contains(strings.ToLower(err.Error()), "contract") {
		t.Fatalf("duplicate error: got %q, want contract-related error", err.Error())
	}

	var contractCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contracts
		WHERE transaction_id = $1
		  AND deleted_at IS NULL
	`, fixture.transactionID).Scan(&contractCount); err != nil {
		t.Fatalf("count contracts: %v", err)
	}

	if contractCount != 1 {
		t.Fatalf("contract count: got %d, want 1", contractCount)
	}

	if len(storage.uploadedKeys) != 1 {
		t.Fatalf("uploaded keys: got %d, want 1", len(storage.uploadedKeys))
	}

	var propertyStatusID int32
	if err := pool.QueryRow(ctx, `
		SELECT status_id
		FROM properties
		WHERE property_id = $1
	`, fixture.propertyID).Scan(&propertyStatusID); err != nil {
		t.Fatalf("query property status: %v", err)
	}

	if propertyStatusID != integrationPropertyStatusSoldID {
		t.Fatalf("property status: got %d, want %d", propertyStatusID, integrationPropertyStatusSoldID)
	}
}
func TestIntegration_Contracts_ListContractsAndGetDetail(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()

	seedContractIntegrationCatalogs(t, ctx, pool)

	rentFixture := createContractIntegrationFixture(t, ctx, pool, "rent")
	defer cleanupContractIntegrationFixture(t, ctx, pool, rentFixture)

	saleFixture := createContractIntegrationFixture(t, ctx, pool, "sale")
	defer cleanupContractIntegrationFixture(t, ctx, pool, saleFixture)

	unrelatedClientID := insertIntegrationUser(t, ctx, pool, integrationRoleClientID, "UnrelatedClient", time.Now().UnixNano())
	defer func() {
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE user_id = $1`, unrelatedClientID); err != nil {
			t.Fatalf("cleanup unrelated integration client: %v", err)
		}
	}()

	repo := NewRepository(pool)
	storage := newIntegrationContractStorage()
	service := NewService(repo, storage)

	startDate := time.Now().UTC().AddDate(0, 0, 1)
	endDate := startDate.AddDate(0, 1, 0)

	rentResult, err := service.GenerateRentContract(ctx, rentFixture.clientID, CreateRentContractInput{
		TransactionID: rentFixture.transactionID,
		PeriodID:      rentFixture.periodID,
		Currency:      "MXN",
		AgreedAmount:  rentFixture.amount,
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err != nil {
		t.Fatalf("generate rent contract for list test: %v", err)
	}

	saleResult, err := service.GenerateSaleContract(ctx, saleFixture.agentID, CreateSaleContractInput{
		TransactionID: saleFixture.transactionID,
		Currency:      "MXN",
		AgreedAmount:  saleFixture.amount,
	})
	if err != nil {
		t.Fatalf("generate sale contract for list test: %v", err)
	}

	rentContractUUID, err := uuid.Parse(rentResult.ContractUUID)
	if err != nil {
		t.Fatalf("parse rent contract uuid: %v", err)
	}

	saleContractUUID, err := uuid.Parse(saleResult.ContractUUID)
	if err != nil {
		t.Fatalf("parse sale contract uuid: %v", err)
	}

	adminList, err := service.ListContracts(ctx, rentFixture.ownerID, integrationRoleAdminID, ListContractsFilter{
		Page:  1,
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("list contracts as admin: %v", err)
	}

	if !contractIntegrationListContains(adminList, rentResult.ContractUUID) {
		t.Fatalf("admin list does not contain rent contract %s", rentResult.ContractUUID)
	}

	if !contractIntegrationListContains(adminList, saleResult.ContractUUID) {
		t.Fatalf("admin list does not contain sale contract %s", saleResult.ContractUUID)
	}

	rentType := "rent"
	rentList, err := service.ListContracts(ctx, rentFixture.ownerID, integrationRoleAdminID, ListContractsFilter{
		TransactionType: &rentType,
		Page:            1,
		Limit:           20,
	})
	if err != nil {
		t.Fatalf("list rent contracts as admin: %v", err)
	}

	if !contractIntegrationListContains(rentList, rentResult.ContractUUID) {
		t.Fatalf("rent filtered list does not contain rent contract %s", rentResult.ContractUUID)
	}

	if contractIntegrationListContains(rentList, saleResult.ContractUUID) {
		t.Fatalf("rent filtered list should not contain sale contract %s", saleResult.ContractUUID)
	}

	clientList, err := service.ListContracts(ctx, rentFixture.clientID, integrationRoleClientID, ListContractsFilter{
		Page:  1,
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("list contracts as rent client: %v", err)
	}

	if !contractIntegrationListContains(clientList, rentResult.ContractUUID) {
		t.Fatalf("client list does not contain own rent contract %s", rentResult.ContractUUID)
	}

	if contractIntegrationListContains(clientList, saleResult.ContractUUID) {
		t.Fatalf("client list should not contain unrelated sale contract %s", saleResult.ContractUUID)
	}

	_, err = service.GetContractDetail(ctx, unrelatedClientID, integrationRoleClientID, rentContractUUID)
	if err == nil {
		t.Fatal("expected unrelated client detail access error")
	}

	saleDetail, err := service.GetContractDetail(ctx, rentFixture.ownerID, integrationRoleAdminID, saleContractUUID)
	if err != nil {
		t.Fatalf("get sale contract detail as admin: %v", err)
	}

	if saleDetail.ContractUUID != saleResult.ContractUUID {
		t.Fatalf("sale detail uuid: got %q, want %q", saleDetail.ContractUUID, saleResult.ContractUUID)
	}

	if saleDetail.AgreedAmount != saleFixture.amount {
		t.Fatalf("sale detail amount: got %.2f, want %.2f", saleDetail.AgreedAmount, saleFixture.amount)
	}

	if saleDetail.EndDate != nil {
		t.Fatalf("sale detail end date: got %v, want nil", saleDetail.EndDate)
	}

	expectedSaleURL := "https://storage.test/" + saleResult.StorageKey
	if saleDetail.PDFUrl != expectedSaleURL {
		t.Fatalf("sale detail pdf url: got %q, want %q", saleDetail.PDFUrl, expectedSaleURL)
	}

	if len(storage.uploadedKeys) != 2 {
		t.Fatalf("uploaded keys: got %d, want 2", len(storage.uploadedKeys))
	}
}
func seedContractIntegrationCatalogs(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	statements := []string{
		`INSERT INTO roles (role_id, name) VALUES
			(1, 'admin'),
			(2, 'agent'),
			(3, 'client')
		ON CONFLICT (role_id) DO NOTHING`,

		`INSERT INTO user_status (status_id, name) VALUES
			(1, 'Active'),
			(3, 'Pending')
		ON CONFLICT (status_id) DO NOTHING`,

		`INSERT INTO property_status (status_id, name) VALUES
			(1, 'Reserved'),
			(2, 'Available'),
			(3, 'Sold'),
			(4, 'Rented')
		ON CONFLICT (status_id) DO NOTHING`,

		`INSERT INTO transaction_status (status_id, name) VALUES
			(1, 'Pending'),
			(2, 'In Progress'),
			(3, 'Closed'),
			(4, 'Cancelled')
		ON CONFLICT (status_id) DO NOTHING`,

		`INSERT INTO contract_status (status_id, name) VALUES
			(1, 'Draft'),
			(2, 'Active'),
			(3, 'Expired'),
			(4, 'Terminated')
		ON CONFLICT (status_id) DO NOTHING`,

		`INSERT INTO rent_periods (period_id, name) VALUES
			(1, 'Daily'),
			(2, 'Weekly'),
			(3, 'Monthly'),
			(4, 'Yearly')
		ON CONFLICT (period_id) DO NOTHING`,

		`INSERT INTO property_types (property_type_id, name, icon, subtype, is_deprecated)
		VALUES ($$9001$$, 'Integration House', 'house', 'residential', false)
		ON CONFLICT (property_type_id) DO UPDATE
		SET name = EXCLUDED.name,
			icon = EXCLUDED.icon,
			subtype = EXCLUDED.subtype,
			is_deprecated = EXCLUDED.is_deprecated`,

		`INSERT INTO modalities (modality_id, name) VALUES
			(9001, 'Sale'),
			(9002, 'Rent')
		ON CONFLICT (modality_id) DO NOTHING`,

		`INSERT INTO property_type_periods (property_type_id, period_id)
		VALUES (9001, 3)
		ON CONFLICT (property_type_id, period_id) DO NOTHING`,

		`INSERT INTO countries (country_id, iso2_code, name, is_active)
		VALUES (9001, 'ZZ', 'Country Integration Contracts', true)
		ON CONFLICT (country_id) DO NOTHING`,

		`INSERT INTO states (state_id, country_id, iso_code, name, is_active)
		VALUES (9001, 9001, 'VER-IT', 'Veracruz Integration', true)
		ON CONFLICT (state_id) DO NOTHING`,

		`INSERT INTO cities (city_id, state_id, name)
		VALUES (9001, 9001, 'Xalapa Integration')
		ON CONFLICT (city_id) DO NOTHING`,

		`INSERT INTO orientations (orientation_id, name)
		VALUES (9001, 'North Integration')
		ON CONFLICT (orientation_id) DO NOTHING`,
	}

	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("seed integration catalog: %v\nstatement:\n%s", err, statement)
		}
	}
	resetContractIntegrationSequence(t, ctx, pool, "users", "user_id")
	resetContractIntegrationSequence(t, ctx, pool, "properties", "property_id")
	resetContractIntegrationSequence(t, ctx, pool, "rent_prices", "rent_price_id")
	resetContractIntegrationSequence(t, ctx, pool, "sale_prices", "price_id")
	resetContractIntegrationSequence(t, ctx, pool, "transactions", "transaction_id")
	resetContractIntegrationSequence(t, ctx, pool, "contracts", "contract_id")
}

func createContractIntegrationFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, transactionType string) integrationContractFixture {
	t.Helper()

	suffix := time.Now().UnixNano()

	ownerID := insertIntegrationUser(t, ctx, pool, integrationRoleAdminID, "Owner", suffix)
	clientID := insertIntegrationUser(t, ctx, pool, integrationRoleClientID, "Client", suffix)
	agentID := insertIntegrationUser(t, ctx, pool, integrationRoleAgentID, "Agent", suffix)

	modalityID := integrationModalityRentID
	amount := 8000.00

	if transactionType == "sale" {
		modalityID = integrationModalitySaleID
		amount = 1500000.00
	}

	propertyUUID := uuid.New().String()

	var propertyID int32
	err := pool.QueryRow(ctx, `
		INSERT INTO properties (
			property_uuid,
			owner_id,
			title,
			description,
			property_type_id,
			modality_id,
			status_id,
			cover_photo_url,
			lot_area,
			is_featured,
			published_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 120.00, false, now(), now())
		RETURNING property_id
	`,
		propertyUUID,
		ownerID,
		fmt.Sprintf("Integration Contract Property %d", suffix),
		"Property created by contracts integration tests",
		integrationPropertyTypeHouseID,
		modalityID,
		integrationPropertyStatusAvailableID,
		fmt.Sprintf("properties/integration/%d/cover.webp", suffix),
	).Scan(&propertyID)
	if err != nil {
		t.Fatalf("insert integration property: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO residential_properties (
			property_id,
			bedrooms,
			bathrooms,
			beds,
			floors,
			parking_spots,
			built_area,
			construction_year,
			orientation_id,
			is_furnished
		)
		VALUES ($1, 3, 2, 3, 2, 1, 120.00, 2020, $2, false)
	`, propertyID, integrationOrientationID); err != nil {
		t.Fatalf("insert integration residential property: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO locations (
			property_id,
			city_id,
			neighborhood,
			street,
			exterior_number,
			postal_code,
			coordinates,
			is_public_address
		)
		VALUES (
			$1,
			$2,
			'Colonia Integration',
			'Calle Integration',
			'123',
			'91000',
			ST_SetSRID(ST_MakePoint(-96.9102, 19.5438), 4326),
			true
		)
	`, propertyID, integrationCityID); err != nil {
		t.Fatalf("insert integration location: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE properties
		SET agent_id = $2
		WHERE property_id = $1
	`, propertyID, agentID); err != nil {
		t.Fatalf("assign integration property agent: %v", err)
	}

	if transactionType == "rent" {
		if _, err := pool.Exec(ctx, `
			INSERT INTO rent_prices (
				property_id,
				period_id,
				rent_price,
				deposit,
				currency,
				is_negotiable,
				is_current,
				valid_from,
				changed_by_user_id
			)
			VALUES ($1, $2, $3, $4, 'MXN', false, true, now(), $5)
		`, propertyID, integrationRentPeriodMonthlyID, amount, amount*2, ownerID); err != nil {
			t.Fatalf("insert integration rent price: %v", err)
		}
	} else {
		if _, err := pool.Exec(ctx, `
			INSERT INTO sale_prices (
				property_id,
				sale_price,
				currency,
				is_negotiable,
				is_current,
				valid_from,
				changed_by_user_id
			)
			VALUES ($1, $2, 'MXN', true, true, now(), $3)
		`, propertyID, amount, ownerID); err != nil {
			t.Fatalf("insert integration sale price: %v", err)
		}
	}

	var transactionID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO transactions (
			property_id,
			client_id,
			agent_id,
			transaction_type,
			status_id,
			final_amount,
			closing_date
		)
		VALUES ($1, $2, $3, $4::transaction_type, $5, $6, $7)
		RETURNING transaction_id
	`,
		propertyID,
		clientID,
		agentID,
		transactionType,
		integrationTransactionStatusPendingID,
		amount,
		time.Now().AddDate(0, 0, 7),
	).Scan(&transactionID)
	if err != nil {
		t.Fatalf("insert integration transaction: %v", err)
	}

	return integrationContractFixture{
		ownerID:       ownerID,
		clientID:      clientID,
		agentID:       agentID,
		propertyID:    propertyID,
		propertyUUID:  propertyUUID,
		transactionID: transactionID,
		amount:        amount,
		periodID:      integrationRentPeriodMonthlyID,
	}
}

func insertIntegrationUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleID int32, label string, suffix int64) int32 {
	t.Helper()

	userUUID := uuid.New().String()
	email := fmt.Sprintf("integration-contract-%s-%d@example.com", strings.ToLower(label), suffix)

	columns := []string{
		"user_uuid",
		"role_id",
		"first_name",
		"last_name",
		"email",
		"phone",
		"status_id",
	}
	values := []any{
		userUUID,
		roleID,
		label,
		"Integration",
		email,
		"5550000000",
		integrationUserStatusActiveID,
	}

	if contractIntegrationColumnExists(t, ctx, pool, "users", "password") {
		columns = append(columns, "password")
		values = append(values, "hashed-password")
	}

	if contractIntegrationColumnExists(t, ctx, pool, "users", "password_hash") {
		columns = append(columns, "password_hash")
		values = append(values, "hashed-password")
	}

	if contractIntegrationColumnExists(t, ctx, pool, "users", "profile_picture_url") {
		columns = append(columns, "profile_picture_url")
		values = append(values, "")
	}

	placeholders := make([]string, 0, len(values))
	for index := range values {
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
	}

	query := fmt.Sprintf(
		"INSERT INTO users (%s) VALUES (%s) RETURNING user_id",
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	var userID int32
	if err := pool.QueryRow(ctx, query, values...).Scan(&userID); err != nil {
		t.Fatalf("insert integration user %s: %v\nquery:\n%s", label, err, query)
	}

	return userID
}
func contractIntegrationColumnExists(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	tableName string,
	columnName string,
) bool {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists); err != nil {
		t.Fatalf("check column %s.%s: %v", tableName, columnName, err)
	}

	return exists
}
func resetContractIntegrationSequence(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	tableName string,
	columnName string,
) {
	t.Helper()

	query := fmt.Sprintf(`
		SELECT setval(
			pg_get_serial_sequence('%s', '%s')::regclass,
			COALESCE((SELECT MAX(%s) FROM %s), 0) + 1,
			false
		)
	`, tableName, columnName, columnName, tableName)

	if _, err := pool.Exec(ctx, query); err != nil {
		t.Fatalf("reset sequence %s.%s: %v", tableName, columnName, err)
	}
}
func cleanupContractIntegrationFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture integrationContractFixture) {
	t.Helper()

	statements := []struct {
		query string
		args  []any
	}{
		{query: `DELETE FROM payments WHERE contract_id IN (SELECT contract_id FROM contracts WHERE transaction_id = $1)`, args: []any{fixture.transactionID}},
		{query: `DELETE FROM contracts WHERE transaction_id = $1`, args: []any{fixture.transactionID}},
		{query: `DELETE FROM transactions WHERE transaction_id = $1`, args: []any{fixture.transactionID}},
		{query: `DELETE FROM property_status_history WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM property_services WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM property_clauses WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM property_photos WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM rent_prices WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM sale_prices WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM locations WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM residential_properties WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM commercial_properties WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `UPDATE properties SET agent_id = NULL WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM properties WHERE property_id = $1`, args: []any{fixture.propertyID}},
		{query: `DELETE FROM users WHERE user_id = $1`, args: []any{fixture.ownerID}},
		{query: `DELETE FROM users WHERE user_id = $1`, args: []any{fixture.clientID}},
		{query: `DELETE FROM users WHERE user_id = $1`, args: []any{fixture.agentID}},
	}

	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("cleanup integration fixture: %v\nquery:\n%s", err, statement.query)
		}
	}
}
func contractIntegrationListContains(items []ContractListItem, contractUUID string) bool {
	for _, item := range items {
		if item.ContractUUID == contractUUID {
			return true
		}
	}

	return false
}
