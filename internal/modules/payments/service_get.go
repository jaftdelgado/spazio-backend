package payments

import (
	"context"

	"github.com/google/uuid"
)

func (s *service) ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	if !isSupportedRole(roleID) {
		return ListPaymentsResult{}, ErrUnsupportedRole
	}

	payments, err := s.repo.ListPayments(ctx, userID, roleID, input)
	if err != nil {
		return ListPaymentsResult{}, err
	}

	total := int64(0)
	if len(payments) > 0 {
		total = payments[0].TotalCount
	}

	return ListPaymentsResult{
		Data: payments,
		Pagination: PaymentsPagination{
			Limit:  input.Limit,
			Offset: input.Offset,
			Total:  total,
		},
	}, nil
}

func (s *service) GetPaymentByUUID(ctx context.Context, userID int32, roleID int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error) {
	paymentRecord, err := s.repo.GetPaymentDetailByUUID(ctx, paymentUUID)
	if err != nil {
		return PaymentDetailResponse{}, err
	}

	switch roleID {
	case roleAdminID:
		// Admin can see everything
	case roleAgentID:
		if paymentRecord.AgentID != userID {
			return PaymentDetailResponse{}, ErrPaymentForbidden
		}
	case roleClientID:
		if paymentRecord.ClientID != userID {
			return PaymentDetailResponse{}, ErrPaymentForbidden
		}
	default:
		return PaymentDetailResponse{}, ErrUnsupportedRole
	}

	return newPaymentDetailResponse(paymentRecord, roleID), nil
}

func newPaymentDetailResponse(record PaymentDetailRecord, roleID int32) PaymentDetailResponse {
	response := PaymentDetailResponse{
		PaymentID:       record.PaymentID,
		ContractID:      record.ContractID,
		PropertyID:      record.PropertyID,
		TransactionID:   record.TransactionID,
		TransactionType: record.TransactionType,
		BillingPeriod:   record.BillingPeriod,
		DueDate:         record.DueDate,
		AgreedAmount:    record.AgreedAmount,
		Amount:          record.Amount,
		Currency:        record.Currency,
		PaymentMethod:   record.PaymentMethod,
		Gateway:         record.Gateway,
		Status:          record.Status,
		PaymentDate:     record.PaymentDate,
	}

	if roleID == roleAdminID {
		response.ClientID = &record.ClientID
		response.AgentID = &record.AgentID
	}

	return response
}
