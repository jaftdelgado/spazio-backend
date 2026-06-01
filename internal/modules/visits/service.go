package visits

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func translateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			if strings.Contains(pgErr.ConstraintName, "property_id") {
				return errors.New("la propiedad seleccionada no existe")
			}
			if strings.Contains(pgErr.ConstraintName, "client_id") || strings.Contains(pgErr.ConstraintName, "agent_id") {
				return errors.New("el usuario involucrado no existe")
			}
			return errors.New("recurso relacionado no encontrado")
		case "23505":
			return errors.New("ya existe una visita programada para ese horario")
		}
	}
	return err
}

func normalizeDate(t time.Time) time.Time {
	loc, _ := time.LoadLocation("America/Mexico_City")
	tLocal := t.In(loc)
	return time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), tLocal.Hour(), 0, 0, 0, loc)
}

func (s *service) validateEntityIntegrity(ctx context.Context, repo Repository, clientID, propertyID int32) error {
	if _, err := repo.CheckUserActive(ctx, clientID); err != nil {
		return errors.New("el cliente no está activo o no existe")
	}

	prop, err := repo.GetPropertyStatusAndCheckDeleted(ctx, propertyID)
	if err != nil {
		return errors.New("la propiedad no existe")
	}
	if prop.DeletedAt.Valid {
		return errors.New("la propiedad ya no está disponible (eliminada)")
	}
	if prop.StatusID != PropertyStatusAvailable {
		return errors.New("la propiedad no está disponible para recibir visitas en este momento")
	}

	return nil
}

func pgTimeToHM(pt pgtype.Time) (int, int) {
	seconds := pt.Microseconds / 1e6
	hour := int(seconds / 3600)
	minute := int((seconds % 3600) / 60)
	return hour, minute
}
