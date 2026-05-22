package payments

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

func TestNewModule(t *testing.T) {
	m := NewModule(&pgxpool.Pool{}, "token", "secret")
	assert.NotNil(t, m)
	assert.NotNil(t, m.Handler)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	m.RegisterRoutes(r.Group("/test"), r.Group("/test2"))
}

func TestRepository_WithTx(t *testing.T) {
	repo := NewRepository(&pgxpool.Pool{})
	txRepo := repo.WithTx(&mockTx{})
	assert.NotNil(t, txRepo)
}

func TestRepository_Pointers(t *testing.T) {
	// Test int4FromPointer
	var nilInt *int32
	valInt := int32(5)
	assert.False(t, int4FromPointer(nilInt).Valid)
	assert.True(t, int4FromPointer(&valInt).Valid)
	assert.Equal(t, int32(5), int4FromPointer(&valInt).Int32)

	// Test dateFromPointer
	var nilDate *time.Time
	valDate := time.Now()
	assert.False(t, dateFromPointer(nilDate).Valid)
	assert.True(t, dateFromPointer(&valDate).Valid)

	// Test textPointer
	assert.Nil(t, textPointer(pgtype.Text{Valid: false}))
	assert.NotNil(t, textPointer(pgtype.Text{String: "hello", Valid: true}))
	assert.Equal(t, "hello", *textPointer(pgtype.Text{String: "hello", Valid: true}))

	// Test timestamptzPointer
	assert.Nil(t, timestamptzPointer(pgtype.Timestamptz{Valid: false}))
	assert.NotNil(t, timestamptzPointer(pgtype.Timestamptz{Time: valDate, Valid: true}))

	// Test formatDate
	assert.Equal(t, "2024-01-01", formatDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
}

func TestRepository_Methods(t *testing.T) {
	repo := NewRepository(&pgxpool.Pool{})
	ctx := context.Background()

	t.Run("CountCompletedPaymentsForContract", func(t *testing.T) {
		assert.Panics(t, func() { repo.CountCompletedPaymentsForContract(ctx, 1) })
	})
	t.Run("UpdateTransactionStatusByContract", func(t *testing.T) {
		assert.Panics(t, func() { repo.UpdateTransactionStatusByContract(ctx, 1, 1) })
	})
	t.Run("UpdatePropertyStatusByContract", func(t *testing.T) {
		assert.Panics(t, func() { repo.UpdatePropertyStatusByContract(ctx, 1, 1) })
	})
	t.Run("UpdateContractStatus", func(t *testing.T) {
		assert.Panics(t, func() { repo.UpdateContractStatus(ctx, 1, 1) })
	})
	t.Run("GetPaymentByContract", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPaymentByContract(ctx, 1, 1) })
	})
	t.Run("CreatePayment", func(t *testing.T) {
		assert.Panics(t, func() { repo.CreatePayment(ctx, sqlcgen.CreatePaymentParams{}) })
	})
	t.Run("GetContractForPayment", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetContractForPayment(ctx, 1) })
	})
	t.Run("GetContractForPaymentWithLock", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetContractForPaymentWithLock(ctx, 1) })
	})
	t.Run("GetPaymentByUUID", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPaymentByUUID(ctx, uuid.New()) })
	})
	t.Run("GetPaymentByGatewayID", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPaymentByGatewayID(ctx, "abc") })
	})
	t.Run("GetLastPaidPeriod", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetLastPaidPeriod(ctx, 1) })
	})
	t.Run("GetPendingPayments", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPendingPayments(ctx, 1) })
	})
	t.Run("UpdatePaymentStatus", func(t *testing.T) {
		assert.Panics(t, func() { repo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{}) })
	})
	t.Run("ListPayments", func(t *testing.T) {
		assert.Panics(t, func() { repo.ListPayments(ctx, 1, 1, ListPaymentsInput{}) })
	})
	t.Run("GetPaymentDetailByUUID", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPaymentDetailByUUID(ctx, uuid.New()) })
	})
	t.Run("Begin", func(t *testing.T) {
		assert.Panics(t, func() { repo.Begin(ctx) })
	})
}
