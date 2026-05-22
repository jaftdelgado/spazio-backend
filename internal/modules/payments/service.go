package payments

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type service struct {
	repo            Repository
	mpAccessToken   string
	mpWebhookSecret string
}

func NewService(repo Repository, mpAccessToken string, mpWebhookSecret string) Service {
	return &service{
		repo:            repo,
		mpAccessToken:   mpAccessToken,
		mpWebhookSecret: mpWebhookSecret,
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
