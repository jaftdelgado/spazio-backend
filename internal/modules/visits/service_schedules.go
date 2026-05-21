package visits

import (
	"context"
	"time"

	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (s *service) GetAvailableSlots(ctx context.Context, propertyID int32, date time.Time) ([]TimeSlot, error) {
	date = date.UTC()

	agentID, err := s.repo.GetPrimaryAgentForProperty(ctx, propertyID)
	if err != nil {
		return nil, err
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

		if !isAvailable {
			continue
		}

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
