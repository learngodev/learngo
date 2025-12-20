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
	timeSlotRepo  repository.TimeSlotRepository
	scheduleRepo  repository.CourseScheduleRepository
	sessionRepo   repository.CourseSessionRepository
	courseRepo    repository.CourseRepository
	teacherRepo   repository.TeacherRepository
	classroomRepo repository.ClassroomRepository
}

func NewScheduleService(
	timeSlotRepo repository.TimeSlotRepository,
	scheduleRepo repository.CourseScheduleRepository,
	sessionRepo repository.CourseSessionRepository,
	courseRepo repository.CourseRepository,
	teacherRepo repository.TeacherRepository,
	classroomRepo repository.ClassroomRepository,
) *ScheduleService {
	return &ScheduleService{
		timeSlotRepo:  timeSlotRepo,
		scheduleRepo:  scheduleRepo,
		sessionRepo:   sessionRepo,
		courseRepo:    courseRepo,
		teacherRepo:   teacherRepo,
		classroomRepo: classroomRepo,
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

func (s *ScheduleService) CreateSchedule(ctx context.Context, schoolID, courseID, classID, teacherID, slotID string, dayOfWeek int, location string, classroomID *string, startDate, endDate time.Time) (*domain.CourseSchedule, error) {
	if !startDate.Before(endDate) {
		return nil, fmt.Errorf("start_date must be before end_date")
	}

	// Validate slot exists and get times
	slot, err := s.timeSlotRepo.FindByID(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, fmt.Errorf("time slot not found")
	}

	// Validate classroom if provided
	if classroomID != nil && *classroomID != "" {
		// Check existence
		classroom, err := s.classroomRepo.GetByID(ctx, *classroomID)
		if err != nil {
			return nil, fmt.Errorf("classroom not found: %v", err)
		}
		location = classroom.Location

		// Check conflict
		existingSchedules, err := s.scheduleRepo.ListByClassroom(ctx, *classroomID)
		if err != nil {
			return nil, err
		}
		for _, sch := range existingSchedules {
			if sch.DayOfWeek == dayOfWeek && sch.SlotID == slotID {
				// Check date overlap
				// Overlap if (StartA <= EndB) and (EndA >= StartB)
				// Here we check if the new schedule overlaps with existing one
				if !sch.StartDate.After(endDate) && !sch.EndDate.Before(startDate) {
					return nil, fmt.Errorf("classroom is already booked for this slot")
				}
			}
		}
	}

	var tid *string
	if teacherID != "" {
		// Try as Profile ID first
		teacher, err := s.teacherRepo.GetByID(ctx, teacherID)
		if err != nil || teacher == nil {
			// Try as Account ID
			teacher, err = s.teacherRepo.GetByAccountID(ctx, teacherID)
			if err != nil {
				return nil, fmt.Errorf("teacher not found (checked ID and AccountID): %v", err)
			}
			if teacher == nil {
				return nil, fmt.Errorf("teacher not found")
			}
			// Use the resolved Profile ID
			teacherID = teacher.ID
		}
		tid = &teacherID

		// Check Teacher Conflict
		existingSchedules, err := s.scheduleRepo.ListByTeacher(ctx, teacherID)
		if err != nil {
			return nil, err
		}
		for _, sch := range existingSchedules {
			if sch.DayOfWeek == dayOfWeek && sch.SlotID == slotID {
				if !sch.StartDate.After(endDate) && !sch.EndDate.Before(startDate) {
					return nil, fmt.Errorf("teacher is already booked for this slot")
				}
			}
		}
	}

	// Check Class Conflict
	if classID != "" {
		existingSchedules, err := s.scheduleRepo.ListByClass(ctx, classID)
		if err != nil {
			return nil, err
		}
		for _, sch := range existingSchedules {
			if sch.DayOfWeek == dayOfWeek && sch.SlotID == slotID {
				if !sch.StartDate.After(endDate) && !sch.EndDate.Before(startDate) {
					return nil, fmt.Errorf("class is already booked for this slot")
				}
			}
		}
	}

	schedule := &domain.CourseSchedule{
		ID:          uuid.New().String(),
		SchoolID:    schoolID,
		CourseID:    courseID,
		ClassID:     classID,
		TeacherID:   tid,
		SlotID:      slotID,
		ClassroomID: classroomID,
		DayOfWeek:   dayOfWeek,
		Location:    location,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, err
	}

	// Generate sessions for this schedule
	startH, startM, _ := parseTime(slot.StartTime)
	endH, endM, _ := parseTime(slot.EndTime)

	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		weekday := int(d.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != dayOfWeek {
			continue
		}

		startsAt := time.Date(d.Year(), d.Month(), d.Day(), startH, startM, 0, 0, d.Location())
		endsAt := time.Date(d.Year(), d.Month(), d.Day(), endH, endM, 0, 0, d.Location())

		session := &domain.CourseSession{
			ID:        uuid.New().String(),
			CourseID:  courseID,
			ClassID:   classID,
			TeacherID: tid,
			SlotID:    slotID,
			StartsAt:  startsAt,
			EndsAt:    endsAt,
			Location:  location,
			Source:    "system",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		// Best effort creation, but log error if possible (we don't have logger here easily)
		// At least we are using the correct TeacherID now.
		if err := s.sessionRepo.Create(ctx, session); err != nil {
			// If we fail to create a session, we should probably know.
			// But failing the whole request might be too harsh if it's a duplicate?
			// For now, let's return error to be safe and debuggable.
			return nil, fmt.Errorf("failed to create session for %s: %v", startsAt, err)
		}
	}

	return schedule, nil
}

func (s *ScheduleService) DeleteSchedule(ctx context.Context, id string) error {
	return s.scheduleRepo.Delete(ctx, id)
}

func (s *ScheduleService) ListSchedules(ctx context.Context, schoolID string, courseID string) ([]domain.CourseScheduleDetail, error) {
	return s.scheduleRepo.ListDetailsBySchool(ctx, schoolID, courseID)
}

func (s *ScheduleService) GetScheduleStats(ctx context.Context, schoolID string) (*domain.ScheduleStats, error) {
	stats, err := s.scheduleRepo.GetStats(ctx, schoolID)
	if err != nil {
		return nil, err
	}

	// Get total courses count
	_, totalCourses, err := s.courseRepo.List(ctx, schoolID, 1, 1)
	if err != nil {
		return nil, err
	}
	stats.TotalCourses = totalCourses
	stats.UnscheduledCoursesCount = totalCourses - stats.ScheduledCoursesCount
	if stats.UnscheduledCoursesCount < 0 {
		stats.UnscheduledCoursesCount = 0
	}

	return stats, nil
}

// Session Generation

func (s *ScheduleService) GenerateSessions(ctx context.Context, schoolID string, start, end time.Time) error {
	if !start.Before(end) {
		return fmt.Errorf("start time must be before end time")
	}

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
	// Use a fixed timezone (CST) for day calculation to ensure consistency with user expectations
	loc := time.FixedZone("CST", 8*3600)

	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {
		dInLoc := d.In(loc)
		weekday := int(dInLoc.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		dbWeekday := weekday

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

			// Create session in the target timezone, then convert to UTC for storage
			startsAt := time.Date(dInLoc.Year(), dInLoc.Month(), dInLoc.Day(), startH, startM, 0, 0, loc).UTC()
			endsAt := time.Date(dInLoc.Year(), dInLoc.Month(), dInLoc.Day(), endH, endM, 0, 0, loc).UTC()

			// Check if session already exists (to avoid duplicates)
			exists, err := s.sessionRepo.Exists(ctx, sched.CourseID, sched.ClassID, sched.TeacherID, sched.SlotID, startsAt)
			if err != nil {
				return err
			}
			if exists {
				continue
			}

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
