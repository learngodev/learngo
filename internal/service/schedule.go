package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

var (
	// ErrScheduleValidation indicates schedule input is invalid.
	ErrScheduleValidation = errors.New("invalid schedule input")
	// ErrScheduleConflict indicates schedule conflicts with existing resources.
	ErrScheduleConflict = errors.New("schedule conflict")
)

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
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return nil, fmt.Errorf("%w: day_of_week must be between 1 and 7", ErrScheduleValidation)
	}
	if !startDate.Before(endDate) {
		return nil, fmt.Errorf("%w: start_date must be before end_date", ErrScheduleValidation)
	}

	// Validate slot exists and get times
	slot, err := s.timeSlotRepo.FindByID(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, fmt.Errorf("%w: time slot not found", ErrScheduleValidation)
	}
	newStartMinute, newEndMinute, err := slotRangeMinutes(*slot)
	if err != nil {
		return nil, err
	}

	availableSlots, err := s.timeSlotRepo.List(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	slotMap := make(map[string]domain.TimeSlot, len(availableSlots))
	for _, item := range availableSlots {
		slotMap[item.ID] = item
	}

	// Validate classroom if provided
	if classroomID != nil && *classroomID != "" {
		// Check existence
		classroom, err := s.classroomRepo.GetByID(ctx, *classroomID)
		if err != nil {
			return nil, fmt.Errorf("%w: classroom not found", ErrScheduleValidation)
		}
		location = classroom.Location

		// Check conflict
		existingSchedules, err := s.scheduleRepo.ListByClassroom(ctx, *classroomID)
		if err != nil {
			return nil, err
		}
		if err := s.ensureNoScheduleConflict(ctx, "classroom", existingSchedules, dayOfWeek, startDate, endDate, newStartMinute, newEndMinute, slotMap); err != nil {
			return nil, err
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
				return nil, fmt.Errorf("%w: teacher not found", ErrScheduleValidation)
			}
			if teacher == nil {
				return nil, fmt.Errorf("%w: teacher not found", ErrScheduleValidation)
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
		if err := s.ensureNoScheduleConflict(ctx, "teacher", existingSchedules, dayOfWeek, startDate, endDate, newStartMinute, newEndMinute, slotMap); err != nil {
			return nil, err
		}
	}

	// Check Class Conflict
	if classID != "" {
		existingSchedules, err := s.scheduleRepo.ListByClass(ctx, classID)
		if err != nil {
			return nil, err
		}
		if err := s.ensureNoScheduleConflict(ctx, "class", existingSchedules, dayOfWeek, startDate, endDate, newStartMinute, newEndMinute, slotMap); err != nil {
			return nil, err
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
	startH, startM, err := parseTime(slot.StartTime)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid slot start time", ErrScheduleValidation)
	}
	endH, endM, err := parseTime(slot.EndTime)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid slot end time", ErrScheduleValidation)
	}

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
			ID:          uuid.New().String(),
			CourseID:    courseID,
			ClassID:     classID,
			TeacherID:   tid,
			SlotID:      slotID,
			ClassroomID: classroomID,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			Location:    location,
			Source:      "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.ensureNoSessionConflict(ctx, *session, ""); err != nil {
			return nil, err
		}
		if err := s.sessionRepo.Create(ctx, session); err != nil {
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
		return fmt.Errorf("%w: start time must be before end time", ErrScheduleValidation)
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
			startH, startM, err := parseTime(slot.StartTime)
			if err != nil {
				return fmt.Errorf("%w: invalid slot start time", ErrScheduleValidation)
			}
			endH, endM, err := parseTime(slot.EndTime)
			if err != nil {
				return fmt.Errorf("%w: invalid slot end time", ErrScheduleValidation)
			}

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
				ID:          uuid.New().String(),
				CourseID:    sched.CourseID,
				ClassID:     sched.ClassID,
				TeacherID:   sched.TeacherID,
				SlotID:      sched.SlotID,
				ClassroomID: sched.ClassroomID,
				StartsAt:    startsAt,
				EndsAt:      endsAt,
				Location:    sched.Location,
				Source:      "system",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := s.ensureNoSessionConflict(ctx, *session, ""); err != nil {
				return err
			}
			if err := s.sessionRepo.Create(ctx, session); err != nil {
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
		return fmt.Errorf("%w: session not found", ErrScheduleValidation)
	}

	// Get new slot details
	slot, err := s.timeSlotRepo.FindByID(ctx, newSlotID)
	if err != nil {
		return err
	}
	if slot == nil {
		return fmt.Errorf("%w: slot not found", ErrScheduleValidation)
	}

	startH, startM, err := parseTime(slot.StartTime)
	if err != nil {
		return fmt.Errorf("%w: invalid slot start time", ErrScheduleValidation)
	}
	endH, endM, err := parseTime(slot.EndTime)
	if err != nil {
		return fmt.Errorf("%w: invalid slot end time", ErrScheduleValidation)
	}

	startsAt := time.Date(newDate.Year(), newDate.Month(), newDate.Day(), startH, startM, 0, 0, newDate.Location())
	endsAt := time.Date(newDate.Year(), newDate.Month(), newDate.Day(), endH, endM, 0, 0, newDate.Location())

	candidate := *session
	candidate.SlotID = newSlotID
	candidate.StartsAt = startsAt
	candidate.EndsAt = endsAt
	if strings.TrimSpace(newLocation) != "" {
		candidate.Location = newLocation
	}
	if err := s.ensureNoSessionConflict(ctx, candidate, session.ID); err != nil {
		return err
	}

	session.SlotID = newSlotID
	session.StartsAt = startsAt
	session.EndsAt = endsAt
	if strings.TrimSpace(newLocation) != "" {
		session.Location = newLocation
	}
	session.Source = "teacher"
	session.UpdatedAt = time.Now()

	return s.sessionRepo.Update(ctx, session)
}

func (s *ScheduleService) ensureNoScheduleConflict(
	ctx context.Context,
	scope string,
	schedules []domain.CourseSchedule,
	dayOfWeek int,
	startDate time.Time,
	endDate time.Time,
	newStartMinute int,
	newEndMinute int,
	slotMap map[string]domain.TimeSlot,
) error {
	for _, sch := range schedules {
		if sch.DayOfWeek != dayOfWeek {
			continue
		}
		if !dateRangesOverlap(startDate, endDate, sch.StartDate, sch.EndDate) {
			continue
		}
		slot, err := s.resolveSlot(ctx, sch.SlotID, slotMap)
		if err != nil {
			return err
		}
		startMinute, endMinute, err := slotRangeMinutes(slot)
		if err != nil {
			return err
		}
		if timeRangesOverlapMinute(newStartMinute, newEndMinute, startMinute, endMinute) {
			return fmt.Errorf("%w: %s is already booked for overlapping time", ErrScheduleConflict, scope)
		}
	}
	return nil
}

func (s *ScheduleService) ensureNoSessionConflict(ctx context.Context, session domain.CourseSession, excludeSessionID string) error {
	allSessions, err := s.sessionRepo.ListBetween(ctx, session.StartsAt, session.EndsAt)
	if err != nil {
		return err
	}

	targetLocation := strings.TrimSpace(strings.ToLower(session.Location))
	for _, existing := range allSessions {
		if existing.ID == excludeSessionID {
			continue
		}
		if !timeRangesOverlapTime(session.StartsAt, session.EndsAt, existing.StartsAt, existing.EndsAt) {
			continue
		}
		if existing.ClassID == session.ClassID {
			return fmt.Errorf("%w: class is already booked for overlapping time", ErrScheduleConflict)
		}
		if session.TeacherID != nil && existing.TeacherID != nil && *session.TeacherID == *existing.TeacherID {
			return fmt.Errorf("%w: teacher is already booked for overlapping time", ErrScheduleConflict)
		}
		if session.ClassroomID != nil && existing.ClassroomID != nil && *session.ClassroomID == *existing.ClassroomID {
			return fmt.Errorf("%w: classroom is already booked for overlapping time", ErrScheduleConflict)
		}
		if targetLocation != "" && strings.TrimSpace(strings.ToLower(existing.Location)) == targetLocation {
			return fmt.Errorf("%w: classroom is already booked for overlapping time", ErrScheduleConflict)
		}
	}

	return nil
}

func (s *ScheduleService) resolveSlot(ctx context.Context, slotID string, slotMap map[string]domain.TimeSlot) (domain.TimeSlot, error) {
	if slot, ok := slotMap[slotID]; ok {
		return slot, nil
	}
	resolved, err := s.timeSlotRepo.FindByID(ctx, slotID)
	if err != nil {
		return domain.TimeSlot{}, err
	}
	if resolved == nil {
		return domain.TimeSlot{}, fmt.Errorf("%w: time slot not found", ErrScheduleValidation)
	}
	slotMap[slotID] = *resolved
	return *resolved, nil
}

func slotRangeMinutes(slot domain.TimeSlot) (int, int, error) {
	startH, startM, err := parseTime(slot.StartTime)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: invalid slot start time", ErrScheduleValidation)
	}
	endH, endM, err := parseTime(slot.EndTime)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: invalid slot end time", ErrScheduleValidation)
	}
	startMinute := startH*60 + startM
	endMinute := endH*60 + endM
	if endMinute <= startMinute {
		return 0, 0, fmt.Errorf("%w: slot end time must be after start time", ErrScheduleValidation)
	}
	return startMinute, endMinute, nil
}

func dateRangesOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func timeRangesOverlapMinute(aStart, aEnd, bStart, bEnd int) bool {
	return aStart < bEnd && bStart < aEnd
}

func timeRangesOverlapTime(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}
