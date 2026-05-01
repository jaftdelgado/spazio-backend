package visits

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

const (
	StatusPending       = 1
	StatusWaitingAgent   = 2
	StatusWaitingClient  = 3
	StatusConfirmed     = 4
	StatusCancelled     = 5
	StatusCompleted     = 6

	PropertyStatusAvailable = 2
)

type ListVisitsFilter struct {
	Date       *time.Time
	StatusID   *int32
	PropertyID *int32
}

type Service interface {
	GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error)
	ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error)
	ListUserVisits(ctx context.Context, userID int32, filter ListVisitsFilter) ([]VisitResponse, error)
	ConfirmVisit(ctx context.Context, userID int32, visitUUID uuid.UUID) error
	RescheduleVisit(ctx context.Context, userID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error)
	CompleteVisit(ctx context.Context, userID int32, visitUUID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// translateError converts database errors to user-friendly messages
func translateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			if strings.Contains(pgErr.ConstraintName, "property_id") {
				return errors.New("la propiedad seleccionada no existe")
			}
			if strings.Contains(pgErr.ConstraintName, "client_id") || strings.Contains(pgErr.ConstraintName, "agent_id") {
				return errors.New("el usuario involucrado no existe")
			}
		case "23505": // unique_violation
			return errors.New("ya existe una visita programada para ese horario")
		}
	}
	return err
}

func normalizeDate(t time.Time) time.Time {
	// Normalize to UTC and set minutes/seconds to 0
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
}

func (s *service) validateEntityIntegrity(ctx context.Context, repo Repository, clientID, propertyID int32) error {
	// 1. Validar que el cliente esté activo
	if _, err := repo.CheckUserActive(ctx, clientID); err != nil {
		return errors.New("el cliente no está activo o no existe")
	}

	// 2. Validar estado de la propiedad y si está borrada
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

func (s *service) GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error) {
	// Normalizar fecha de entrada (solo el día importa para la consulta)
	date = date.UTC()

	agentID, err := s.repo.GetPrimaryAgentForProperty(ctx, propertyID)
	if err != nil {
		return nil, errors.New("la propiedad no tiene un agente asignado")
	}

	dayOfWeek := int16(date.Weekday())
	allSchedules, err := s.repo.GetAgentSchedule(ctx, agentID)
	if err != nil {
		return nil, err
	}

	var daySchedule *sqlcgen.GetAgentScheduleRow
	for _, sch := range allSchedules {
		if sch.DayOfWeek == dayOfWeek {
			daySchedule = &sch
			break
		}
	}

	if daySchedule == nil {
		return []TimeSlot{}, nil
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	exceptions, err := s.repo.GetPropertyExceptions(ctx, propertyID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	occupied, err := s.repo.GetOccupiedVisits(ctx, agentID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	slots := []TimeSlot{}
	hStart, mStart := pgTimeToHM(daySchedule.StartTime)
	hEnd, mEnd := pgTimeToHM(daySchedule.EndTime)

	workStart := time.Date(date.Year(), date.Month(), date.Day(), hStart, mStart, 0, 0, time.UTC)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(), hEnd, mEnd, 0, 0, time.UTC)

	for current := workStart; current.Add(time.Hour).Before(workEnd) || current.Add(time.Hour).Equal(workEnd); current = current.Add(time.Hour) {
		slotEnd := current.Add(time.Hour)
		isAvailable := true

		for _, ex := range exceptions {
			if ex.StartTime.Valid && ex.EndTime.Valid {
				ehStart, emStart := pgTimeToHM(ex.StartTime)
				ehEnd, emEnd := pgTimeToHM(ex.EndTime)
				exStart := time.Date(date.Year(), date.Month(), date.Day(), ehStart, emStart, 0, 0, time.UTC)
				exEnd := time.Date(date.Year(), date.Month(), date.Day(), ehEnd, emEnd, 0, 0, time.UTC)
				if current.Before(exEnd) && exStart.Before(slotEnd) {
					isAvailable = false
					break
				}
			} else {
				isAvailable = false
				break
			}
		}

		if !isAvailable { continue }

		for _, occ := range occupied {
			occUTC := occ.UTC()
			occEnd := occUTC.Add(time.Hour)
			if current.Before(occEnd) && occUTC.Before(slotEnd) {
				isAvailable = false
				break
			}
		}

		slots = append(slots, TimeSlot{
			StartTime: current,
			EndTime:   slotEnd,
			Available: isAvailable,
		})
	}
	return slots, nil
}

func (s *service) ScheduleVisit(ctx context.Context, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	return s.scheduleVisitInternal(ctx, s.repo, clientID, propertyID, visitDate)
}

func (s *service) scheduleVisitInternal(ctx context.Context, repo Repository, clientID int32, propertyID int32, visitDate time.Time) (VisitResponse, error) {
	visitDate = normalizeDate(visitDate)
	now := time.Now().UTC()

	// Validar 48h de anticipación (ahora en UTC)
	if visitDate.Before(now.Add(48 * time.Hour)) {
		return VisitResponse{}, errors.New("debe agendar con al menos 48 horas de anticipación")
	}
	if visitDate.After(now.Add(30 * 24 * time.Hour)) {
		return VisitResponse{}, errors.New("no puede agendar con más de 30 días de anticipación")
	}

	// Blindaje: Validar estado de propiedad y borrado lógico
	if err := s.validateEntityIntegrity(ctx, repo, clientID, propertyID); err != nil {
		return VisitResponse{}, err
	}

	agentID, err := repo.GetPrimaryAgentForProperty(ctx, propertyID)
	if err != nil {
		return VisitResponse{}, errors.New("la propiedad no tiene un agente asignado disponible")
	}

	// Validar disponibilidad real
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
		VisitUUID:     visit.VisitUuid.Bytes,
		PropertyID:    visit.PropertyID,
		AgentID:       visit.AgentID.Int32,
		VisitDate:     visit.VisitDate.Time,
		Status:        "Pending",
		CreatedAt:     visit.CreatedAt.Time,
		ClientName:    "", // Se llena en el listado
	}, nil
}

func (s *service) ListUserVisits(ctx context.Context, userID int32, filter ListVisitsFilter) ([]VisitResponse, error) {
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil { return nil, err }

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
	case 1: // Admin
	case 2: params.AgentID = pgtype.Int4{Int32: userID, Valid: true}
	case 3: params.ClientID = pgtype.Int4{Int32: userID, Valid: true}
	default: return nil, errors.New("rol de usuario no reconocido")
	}

	rows, err := s.repo.ListVisits(ctx, params)
	if err != nil { return nil, err }

	res := make([]VisitResponse, len(rows))
	for i, r := range rows {
		clientName := ""
		if r.ClientName != nil { clientName = r.ClientName.(string) }
		agentName := ""
		if r.AgentName != nil { agentName = r.AgentName.(string) }
		agentPhone := ""
		if r.AgentPhone.Valid { agentPhone = r.AgentPhone.String }
		cityName := ""
		if r.CityName.Valid { cityName = r.CityName.String }
		address := ""
		if r.Address != nil { address = r.Address.(string) }

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

func (s *service) ConfirmVisit(ctx context.Context, userID int32, visitUUID uuid.UUID) error {
	visit, err := s.repo.GetVisitByUUID(ctx, visitUUID)
	if err != nil { return errors.New("visita no encontrada") }

	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil { return err }

	currentStatus := visit.StatusID
	newStatus := currentStatus

	if role == 3 { // Cliente
		if visit.ClientID != userID { return errors.New("no tienes permiso para confirmar esta visita") }
		switch currentStatus {
		case StatusPending: newStatus = StatusWaitingAgent
		case StatusWaitingClient: newStatus = StatusConfirmed
		}
	} else if role == 2 { // Agente
		if !visit.AgentID.Valid || visit.AgentID.Int32 != userID { return errors.New("no eres el agente asignado a esta visita") }
		switch currentStatus {
		case StatusPending: newStatus = StatusWaitingClient
		case StatusWaitingAgent: newStatus = StatusConfirmed
		}
	} else if role == 1 { newStatus = StatusConfirmed }

	if newStatus == currentStatus { return errors.New("operación no válida o ya confirmada") }

	if err := s.repo.UpdateVisitStatus(ctx, visit.VisitID, newStatus); err != nil { return translateError(err) }
	return s.repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{
		VisitID: visit.VisitID, PreviousStatusID: currentStatus, NewStatusID: newStatus, ChangedByUserID: userID,
	})
}

func (s *service) RescheduleVisit(ctx context.Context, userID int32, visitUUID uuid.UUID, newDate time.Time) (VisitResponse, error) {
	// Iniciar Transacción Atómica
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	// 1. Obtener cita original
	oldVisit, err := txRepo.GetVisitByUUID(ctx, visitUUID)
	if err != nil { return VisitResponse{}, errors.New("visita no encontrada") }

	if oldVisit.StatusID == StatusCancelled {
		return VisitResponse{}, errors.New("no se puede reagendar una cita ya cancelada")
	}

	// 2. Validar permisos
	role, err := txRepo.GetUserRole(ctx, userID)
	if err != nil { return VisitResponse{}, err }

	canReschedule := (role == 1) || (role == 3 && oldVisit.ClientID == userID) || (role == 2 && oldVisit.AgentID.Valid && oldVisit.AgentID.Int32 == userID)
	if !canReschedule {
		return VisitResponse{}, errors.New("no tienes permiso para reagendar esta visita")
	}

	// 3. Crear la nueva visita usando el repositorio transaccional
	newVisit, err := s.scheduleVisitInternal(ctx, txRepo, oldVisit.ClientID, oldVisit.PropertyID, newDate)
	if err != nil {
		return VisitResponse{}, err
	}

	// 4. Cancelar la cita anterior
	if err := txRepo.UpdateVisitStatus(ctx, oldVisit.VisitID, StatusCancelled); err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al cancelar cita anterior: %w", err)
	}

	// Registrar historia de cancelación
	if err := txRepo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{
		VisitID:          oldVisit.VisitID,
		PreviousStatusID: oldVisit.StatusID,
		NewStatusID:      StatusCancelled,
		ChangedByUserID:  userID,
	}); err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al registrar historial: %w", err)
	}

	// Commit exitoso: Ambas acciones se guardan o ninguna
	if err := tx.Commit(ctx); err != nil {
		return VisitResponse{}, fmt.Errorf("fallo al confirmar cambios: %w", err)
	}

	return newVisit, nil
}

func (s *service) CompleteVisit(ctx context.Context, userID int32, visitUUID uuid.UUID) error {
	visit, err := s.repo.GetVisitByUUID(ctx, visitUUID)
	if err != nil { return errors.New("visita no encontrada") }

	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil { return err }

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
