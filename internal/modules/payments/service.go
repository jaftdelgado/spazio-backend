package payments

import (
	"context"
	"errors"
	"fmt"
)

const (
	roleAdminID  int32 = 1
	roleAgentID  int32 = 2
	roleClientID int32 = 3
)

type service struct {
	repository PaymentsRepository
}

// NewService builds a payments service implementation.
func NewService(repository PaymentsRepository) PaymentsService {
	return &service{repository: repository}
}

func (s *service) ListPayments(ctx context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	roleID, err := s.repository.GetUserRole(ctx, userID)
	if err != nil {
		return ListPaymentsResult{}, fmt.Errorf("list payments: %w", err)
	}

	if !isSupportedRole(roleID) {
		return ListPaymentsResult{}, ErrUnsupportedRole
	}

	items, err := s.repository.ListPayments(ctx, userID, roleID, input)
	if err != nil {
		return ListPaymentsResult{}, fmt.Errorf("list payments: %w", err)
	}

	var total int64
	if len(items) > 0 {
		total = items[0].TotalCount
	}

	return ListPaymentsResult{
		Data: items,
		Pagination: PaymentsPagination{
			Limit:  input.Limit,
			Offset: input.Offset,
			Total:  total,
		},
	}, nil
}

func (s *service) GetPaymentByID(ctx context.Context, userID int32, paymentID int32) (PaymentDetail, error) {
	payment, err := s.repository.GetPaymentByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return PaymentDetail{}, ErrPaymentNotFound
		}
		return PaymentDetail{}, fmt.Errorf("get payment by id: %w", err)
	}

	roleID, err := s.repository.GetUserRole(ctx, userID)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("get payment by id: %w", err)
	}

	switch roleID {
	case roleAdminID:
		return payment, nil
	case roleAgentID:
		if payment.AgentID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	case roleClientID:
		if payment.ClientID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	default:
		return PaymentDetail{}, ErrUnsupportedRole
	}

	return payment, nil
}

func isSupportedRole(roleID int32) bool {
	return roleID == roleAdminID || roleID == roleAgentID || roleID == roleClientID
}
