package payments

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type serviceTestRepository struct {
	roleID             int32
	roleErr            error
	listItems          []PaymentListItem
	listErr            error
	detail             PaymentDetail
	detailErr          error
	getUserRoleCalled  bool
	listPaymentsCalled bool
	getPaymentCalled   bool
	lastListUserID     int32
	lastListRoleID     int32
	lastListInput      ListPaymentsInput
	lastPaymentID      int32
}

func (m *serviceTestRepository) ListPayments(_ context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error) {
	m.listPaymentsCalled = true
	m.lastListUserID = userID
	m.lastListRoleID = roleID
	m.lastListInput = input
	return m.listItems, m.listErr
}

func (m *serviceTestRepository) GetPaymentByID(_ context.Context, paymentID int32) (PaymentDetail, error) {
	m.getPaymentCalled = true
	m.lastPaymentID = paymentID
	return m.detail, m.detailErr
}

func (m *serviceTestRepository) GetUserRole(_ context.Context, _ int32) (int32, error) {
	m.getUserRoleCalled = true
	return m.roleID, m.roleErr
}

func TestService_ListPayments_GetUserRoleFails(t *testing.T) {
	repo := &serviceTestRepository{roleErr: errors.New("role lookup failed")}
	svc := NewService(repo)

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "list payments:") {
		t.Fatalf("error = %q, want wrapped list payments error", err.Error())
	}
	if repo.listPaymentsCalled {
		t.Fatal("did not expect ListPayments repository call")
	}
}

func TestService_ListPayments_UnsupportedRole(t *testing.T) {
	repo := &serviceTestRepository{roleID: 99}
	svc := NewService(repo)

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	if !errors.Is(err, ErrUnsupportedRole) {
		t.Fatalf("error = %v, want %v", err, ErrUnsupportedRole)
	}
	if repo.listPaymentsCalled {
		t.Fatal("did not expect ListPayments repository call")
	}
}

func TestService_ListPayments_RepositoryListFails(t *testing.T) {
	repo := &serviceTestRepository{roleID: roleAdminID, listErr: errors.New("db down")}
	svc := NewService(repo)

	_, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "list payments:") {
		t.Fatalf("error = %q, want wrapped list payments error", err.Error())
	}
	if !repo.listPaymentsCalled {
		t.Fatal("expected ListPayments repository call")
	}
}

func TestService_ListPayments_EmptyResult(t *testing.T) {
	repo := &serviceTestRepository{roleID: roleAdminID, listItems: []PaymentListItem{}}
	svc := NewService(repo)

	result, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListPayments() error = %v, want nil", err)
	}
	if len(result.Data) != 0 {
		t.Fatalf("len(data) = %d, want 0", len(result.Data))
	}
	if result.Pagination.Total != 0 {
		t.Fatalf("pagination.total = %d, want 0", result.Pagination.Total)
	}
}

func TestService_ListPayments_TotalFromFirstItem(t *testing.T) {
	repo := &serviceTestRepository{
		roleID: roleAdminID,
		listItems: []PaymentListItem{
			{PaymentID: 1, TotalCount: 84},
			{PaymentID: 2, TotalCount: 84},
		},
	}
	svc := NewService(repo)

	result, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListPayments() error = %v, want nil", err)
	}
	if result.Pagination.Total != 84 {
		t.Fatalf("pagination.total = %d, want 84", result.Pagination.Total)
	}
}

func TestService_ListPayments_PaginationPropagated(t *testing.T) {
	repo := &serviceTestRepository{
		roleID:    roleAdminID,
		listItems: []PaymentListItem{{PaymentID: 1, TotalCount: 1}},
	}
	svc := NewService(repo)

	result, err := svc.ListPayments(context.Background(), 7, ListPaymentsInput{Limit: 10, Offset: 20})
	if err != nil {
		t.Fatalf("ListPayments() error = %v, want nil", err)
	}
	if result.Pagination.Limit != 10 {
		t.Fatalf("pagination.limit = %d, want 10", result.Pagination.Limit)
	}
	if result.Pagination.Offset != 20 {
		t.Fatalf("pagination.offset = %d, want 20", result.Pagination.Offset)
	}
}

func TestService_ListPayments_AdminReceivesAllPayments(t *testing.T) {
	repo := &serviceTestRepository{
		roleID: roleAdminID,
		listItems: []PaymentListItem{
			{PaymentID: 1, TotalCount: 3},
			{PaymentID: 2, TotalCount: 3},
			{PaymentID: 3, TotalCount: 3},
		},
	}
	svc := NewService(repo)

	result, err := svc.ListPayments(context.Background(), 1, ListPaymentsInput{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("ListPayments() error = %v, want nil", err)
	}
	if len(result.Data) != 3 {
		t.Fatalf("len(data) = %d, want 3", len(result.Data))
	}
	if repo.lastListRoleID != roleAdminID {
		t.Fatalf("roleID = %d, want %d", repo.lastListRoleID, roleAdminID)
	}
}

func TestService_GetPaymentByID_PaymentNotFound(t *testing.T) {
	repo := &serviceTestRepository{detailErr: ErrPaymentNotFound}
	svc := NewService(repo)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrPaymentNotFound)
	}
	if repo.getUserRoleCalled {
		t.Fatal("did not expect GetUserRole after payment not found")
	}
}

func TestService_GetPaymentByID_UnexpectedRepositoryError(t *testing.T) {
	repo := &serviceTestRepository{detailErr: errors.New("timeout")}
	svc := NewService(repo)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("error = %v, did not expect ErrPaymentNotFound", err)
	}
	if !strings.Contains(err.Error(), "get payment by id:") {
		t.Fatalf("error = %q, want wrapped get payment by id error", err.Error())
	}
}

func TestService_GetPaymentByID_GetUserRoleFailsAfterPaymentFound(t *testing.T) {
	repo := &serviceTestRepository{
		detail:  PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 2},
		roleErr: errors.New("role lookup failed"),
	}
	svc := NewService(repo)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get payment by id:") {
		t.Fatalf("error = %q, want wrapped get payment by id error", err.Error())
	}
	if !repo.getUserRoleCalled {
		t.Fatal("expected GetUserRole to be called")
	}
}

func TestService_GetPaymentByID_AdminAccessesForeignPayment(t *testing.T) {
	repo := &serviceTestRepository{
		detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 99},
		roleID: roleAdminID,
	}
	svc := NewService(repo)

	result, err := svc.GetPaymentByID(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("GetPaymentByID() error = %v, want nil", err)
	}
	if result.PaymentID != 1 {
		t.Fatalf("payment_id = %d, want 1", result.PaymentID)
	}
}

func TestService_GetPaymentByID_AgentAccessesOwnPayment(t *testing.T) {
	repo := &serviceTestRepository{
		detail: PaymentDetail{PaymentID: 1, ClientID: 3, AgentID: 7},
		roleID: roleAgentID,
	}
	svc := NewService(repo)

	result, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("GetPaymentByID() error = %v, want nil", err)
	}
	if result.PaymentID != 1 {
		t.Fatalf("payment_id = %d, want 1", result.PaymentID)
	}
}

func TestService_GetPaymentByID_AgentAccessesForeignPayment(t *testing.T) {
	repo := &serviceTestRepository{
		detail: PaymentDetail{PaymentID: 1, ClientID: 3, AgentID: 99},
		roleID: roleAgentID,
	}
	svc := NewService(repo)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if !errors.Is(err, ErrPaymentForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrPaymentForbidden)
	}
}

func TestService_GetPaymentByID_ClientAccessesOwnPayment(t *testing.T) {
	repo := &serviceTestRepository{
		detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 2},
		roleID: roleClientID,
	}
	svc := NewService(repo)

	result, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("GetPaymentByID() error = %v, want nil", err)
	}
	if result.PaymentID != 1 {
		t.Fatalf("payment_id = %d, want 1", result.PaymentID)
	}
}

func TestService_GetPaymentByID_ClientAccessesForeignPayment(t *testing.T) {
	repo := &serviceTestRepository{
		detail: PaymentDetail{PaymentID: 1, ClientID: 99, AgentID: 2},
		roleID: roleClientID,
	}
	svc := NewService(repo)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if !errors.Is(err, ErrPaymentForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrPaymentForbidden)
	}
}

func TestService_GetPaymentByID_UnsupportedRole(t *testing.T) {
	repo := &serviceTestRepository{
		detail: PaymentDetail{PaymentID: 1, ClientID: 7, AgentID: 2},
		roleID: 99,
	}
	svc := NewService(repo)

	_, err := svc.GetPaymentByID(context.Background(), 7, 1)
	if !errors.Is(err, ErrUnsupportedRole) {
		t.Fatalf("error = %v, want %v", err, ErrUnsupportedRole)
	}
}

var _ PaymentsRepository = (*serviceTestRepository)(nil)
