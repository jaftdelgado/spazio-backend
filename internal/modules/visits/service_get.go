package visits

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (s *service) ListUserVisits(ctx context.Context, userID int32, role int32, filter ListVisitsFilter) ([]VisitResponse, error) {
	params := sqlcgen.ListVisitsParams{}

	if filter.StatusID != nil {
		params.StatusID = pgtype.Int4{Int32: *filter.StatusID, Valid: true}
	}
	if filter.PropertyID != nil {
		params.PropertyID = pgtype.Int4{Int32: *filter.PropertyID, Valid: true}
	}
	if filter.Date != nil {
		params.VisitDate = pgtype.Date{Time: filter.Date.UTC(), Valid: true}
	}

	switch role {
	case 1:
	case 2:
		params.AgentID = pgtype.Int4{Int32: userID, Valid: true}
	case 3:
		params.ClientID = pgtype.Int4{Int32: userID, Valid: true}
	default:
		return nil, errors.New("rol de usuario no reconocido")
	}

	rows, err := s.repo.ListVisits(ctx, params)
	if err != nil {
		return nil, err
	}

	res := make([]VisitResponse, len(rows))
	for i, r := range rows {
		clientName := ""
		if r.ClientName != nil {
			clientName = r.ClientName.(string)
		}
		agentName := ""
		if r.AgentName != nil {
			agentName = r.AgentName.(string)
		}
		agentPhone := ""
		if r.AgentPhone.Valid {
			agentPhone = r.AgentPhone.String
		}
		cityName := ""
		if r.CityName.Valid {
			cityName = r.CityName.String
		}
		address := ""
		if r.Address != nil {
			address = r.Address.(string)
		}

		res[i] = VisitResponse{
			VisitUUID:     r.VisitUuid.Bytes,
			PropertyID:    r.PropertyID,
			PropertyTitle: r.PropertyTitle,
			AgentID:       r.AgentID.Int32,
			AgentName:     agentName,
			AgentPhone:    agentPhone,
			VisitDate:     r.VisitDate.Time,
			Status:        r.StatusName,
			CreatedAt:     r.CreatedAt.Time,
			ClientName:    clientName,
			ClientPhone:   r.ClientPhone,
			CityName:      cityName,
			Address:       address,
		}
	}
	return res, nil
}
