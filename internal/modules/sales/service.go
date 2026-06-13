package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type service struct {
	repo            Repository
	contractsClient ContractsClient
}

func NewService(repo Repository, contractsClient ContractsClient) Service {
	return &service{
		repo:            repo,
		contractsClient: contractsClient,
	}
}

func (s *service) ConfirmSale(ctx context.Context, auth AuthContext, input SaleInput) (SaleResponse, error) {
	if auth.RoleID != roleAgentID {
		return SaleResponse{}, newStatusError(http.StatusForbidden, "only authenticated agents can sell properties")
	}

	property, err := s.repo.GetSalePropertyByUUID(ctx, input.PropertyUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SaleResponse{}, newStatusError(http.StatusNotFound, "property not found")
		}

		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not load property")
	}

	if property.ModalityID != modalitySale && property.ModalityID != modalityMixed {
		return SaleResponse{}, newStatusError(http.StatusUnprocessableEntity, "property is not available for sale")
	}

	if property.StatusID != propertyStatusAvailable {
		return SaleResponse{}, newStatusError(http.StatusUnprocessableEntity, "property is not currently available")
	}

	price, err := s.repo.GetCurrentSalePriceByPropertyID(ctx, property.PropertyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SaleResponse{}, newStatusError(http.StatusUnprocessableEntity, "no current sale price exists for this property")
		}

		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not load current sale price")
	}

	expectedCents, err := numericToCents(price.SalePrice)
	if err != nil {
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not parse current sale price")
	}

	agreedAmountCents := amountToCents(input.AgreedAmount)
	if agreedAmountCents != expectedCents {
		return SaleResponse{}, newStatusError(
			http.StatusUnprocessableEntity,
			"agreed_amount must match current sale price exactly; expected %.2f %s",
			centsToFloat(expectedCents),
			strings.TrimSpace(price.Currency),
		)
	}

	insertTx, err := s.repo.Begin(ctx)
	if err != nil {
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not start sale transaction")
	}
	defer insertTx.Rollback(ctx)

	insertRepo := s.repo.WithTx(insertTx)
	transaction, err := insertRepo.CreateSaleTransaction(ctx, sqlcgen.CreateSaleTransactionParams{
		PropertyID:  property.PropertyID,
		AgentID:     auth.UserID,
		StatusID:    transactionStatusPending,
		FinalAmount: numericFromCents(agreedAmountCents),
	})
	if err != nil {
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not create sale transaction")
	}

	if err := insertTx.Commit(ctx); err != nil {
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not commit sale transaction")
	}

	contractResult, err := s.contractsClient.CreateSaleContract(ctx, auth.AuthHeader, ContractCreateInput{
		TransactionID: transaction.TransactionID,
		Currency:      strings.TrimSpace(price.Currency),
		AgreedAmount:  centsToFloat(agreedAmountCents),
	})
	if err != nil {
		log.Printf("sales: contract creation failed after transaction commit, transaction_id=%d transaction_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), err)
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "could not generate sale contract")
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		log.Printf("sales: contract created but local transaction could not start, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "contract created but property status history failed")
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)
	if err := txRepo.CreateSalePropertyStatusHistory(ctx, sqlcgen.CreateSalePropertyStatusHistoryParams{
		PropertyID:       property.PropertyID,
		PreviousStatusID: propertyStatusAvailable,
		NewStatusID:      propertyStatusSold,
		ChangedByUserID:  auth.UserID,
	}); err != nil {
		log.Printf("sales: property status history failed after contract creation, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "contract created but property status history failed")
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("sales: commit failed after contract creation, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return SaleResponse{}, newStatusError(http.StatusInternalServerError, "contract created but local commit failed")
	}

	return SaleResponse{
		TransactionUUID: formatUUIDValue(transaction.TransactionUuid),
		ContractUUID:    contractResult.ContractUUID,
		PropertyUUID:    property.PropertyUuid.String(),
		Status:          "formalized",
		FinalAmount:     centsToFloat(agreedAmountCents),
		Currency:        strings.TrimSpace(price.Currency),
	}, nil
}

func NewHTTPContractsClient(baseURL string) ContractsClient {
	return &httpContractsClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *httpContractsClient) CreateSaleContract(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return ContractCreateResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/contracts/sale", bytes.NewReader(payload))
	if err != nil {
		return ContractCreateResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	resp, err := c.client.Do(req)
	if err != nil {
		return ContractCreateResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return ContractCreateResult{}, fmt.Errorf("contracts endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		return ContractCreateResult{}, fmt.Errorf("contracts endpoint returned status %d", resp.StatusCode)
	}

	var result ContractCreateResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ContractCreateResult{}, err
	}

	return result, nil
}

func numericToCents(value pgtype.Numeric) (int64, error) {
	floatValue, err := value.Float64Value()
	if err != nil {
		return 0, err
	}
	if !floatValue.Valid {
		return 0, fmt.Errorf("numeric value is invalid")
	}

	return amountToCents(floatValue.Float64), nil
}

func numericFromCents(cents int64) pgtype.Numeric {
	var value pgtype.Numeric
	_ = value.Scan(fmt.Sprintf("%.2f", centsToFloat(cents)))
	return value
}

func amountToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func centsToFloat(cents int64) float64 {
	return float64(cents) / 100
}

func formatUUIDValue(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", value.Bytes[0:4], value.Bytes[4:6], value.Bytes[6:8], value.Bytes[8:10], value.Bytes[10:16])
}
