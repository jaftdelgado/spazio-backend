package payments

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestMapListPaymentsRows(t *testing.T) {
	now := time.Now()
	paymentUUID := uuid.New()
	rows := []sqlcgen.ListPaymentsRow{
		{
			PaymentID:     1,
			PaymentUuid:   pgtype.UUID{Bytes: paymentUUID, Valid: true},
			ContractID:    2,
			BillingPeriod: pgtype.Date{Time: now, Valid: true},
			Gateway:       pgtype.Text{String: "MP", Valid: true},
		},
		{
			PaymentID:     2,
			ContractID:    2,
			BillingPeriod: pgtype.Date{Time: now, Valid: true},
			Gateway:       pgtype.Text{Valid: false},
		},
	}

	items := mapListPaymentsRows(rows)

	if len(items) != 2 {
		t.Errorf("expected len %v, got %v", 2, len(items))
	}
	if int32(1) != items[0].PaymentID {
		t.Errorf("expected %v, got %v", int32(1), items[0].PaymentID)
	}
	if paymentUUID != items[0].PaymentUUID {
		t.Errorf("expected %v, got %v", paymentUUID, items[0].PaymentUUID)
	}
	if "MP" != *items[0].Gateway {
		t.Errorf("expected %v, got %v", "MP", *items[0].Gateway)
	}
	if items[1].Gateway != nil {
		t.Errorf("expected nil, got %v", items[1].Gateway)
	}
}

func TestMapPaymentDetailRow(t *testing.T) {
	now := time.Now()
	row := sqlcgen.GetPaymentDetailByUUIDRow{
		PaymentID:     1,
		ContractID:    2,
		BillingPeriod: pgtype.Date{Time: now, Valid: true},
		Gateway:       pgtype.Text{String: "MP", Valid: true},
		PaymentDate:   pgtype.Timestamptz{Time: now, Valid: true},
	}

	detail := mapPaymentDetailRow(row)

	if int32(1) != detail.PaymentID {
		t.Errorf("expected %v, got %v", int32(1), detail.PaymentID)
	}
	if "MP" != *detail.Gateway {
		t.Errorf("expected %v, got %v", "MP", *detail.Gateway)
	}
	if detail.PaymentDate == nil {
		t.Errorf("expected not nil")
	}
}
