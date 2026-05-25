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
)

func TestNewModule(t *testing.T) {
	m := NewModule(&pgxpool.Pool{}, "token", "secret")
	if m == nil {
		t.Errorf("expected not nil")
	}
	if m.Handler == nil {
		t.Errorf("expected not nil")
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	m.RegisterRoutes(r.Group("/test"), r.Group("/test2"))
}

func TestRepository_WithTx(t *testing.T) {
	repo := NewRepository(&pgxpool.Pool{})
	txRepo := repo.WithTx(&mockTx{})
	if txRepo == nil {
		t.Errorf("expected not nil")
	}
}

func TestRepository_Pointers(t *testing.T) {
	// Test int4FromPointer
	var nilInt *int32
	valInt := int32(5)
	if int4FromPointer(nilInt).Valid {
		t.Errorf("expected false")
	}
	if !(int4FromPointer(&valInt).Valid) {
		t.Errorf("expected true")
	}
	if int32(5) != int4FromPointer(&valInt).Int32 {
		t.Errorf("expected %v, got %v", int32(5), int4FromPointer(&valInt).Int32)
	}

	// Test dateFromPointer
	var nilDate *time.Time
	valDate := time.Now()
	if dateFromPointer(nilDate).Valid {
		t.Errorf("expected false")
	}
	if !(dateFromPointer(&valDate).Valid) {
		t.Errorf("expected true")
	}

	// Test textPointer
	if textPointer(pgtype.Text{Valid: false}) != nil {
		t.Errorf("expected nil, got %v", textPointer(pgtype.Text{Valid: false}))
	}
	if textPointer(pgtype.Text{String: "hello", Valid: true}) == nil {
		t.Errorf("expected not nil")
	}
	if "hello" != *textPointer(pgtype.Text{String: "hello", Valid: true}) {
		t.Errorf("expected %v, got %v", "hello", *textPointer(pgtype.Text{String: "hello", Valid: true}))
	}

	// Test timestamptzPointer
	if timestamptzPointer(pgtype.Timestamptz{Valid: false}) != nil {
		t.Errorf("expected nil, got %v", timestamptzPointer(pgtype.Timestamptz{Valid: false}))
	}
	if timestamptzPointer(pgtype.Timestamptz{Time: valDate, Valid: true}) == nil {
		t.Errorf("expected not nil")
	}

	// Test formatDate
	if "2024-01-01" != formatDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected %v, got %v", "2024-01-01", formatDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	}
}

func TestRepository_Methods(t *testing.T) {
	repo := NewRepository(&pgxpool.Pool{})
	ctx := context.Background()

	t.Run("CountCompletedPaymentsForContract", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.CountCompletedPaymentsForContract(ctx, 1)
		}()
	})
	t.Run("UpdateTransactionStatusByContract", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.UpdateTransactionStatusByContract(ctx, 1, 1)
		}()
	})
	t.Run("UpdatePropertyStatusByContract", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.UpdatePropertyStatusByContract(ctx, 1, 1)
		}()
	})
	t.Run("UpdateContractStatus", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.UpdateContractStatus(ctx, 1, 1)
		}()
	})
	t.Run("GetPaymentByContract", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPaymentByContract(ctx, 1, 1)
		}()
	})
	t.Run("CreatePayment", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.CreatePayment(ctx, sqlcgen.CreatePaymentParams{})
		}()
	})
	t.Run("GetContractForPayment", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetContractForPayment(ctx, 1)
		}()
	})
	t.Run("GetContractForPaymentWithLock", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetContractForPaymentWithLock(ctx, 1)
		}()
	})
	t.Run("GetPaymentByUUID", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPaymentByUUID(ctx, uuid.New())
		}()
	})
	t.Run("GetPaymentByGatewayID", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPaymentByGatewayID(ctx, "abc")
		}()
	})
	t.Run("GetLastPaidPeriod", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetLastPaidPeriod(ctx, 1)
		}()
	})
	t.Run("GetPendingPayments", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPendingPayments(ctx, 1)
		}()
	})
	t.Run("UpdatePaymentStatus", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{})
		}()
	})
	t.Run("ListPayments", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.ListPayments(ctx, 1, 1, ListPaymentsInput{})
		}()
	})
	t.Run("GetPaymentDetailByUUID", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPaymentDetailByUUID(ctx, uuid.New())
		}()
	})
	t.Run("Begin", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.Begin(ctx)
		}()
	})
}
