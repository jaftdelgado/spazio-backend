package payments

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
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

	assert.Len(t, items, 2)
	assert.Equal(t, int32(1), items[0].PaymentID)
	assert.Equal(t, paymentUUID, items[0].PaymentUUID)
	assert.Equal(t, "MP", *items[0].Gateway)
	assert.Nil(t, items[1].Gateway)
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

	assert.Equal(t, int32(1), detail.PaymentID)
	assert.Equal(t, "MP", *detail.Gateway)
	assert.NotNil(t, detail.PaymentDate)
}
