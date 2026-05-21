package payments

import (
	"context"
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

func (s *service) GetPaymentByID(ctx context.Context, userID int32, roleID int32, paymentID int32) (PaymentDetail, error) {
	paymentRecord, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return PaymentDetail{}, err
	}

	switch roleID {
	case roleAdminID:
		// Admin can see everything
	case roleAgentID:
		if paymentRecord.AgentID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	case roleClientID:
		if paymentRecord.ClientID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	default:
		return PaymentDetail{}, ErrUnsupportedRole
	}

	return paymentRecord, nil
}
