package payments

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_ListPayments(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		roleID    int32
		setupRepo func() *mockPaymentRepository
		wantErr   bool
	}{
		{
			name:   "success listing payments for admin",
			roleID: roleAdminID,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					listPaymentsFunc: func(ctx context.Context, uid, rid int32, in ListPaymentsInput) ([]PaymentListItem, error) {
						return []PaymentListItem{{PaymentID: 1, TotalCount: 1}}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "error when role is not supported",
			roleID: 99,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo, "TOKEN", "SECRET")

			res, err := svc.ListPayments(ctx, 1, tt.roleID, ListPaymentsInput{})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res.Data)
			}
		})
	}
}

func TestService_GetPaymentByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    int32
		roleID    int32
		setupRepo func() *mockPaymentRepository
		wantErr   bool
	}{
		{
			name:   "success for admin to see any payment",
			userID: 1,
			roleID: roleAdminID,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					getPaymentByIDFunc: func(ctx context.Context, id int32) (PaymentDetail, error) {
						return PaymentDetail{PaymentID: 1, ClientID: 10}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "forbidden for client to see someone else's payment",
			userID: 5,
			roleID: roleClientID,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					getPaymentByIDFunc: func(ctx context.Context, id int32) (PaymentDetail, error) {
						return PaymentDetail{PaymentID: 1, ClientID: 10}, nil
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo, "TOKEN", "SECRET")

			_, err := svc.GetPaymentByID(ctx, tt.userID, tt.roleID, 1)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
