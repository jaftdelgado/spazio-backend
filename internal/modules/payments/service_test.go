package payments

import (
	"context"
	"errors"
	"testing"
)

type serviceMockRepository struct {
	roleID         int32
	roleErr        error
	listItems      []PaymentListItem
	listErr        error
	detail         PaymentDetail
	detailErr      error
	lastListUserID int32
	lastListRoleID int32
	lastListInput  ListPaymentsInput
	lastDetailID   int32
}

func (m *serviceMockRepository) ListPayments(_ context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error) {
	m.lastListUserID = userID
	m.lastListRoleID = roleID
	m.lastListInput = input
	return m.listItems, m.listErr
}

func (m *serviceMockRepository) GetPaymentByID(_ context.Context, paymentID int32) (PaymentDetail, error) {
	m.lastDetailID = paymentID
	return m.detail, m.detailErr
}

func (m *serviceMockRepository) GetUserRole(_ context.Context, userID int32) (int32, error) {
	_ = userID
	return m.roleID, m.roleErr
}

func TestListPaymentsService(t *testing.T) {
	tests := []struct {
		name         string
		userID       int32
		input        ListPaymentsInput
		repo         *serviceMockRepository
		wantErr      error
		wantTotal    int64
		wantListRole int32
	}{
		{
			name:         "lists payments as admin",
			userID:       1,
			input:        ListPaymentsInput{Limit: 20, Offset: 0},
			repo:         &serviceMockRepository{roleID: roleAdminID, listItems: []PaymentListItem{{PaymentID: 1, TotalCount: 5}}},
			wantTotal:    5,
			wantListRole: roleAdminID,
		},
		{
			name:         "returns zero total on empty page",
			userID:       2,
			input:        ListPaymentsInput{Limit: 20, Offset: 40},
			repo:         &serviceMockRepository{roleID: roleAgentID, listItems: []PaymentListItem{}},
			wantTotal:    0,
			wantListRole: roleAgentID,
		},
		{
			name:         "lists payments as client",
			userID:       3,
			input:        ListPaymentsInput{Limit: 20, Offset: 0},
			repo:         &serviceMockRepository{roleID: roleClientID, listItems: []PaymentListItem{{PaymentID: 2, TotalCount: 1}}},
			wantTotal:    1,
			wantListRole: roleClientID,
		},
		{
			name:    "rejects unsupported role",
			userID:  9,
			input:   ListPaymentsInput{Limit: 20, Offset: 0},
			repo:    &serviceMockRepository{roleID: 99},
			wantErr: ErrUnsupportedRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

			result, err := svc.ListPayments(context.Background(), tt.userID, tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ListPayments() error = %v, want nil", err)
			}

			if result.Pagination.Total != tt.wantTotal {
				t.Fatalf("total = %d, want %d", result.Pagination.Total, tt.wantTotal)
			}

			if tt.repo.lastListRoleID != tt.wantListRole {
				t.Fatalf("list role = %d, want %d", tt.repo.lastListRoleID, tt.wantListRole)
			}
		})
	}
}

func TestGetPaymentByIDService(t *testing.T) {
	tests := []struct {
		name    string
		userID  int32
		repo    *serviceMockRepository
		wantErr error
	}{
		{
			name:   "allows admin to read any payment",
			userID: 1,
			repo: &serviceMockRepository{
				roleID: roleAdminID,
				detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 2},
			},
		},
		{
			name:   "allows agent to read own payment",
			userID: 2,
			repo: &serviceMockRepository{
				roleID: roleAgentID,
				detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 2},
			},
		},
		{
			name:   "allows client to read own payment",
			userID: 7,
			repo: &serviceMockRepository{
				roleID: roleClientID,
				detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 2},
			},
		},
		{
			name:   "rejects foreign payment for agent",
			userID: 2,
			repo: &serviceMockRepository{
				roleID: roleAgentID,
				detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 9},
			},
			wantErr: ErrPaymentForbidden,
		},
		{
			name:   "returns not found when payment does not exist",
			userID: 1,
			repo: &serviceMockRepository{
				roleID:    roleAdminID,
				detailErr: ErrPaymentNotFound,
			},
			wantErr: ErrPaymentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)

			_, err := svc.GetPaymentByID(context.Background(), tt.userID, 1)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetPaymentByID() error = %v, want nil", err)
			}
		})
	}
}

var _ PaymentsRepository = (*serviceMockRepository)(nil)
