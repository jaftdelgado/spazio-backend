package rentals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

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

func (s *service) PreviewRental(ctx context.Context, auth AuthContext, input RentalPreviewInput) (RentalPreviewResponse, error) {
	property, pricing, blockedDates, err := s.validateRental(ctx, auth, input)
	if err != nil {
		return RentalPreviewResponse{}, err
	}

	return RentalPreviewResponse{
		PropertyUUID:    property.PropertyUuid.String(),
		Period:          pricing.RequestedPeriodName,
		PeriodID:        pricing.RequestedPeriodID,
		StartDate:       formatDate(input.StartDate),
		EndDate:         formatDate(input.EndDate),
		Units:           pricing.Units,
		UnitPrice:       centsToString(pricing.UnitPriceCents),
		Currency:        pricing.Currency,
		Subtotal:        centsToString(pricing.SubtotalCents),
		Deposit:         centsToString(pricing.DepositCents),
		Total:           centsToString(pricing.TotalCents),
		IsNegotiable:    pricing.IsNegotiable,
		BlockedDates:    blockedDates,
		Breakdown:       pricing.Breakdown,
		PriceComponents: pricing.PriceComponents,
	}, nil
}

func (s *service) ConfirmRental(ctx context.Context, auth AuthContext, input RentalConfirmInput) (RentalResponse, error) {
	if auth.UserUUID != input.ClientUUID {
		return RentalResponse{}, newStatusError(http.StatusForbidden, "client_uuid must match the authenticated client")
	}

	property, pricing, _, err := s.validateRental(ctx, auth, RentalPreviewInput{
		PropertyUUID: input.PropertyUUID,
		PeriodID:     input.PeriodID,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
	})
	if err != nil {
		return RentalResponse{}, err
	}

	agentID, err := s.repo.GetPrimaryRentalAgentForProperty(ctx, property.PropertyID)
	if err != nil || agentID <= 0 {
		return RentalResponse{}, newStatusError(http.StatusUnprocessableEntity, "the property does not have a primary agent assigned")
	}

	totalNumeric := numericFromCents(pricing.TotalCents)

	insertTx, err := s.repo.Begin(ctx)
	if err != nil {
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "could not start rental transaction")
	}
	defer insertTx.Rollback(ctx)

	insertRepo := s.repo.WithTx(insertTx)
	transaction, err := insertRepo.CreateRentalTransaction(ctx, createRentalTransactionParams(property.PropertyID, auth.UserID, agentID, input.EndDate, totalNumeric))
	if err != nil {
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "could not create rental transaction")
	}

	if err := insertTx.Commit(ctx); err != nil {
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "could not commit rental transaction")
	}

	agreedAmountCents, err := numericToCents(transaction.FinalAmount)
	if err != nil {
		log.Printf("rentals: committed transaction amount could not be read, transaction_id=%d transaction_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "could not prepare rental contract")
	}

	contractResult, err := s.contractsClient.CreateContract(ctx, auth.AuthHeader, ContractCreateInput{
		TransactionID: transaction.TransactionID,
		PeriodID:      input.PeriodID,
		Currency:      pricing.Currency,
		AgreedAmount:  centsToFloat(agreedAmountCents),
		StartDate:     input.StartDate.UTC(),
		EndDate:       input.EndDate.UTC(),
	})
	if err != nil {
		log.Printf("rentals: contract creation failed after transaction commit, transaction_id=%d transaction_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "could not generate rental contract")
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		log.Printf("rentals: contract created but local transaction could not start, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "contract created but property status update failed")
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)
	if err := txRepo.UpdateRentalPropertyStatus(ctx, property.PropertyID, PropertyStatusReserved); err != nil {
		log.Printf("rentals: property status update failed after contract creation, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "contract created but property status update failed")
	}

	if err := txRepo.CreateRentalPropertyStatusHistory(ctx, sqlcgen.CreateRentalPropertyStatusHistoryParams{
		PropertyID:       property.PropertyID,
		PreviousStatusID: PropertyStatusAvailable,
		NewStatusID:      PropertyStatusReserved,
		ChangedByUserID:  auth.UserID,
	}); err != nil {
		log.Printf("rentals: property status history failed after contract creation, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "contract created but property status history failed")
	}

	if err := txRepo.UpdateRentalTransactionStatus(ctx, transaction.TransactionID, TransactionStatusCompleted); err != nil {
		log.Printf("rentals: transaction completion failed after contract creation, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "contract created but transaction finalization failed")
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("rentals: commit failed after contract creation, transaction_id=%d transaction_uuid=%s contract_uuid=%s err=%v", transaction.TransactionID, formatUUIDValue(transaction.TransactionUuid), contractResult.ContractUUID, err)
		return RentalResponse{}, newStatusError(http.StatusInternalServerError, "contract created but local commit failed")
	}

	return RentalResponse{
		TransactionUUID: transaction.TransactionUuid.String(),
		ContractUUID:    contractResult.ContractUUID,
		PropertyUUID:    property.PropertyUuid.String(),
		Status:          "Completed",
		Period:          pricing.RequestedPeriodName,
		PeriodID:        pricing.RequestedPeriodID,
		StartDate:       formatDate(input.StartDate),
		EndDate:         formatDate(input.EndDate),
		Currency:        pricing.Currency,
		Subtotal:        centsToString(pricing.SubtotalCents),
		Deposit:         centsToString(pricing.DepositCents),
		Total:           centsToString(pricing.TotalCents),
		IsNegotiable:    pricing.IsNegotiable,
		Breakdown:       pricing.Breakdown,
		PriceComponents: pricing.PriceComponents,
	}, nil
}

func (s *service) validateRental(ctx context.Context, auth AuthContext, input RentalPreviewInput) (sqlcgen.GetRentalPropertyByUUIDRow, pricingDetails, []string, error) {
	if auth.RoleID != 3 {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusForbidden, "only authenticated clients can rent properties")
	}

	property, err := s.repo.GetRentalPropertyByUUID(ctx, input.PropertyUUID)
	if err != nil {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusNotFound, "property not found")
	}

	if property.ModalityID != ModalityRent && property.ModalityID != ModalityMixed {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusUnprocessableEntity, "property is not available for rent")
	}

	if property.StatusID != PropertyStatusAvailable {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusUnprocessableEntity, "property is not currently available")
	}

	allowedPeriods, err := s.repo.GetAllowedRentalPeriods(ctx, property.PropertyTypeID)
	if err != nil {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusInternalServerError, "could not validate allowed rent periods")
	}
	if !containsInt32(allowedPeriods, input.PeriodID) {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusUnprocessableEntity, "period_id is not valid for this property type")
	}

	prices, err := s.repo.ListRentalActivePrices(ctx, property.PropertyID)
	if err != nil {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusInternalServerError, "could not load current rent prices")
	}
	if !hasPriceForPeriod(prices, input.PeriodID) {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusUnprocessableEntity, "no current rent price exists for the requested period")
	}

	if !input.StartDate.Before(input.EndDate) {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusBadRequest, "start_date must be before end_date")
	}

	blockedRows, err := s.repo.ListRentalBlockedDates(ctx, property.PropertyID, input.StartDate, input.EndDate)
	if err != nil {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusInternalServerError, "could not validate blocked dates")
	}

	blockedDates := make([]string, 0, len(blockedRows))
	for _, blocked := range blockedRows {
		blockedDates = append(blockedDates, blocked.ExceptionDate.Time.Format(dateLayout))
	}
	if len(blockedDates) > 0 {
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, blockedDates, newStatusError(http.StatusUnprocessableEntity, "requested rental dates overlap blocked dates: %s", strings.Join(blockedDates, ", "))
	}

	pricing, err := buildPricing(input, prices)
	if err != nil {
		var statusErr *statusError
		if errors.As(err, &statusErr) {
			return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, statusErr
		}
		return sqlcgen.GetRentalPropertyByUUIDRow{}, pricingDetails{}, nil, newStatusError(http.StatusInternalServerError, "could not calculate rental price")
	}

	return property, pricing, blockedDates, nil
}

func buildPricing(input RentalPreviewInput, rows []sqlcgen.ListRentalActivePricesRow) (pricingDetails, error) {
	priceMap := make(map[int32]sqlcgen.ListRentalActivePricesRow, len(rows))
	for _, row := range rows {
		priceMap[row.PeriodID] = row
	}

	requestedPrice, ok := priceMap[input.PeriodID]
	if !ok {
		return pricingDetails{}, newStatusError(http.StatusUnprocessableEntity, "no current rent price exists for the requested period")
	}

	requestedUnitPrice, err := numericToCents(requestedPrice.RentPrice)
	if err != nil {
		return pricingDetails{}, err
	}
	depositCents, err := numericToCents(requestedPrice.Deposit)
	if err != nil {
		return pricingDetails{}, err
	}

	details := pricingDetails{
		RequestedPeriodID:   input.PeriodID,
		RequestedPeriodName: normalizePeriodName(requestedPrice.PeriodName),
		UnitPriceCents:      requestedUnitPrice,
		Currency:            strings.TrimSpace(requestedPrice.Currency),
		DepositCents:        depositCents,
		IsNegotiable:        requestedPrice.IsNegotiable,
	}

	current := input.StartDate
	remainingEnd := input.EndDate

	switch input.PeriodID {
	case PeriodNightly:
		nights := daysBetween(current, remainingEnd)
		details.Units = int32(nights)
		if err := appendPricedComponent(&details, priceMap, PeriodNightly, int32(nights)); err != nil {
			return pricingDetails{}, err
		}
		current = current.AddDate(0, 0, nights)
	case PeriodWeekly:
		weeks, afterWeeks := consumeWeeks(current, remainingEnd)
		details.Units = int32(weeks)
		if err := appendPricedComponent(&details, priceMap, PeriodWeekly, int32(weeks)); err != nil {
			return pricingDetails{}, err
		}
		current = afterWeeks
	case PeriodMonthly:
		months, afterMonths := consumeMonths(current, remainingEnd)
		details.Units = int32(months)
		if err := appendPricedComponent(&details, priceMap, PeriodMonthly, int32(months)); err != nil {
			return pricingDetails{}, err
		}
		current = afterMonths
	case PeriodYearly:
		years, afterYears := consumeYears(current, remainingEnd)
		details.Units = int32(years)
		if err := appendPricedComponent(&details, priceMap, PeriodYearly, int32(years)); err != nil {
			return pricingDetails{}, err
		}
		current = afterYears
	default:
		return pricingDetails{}, newStatusError(http.StatusUnprocessableEntity, "period_id is not supported")
	}

	switch input.PeriodID {
	case PeriodYearly:
		current = applyYearlyRemainder(&details, priceMap, current, remainingEnd)
	case PeriodMonthly:
		current = applyMonthlyRemainder(&details, priceMap, current, remainingEnd)
	case PeriodWeekly:
		current = applyWeeklyRemainder(&details, priceMap, current, remainingEnd)
	}

	if current.Before(remainingEnd) {
		return pricingDetails{}, newStatusError(http.StatusUnprocessableEntity, "nightly fallback pricing is required for the remaining rental days")
	}

	details.SubtotalCents = sumComponents(details.PriceComponents)
	details.TotalCents = details.SubtotalCents + details.DepositCents
	return details, nil
}

func applyYearlyRemainder(details *pricingDetails, priceMap map[int32]sqlcgen.ListRentalActivePricesRow, current, end time.Time) time.Time {
	months, afterMonths := consumeMonths(current, end)
	if months > 0 {
		if err := appendPricedComponent(details, priceMap, PeriodMonthly, int32(months)); err == nil {
			current = afterMonths
		}
	}
	current = applyMonthlyRemainder(details, priceMap, current, end)
	return current
}

func applyMonthlyRemainder(details *pricingDetails, priceMap map[int32]sqlcgen.ListRentalActivePricesRow, current, end time.Time) time.Time {
	weeks, afterWeeks := consumeWeeks(current, end)
	if weeks > 0 {
		if err := appendPricedComponent(details, priceMap, PeriodWeekly, int32(weeks)); err == nil {
			current = afterWeeks
		}
	}
	return applyWeeklyRemainder(details, priceMap, current, end)
}

func applyWeeklyRemainder(details *pricingDetails, priceMap map[int32]sqlcgen.ListRentalActivePricesRow, current, end time.Time) time.Time {
	nights := daysBetween(current, end)
	if nights <= 0 {
		return end
	}
	if err := appendPricedComponent(details, priceMap, PeriodNightly, int32(nights)); err != nil {
		return current
	}
	return current.AddDate(0, 0, nights)
}

func appendPricedComponent(details *pricingDetails, priceMap map[int32]sqlcgen.ListRentalActivePricesRow, periodID int32, units int32) error {
	if units <= 0 {
		return nil
	}
	row, ok := priceMap[periodID]
	if !ok {
		return newStatusError(http.StatusUnprocessableEntity, "missing pricing for period %d", periodID)
	}
	if err := ensureCurrency(details.Currency, row.Currency); err != nil {
		return err
	}
	addPriceComponent(details, row, units)
	switch periodID {
	case PeriodNightly:
		details.Breakdown.Nights += units
	case PeriodWeekly:
		details.Breakdown.Weeks += units
	case PeriodMonthly:
		details.Breakdown.Months += units
	case PeriodYearly:
		details.Breakdown.Years += units
	}
	return nil
}

func addPriceComponent(details *pricingDetails, row sqlcgen.ListRentalActivePricesRow, units int32) {
	if units <= 0 {
		return
	}
	unitPrice, err := numericToCents(row.RentPrice)
	if err != nil {
		return
	}
	details.PriceComponents = append(details.PriceComponents, RentalPriceComponent{
		PeriodID:  row.PeriodID,
		Period:    normalizePeriodName(row.PeriodName),
		Units:     units,
		UnitPrice: centsToString(unitPrice),
		LineTotal: centsToString(unitPrice * int64(units)),
	})
}

func sumComponents(components []RentalPriceComponent) int64 {
	var total int64
	for _, component := range components {
		amount, _ := parseMoneyString(component.LineTotal)
		total += amount
	}
	return total
}

func createRentalTransactionParams(propertyID, clientID, agentID int32, endDate time.Time, total pgtype.Numeric) sqlcgen.CreateRentalTransactionParams {
	return sqlcgen.CreateRentalTransactionParams{
		PropertyID:  propertyID,
		ClientID:    clientID,
		AgentID:     agentID,
		StatusID:    TransactionStatusPending,
		FinalAmount: total,
		ClosingDate: pgtype.Date{Time: endDate, Valid: true},
	}
}

func consumeYears(start, end time.Time) (int, time.Time) {
	current := start
	count := 0
	for !current.AddDate(1, 0, 0).After(end) {
		current = current.AddDate(1, 0, 0)
		count++
	}
	return count, current
}

func consumeMonths(start, end time.Time) (int, time.Time) {
	current := start
	count := 0
	for !current.AddDate(0, 1, 0).After(end) {
		current = current.AddDate(0, 1, 0)
		count++
	}
	return count, current
}

func consumeWeeks(start, end time.Time) (int, time.Time) {
	days := daysBetween(start, end)
	weeks := days / 7
	return weeks, start.AddDate(0, 0, weeks*7)
}

func daysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours() / 24)
}

func ensureCurrency(expected, actual string) error {
	if strings.TrimSpace(expected) != strings.TrimSpace(actual) {
		return newStatusError(http.StatusUnprocessableEntity, "rent pricing currencies are inconsistent for the selected property")
	}
	return nil
}

func containsInt32(values []int32, target int32) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasPriceForPeriod(rows []sqlcgen.ListRentalActivePricesRow, periodID int32) bool {
	for _, row := range rows {
		if row.PeriodID == periodID {
			return true
		}
	}
	return false
}

func normalizePeriodName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "nightly":
		return "Nightly"
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	case "yearly":
		return "Yearly"
	default:
		return strings.TrimSpace(name)
	}
}

func numericToCents(value pgtype.Numeric) (int64, error) {
	floatValue, err := value.Float64Value()
	if err != nil {
		return 0, err
	}
	return int64(math.Round(floatValue.Float64 * 100)), nil
}

func numericFromCents(cents int64) pgtype.Numeric {
	var numeric pgtype.Numeric
	numeric.Scan(centsToString(cents))
	return numeric
}

func centsToString(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func centsToFloat(cents int64) float64 {
	return float64(cents) / 100
}

func parseMoneyString(value string) (int64, error) {
	var units int64
	var cents int64
	_, err := fmt.Sscanf(value, "%d.%02d", &units, &cents)
	if err != nil {
		return 0, err
	}
	return units*100 + cents, nil
}

func formatUUIDValue(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", value.Bytes[0:4], value.Bytes[4:6], value.Bytes[6:8], value.Bytes[8:10], value.Bytes[10:16])
}

type httpContractsClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPContractsClient(baseURL string) ContractsClient {
	return &httpContractsClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *httpContractsClient) CreateContract(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return ContractCreateResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/contracts/rent", bytes.NewReader(payload))
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
		return ContractCreateResult{}, fmt.Errorf("contracts endpoint returned status %d", resp.StatusCode)
	}

	var result ContractCreateResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ContractCreateResult{}, err
	}
	return result, nil
}
