package service

import (
	"context"
	"fmt"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"github.com/google/uuid"
)

type ScheduleService struct {
	timeSlotRepo repository.TimeSlotRepository
	scheduleRepo repository.CourseScheduleRepository
	sessionRepo  repository.CourseSessionRepository
	courseRepo   repository.CourseRepository
}

func NewScheduleService(
	timeSlotRepo repository.TimeSlotRepository,
	scheduleRepo repository.CourseScheduleRepository,
	sessionRepo repository.CourseSessionRepository,
	courseRepo repository.CourseRepository,
) *ScheduleService {
	return &ScheduleService{
		timeSlotRepo: timeSlotRepo,
		scheduleRepo: scheduleRepo,
		sessionRepo:  sessionRepo,
		courseRepo:   courseRepo,
	}
}

// Time Slot Management

func (s *ScheduleService) CreateTimeSlot(ctx context.Context, schoolID, name, startTime, endTime string) (*domain.TimeSlot, error) {
	slot := &domain.TimeSlot{
		ID:        uuid.New().String(),
		SchoolID:  schoolID,
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.timeSlotRepo.Create(ctx, slot); err != nil {
		return nil, err
	}
	return slot, nil
}

func (s *ScheduleService) ListTimeSlots(ctx context.Context, schoolID string) ([]domain.TimeSlot, error) {
	return s.timeSlotRepo.List(ctx, schoolID)
}

// Schedule Rule Management

func (s *ScheduleService) CreateSchedule(ctx context.Context, schoolID, courseID, classID, teacherID, slotID string, dayOfWeek int, location string, startDate, endDate time.Time) (*domain.CourseSchedule, error) {
	schedule := &domain.CourseSchedule{
		ID:        uuid.New().String(),
		SchoolID:  schoolID,
		CourseID:  courseID,
		ClassID:   classID,
		TeacherID: teacherID,
		SlotID:    slotID,
		DayOfWeek: dayOfWeek,
		Location:  location,
		StartDate: startDate,
		EndDate:   endDate,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

// Session Generation

func (s *ScheduleService) GenerateSessions(ctx context.Context, schoolID string, start, end time.Time) error {
	// 1. List all schedules for the school
	schedules, err := s.scheduleRepo.ListBySchool(ctx, schoolID)
	if err != nil {
		return err
	}

	// 2. Pre-fetch time slots
	slots, err := s.timeSlotRepo.List(ctx, schoolID)
	if err != nil {
		return err
	}
	slotMap := make(map[string]domain.TimeSlot)
	for _, slot := range slots {
		slotMap[slot.ID] = slot
	}

	// 3. Iterate through days in range
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {
		weekday := int(d.Weekday())
		if weekday == 0 {
			weekday = 7
		} // Convert Sunday=0 to 7 if needed, or match domain logic.
		// Go's time.Weekday: Sunday=0, Monday=1...Saturday=6.
		// My DB schema comment said 1=Monday, 7=Sunday.
		// So if d.Weekday() == 0 (Sunday), use 7.
		dbWeekday := weekday
		if dbWeekday == 0 {
			dbWeekday = 7
		}

		for _, sched := range schedules {
			if sched.DayOfWeek != dbWeekday {
				continue
			}
			if d.Before(sched.StartDate) || d.After(sched.EndDate) {
				continue
			}

			slot, ok := slotMap[sched.SlotID]
			if !ok {
				continue
			}

			// Parse HH:MM
			startH, startM, _ := parseTime(slot.StartTime)
			endH, endM, _ := parseTime(slot.EndTime)

			startsAt := time.Date(d.Year(), d.Month(), d.Day(), startH, startM, 0, 0, d.Location())
			endsAt := time.Date(d.Year(), d.Month(), d.Day(), endH, endM, 0, 0, d.Location())

			// Check if session already exists (to avoid duplicates)
			// Ideally repo has FindByScheduleAndDate or similar.
			// For now, let's assume we just create. Or we can rely on unique constraints if we had them.
			// A better approach is to list existing sessions for this range and skip.

			session := &domain.CourseSession{
				ID:        uuid.New().String(),
				CourseID:  sched.CourseID,
				ClassID:   sched.ClassID,
				TeacherID: sched.TeacherID,
				SlotID:    sched.SlotID,
				StartsAt:  startsAt,
				EndsAt:    endsAt,
				Location:  sched.Location,
				Source:    "system",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.sessionRepo.Create(ctx, session); err != nil {
				// Log error or continue
				return err
			}
		}
	}
	return nil
}

func parseTime(t string) (int, int, error) {
	var h, m int
	_, err := fmt.Sscanf(t, "%d:%d", &h, &m)
	return h, m, err
}

// Teacher Adjustment

func (s *ScheduleService) UpdateSession(ctx context.Context, sessionID string, newSlotID string, newDate time.Time, newLocation string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Get new slot details
	slot, err := s.timeSlotRepo.FindByID(ctx, newSlotID)
	if err != nil {
		return err
	}
	if slot == nil {
		return fmt.Errorf("slot not found")
	}

	startH, startM, _ := parseTime(slot.StartTime)
	endH, endM, _ := parseTime(slot.EndTime)

	startsAt := time.Date(newDate.Year(), newDate.Month(), newDate.Day(), startH, startM, 0, 0, newDate.Location())
	endsAt := time.Date(newDate.Year(), newDate.Month(), newDate.Day(), endH, endM, 0, 0, newDate.Location())

	session.SlotID = newSlotID
	session.StartsAt = startsAt
	session.EndsAt = endsAt
	session.Location = newLocation
	session.Source = "teacher"
	session.UpdatedAt = time.Now()

	return s.sessionRepo.Update(ctx, session)
}
