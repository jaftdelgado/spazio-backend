package visits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (s *service) ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	return s.scheduleVisitInternal(ctx, s.repo, clientID, propertyID, visitDate)
}

func (s *service) scheduleVisitInternal(ctx context.Context, repo Repository, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	visitDate = normalizeDate(visitDate)
	now := time.Now().UTC()

	if visitDate.Before(now.Add(48 * time.Hour)) {
		return VisitResponse{}, errors.New("debe agendar con al menos 48 horas de anticipación")
	}
	if visitDate.After(now.Add(30 * 24 * time.Hour)) {
		return VisitResponse{}, errors.New("no puede agendar con más de 30 días de anticipación")
	}

	if err := s.validateEntityIntegrity(ctx, repo, clientID, propertyID); err != nil {
		return VisitResponse{}, err
	}

	agentID, err := repo.GetPrimaryAgentForProperty(ctx, propertyID)
	if err != nil {
		return VisitResponse{}, errors.New("la propiedad no tiene un agente asignado disponible")
	}

	availableSlots, err := s.GetAvailableSlots(ctx, propertyID, visitDate)
	if err != nil {
		return VisitResponse{}, err
	}

	isValidSlot := false
	for _, slot := range availableSlots {
		if slot.StartTime.Equal(visitDate) && slot.Available {
			isValidSlot = true
			break
		}
	}

	if !isValidSlot {
		return VisitResponse{}, errors.New("el horario seleccionado ya no está disponible")
	}

	visit, err := repo.CreateVisit(ctx, sqlcgen.CreateVisitParams{
		PropertyID: propertyID,
		ClientID:   clientID,
		AgentID:    pgtype.Int4{Int32: agentID, Valid: true},
		VisitDate:  pgtype.Timestamptz{Time: visitDate, Valid: true},
		StatusID:   StatusPending,
	})

	if err != nil {
		return VisitResponse{}, translateError(err)
	}

	return VisitResponse{
		VisitUUID:  visit.VisitUuid.Bytes,
		PropertyID: visit.PropertyID,
		AgentID:    visit.AgentID.Int32,
		VisitDate:  visit.VisitDate.Time,
		Status:     "Pending",
		CreatedAt:  visit.CreatedAt.Time,
		ClientName: "",
	}, nil
}

func (s *service) RescheduleVisit(ctx context.Context, userID int32, role int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	oldVisit, err := txRepo.GetVisitByUUID(ctx, visitUUID)
	if err != nil {
		return VisitResponse{}, errors.New("visita no encontrada")
	}

	if oldVisit.StatusID == StatusCancelled {
		return VisitResponse{}, errors.New("no se puede reagendar una cita ya cancelada")
	}

	canReschedule := (role == 1) || (role == 3 && oldVisit.ClientID == userID) || (role == 2 && oldVisit.AgentID.Valid && oldVisit.AgentID.Int32 == userID)
	if !canReschedule {
		return VisitResponse{}, errors.New("no tienes permiso para reagendar esta visita")
	}

	newVisit, err := s.scheduleVisitInternal(ctx, txRepo, oldVisit.ClientID, oldVisit.PropertyID, newDate)
	if err != nil {
		return VisitResponse{}, err
	}

	if err := txRepo.UpdateVisitStatus(ctx, oldVisit.VisitID, StatusCancelled); err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al cancelar cita anterior: %w", err)
	}

	if err := txRepo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{
		VisitID:          oldVisit.VisitID,
		PreviousStatusID: oldVisit.StatusID,
		NewStatusID:      StatusCancelled,
		ChangedByUserID:  userID,
	}); err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al registrar historial: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al confirmar cambios: %w", err)
	}

	return newVisit, nil
}
