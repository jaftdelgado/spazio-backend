package payments

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
	mpConfig "github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

// MPClient defines the interface for communicating with MercadoPago API.
type MPClient interface {
	CreatePayment(ctx context.Context, req payment.Request) (*payment.Response, error)
	GetPayment(ctx context.Context, id int) (*payment.Response, error)
}

// realMPClient is the production implementation of MPClient using the MercadoPago SDK.
type realMPClient struct {
	client payment.Client
}

func (c *realMPClient) CreatePayment(ctx context.Context, req payment.Request) (*payment.Response, error) {
	return c.client.Create(ctx, req)
}

func (c *realMPClient) GetPayment(ctx context.Context, id int) (*payment.Response, error) {
	return c.client.Get(ctx, id)
}

type service struct {
	repo            Repository
	mpAccessToken   string
	mpWebhookSecret string
	mpClient        MPClient
}

func NewService(repo Repository, mpAccessToken string, mpWebhookSecret string) Service {
	var mpClient MPClient
	cfg, err := mpConfig.New(mpAccessToken)
	if err == nil {
		mpClient = &realMPClient{client: payment.NewClient(cfg)}
	}
	return &service{
		repo:            repo,
		mpAccessToken:   mpAccessToken,
		mpWebhookSecret: mpWebhookSecret,
		mpClient:        mpClient,
	}
}

// NewTestService constructs a service instance with a custom MPClient (for mocking).
func NewTestService(repo Repository, mpClient MPClient, mpAccessToken string, mpWebhookSecret string) Service {
	return &service{
		repo:            repo,
		mpAccessToken:   mpAccessToken,
		mpWebhookSecret: mpWebhookSecret,
		mpClient:        mpClient,
	}
}

func translateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23503" {
			return errors.New("el contrato o método de pago seleccionado no existe")
		}
	}
	return err
}

func isSupportedRole(roleID int32) bool {
	return roleID == shared.RoleAdminID || roleID == shared.RoleAgentID || roleID == shared.RoleClientID
}
