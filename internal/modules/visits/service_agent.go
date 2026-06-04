package visits

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (s *service) ConfirmVisit(ctx context.Context, userID int32, role int32, visitUUID uuid.UUID) error {
	visit, err := s.repo.GetVisitByUUID(ctx, visitUUID)
	if err != nil {
		return errors.New("visita no encontrada")
	}

	currentStatus := visit.StatusID
	newStatus := currentStatus

	if role == 3 {
		if visit.ClientID != userID {
			return errors.New("no tienes permiso para confirmar esta visita")
		}
		switch currentStatus {
		case StatusPending:
			newStatus = StatusWaitingAgent
		case StatusWaitingClient:
			newStatus = StatusConfirmed
		}
	} else if role == 2 { // Agente
		if !visit.AgentID.Valid || visit.AgentID.Int32 != userID {
			return errors.New("no eres el agente asignado a esta visita")
		}
		switch currentStatus {
		case StatusPending:
			newStatus = StatusWaitingClient
		case StatusWaitingAgent:
			newStatus = StatusConfirmed
		}
	} else if role == 1 {
		newStatus = StatusConfirmed
	}

	if newStatus == currentStatus {
		return errors.New("operación no válida o ya confirmada")
	}

	if err := s.repo.UpdateVisitStatus(ctx, visit.VisitID, newStatus); err != nil {
		return translateError(err)
	}
	return s.repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{
		VisitID: visit.VisitID, PreviousStatusID: currentStatus, NewStatusID: newStatus, ChangedByUserID: userID,
	})
}

func (s *service) CompleteVisit(ctx context.Context, userID int32, role int32, visitUUID uuid.UUID) error {
	visit, err := s.repo.GetVisitByUUID(ctx, visitUUID)
	if err != nil {
		return errors.New("visita no encontrada")
	}

	if role != 1 && role != 2 {
		return errors.New("solo el agente o administrador pueden marcar la visita como completada")
	}

	if role == 2 && (!visit.AgentID.Valid || visit.AgentID.Int32 != userID) {
		return errors.New("no eres el agente asignado a esta visita")
	}

	if visit.StatusID != StatusConfirmed {
		return errors.New("solo se pueden completar visitas que estén confirmadas")
	}

	if err := s.repo.UpdateVisitStatus(ctx, visit.VisitID, StatusCompleted); err != nil {
		return translateError(err)
	}

	return s.repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{
		VisitID:          visit.VisitID,
		PreviousStatusID: visit.StatusID,
		NewStatusID:      StatusCompleted,
		ChangedByUserID:  userID,
	})
}

func (s *service) CancelVisit(ctx context.Context, userID int32, role int32, visitUUID uuid.UUID) error {
	visit, err := s.repo.GetVisitByUUID(ctx, visitUUID)
	if err != nil {
		return errors.New("visita no encontrada")
	}

	if role == 3 {
		if visit.ClientID != userID {
			return errors.New("no tienes permiso para cancelar esta visita")
		}
	} else if role == 2 {
		if !visit.AgentID.Valid || visit.AgentID.Int32 != userID {
			return errors.New("no eres el agente asignado a esta visita")
		}
	} else {
		return errors.New("rol no autorizado para cancelar visitas")
	}

	if visit.StatusID != StatusPending && visit.StatusID != StatusWaitingAgent && visit.StatusID != StatusWaitingClient {
		return errors.New("solo se pueden cancelar visitas que no han sido confirmadas totalmente")
	}

	if err := s.repo.UpdateVisitStatus(ctx, visit.VisitID, StatusCancelled); err != nil {
		return translateError(err)
	}

	return s.repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{
		VisitID:          visit.VisitID,
		PreviousStatusID: visit.StatusID,
		NewStatusID:      StatusCancelled,
		ChangedByUserID:  userID,
	})
}
