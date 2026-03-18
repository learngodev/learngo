package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

type scheduleConflictDetails struct {
	ResourceType   string `json:"resource_type"`
	ResourceID     string `json:"resource_id,omitempty"`
	Date           string `json:"date"`
	TimeWindow     string `json:"time_window"`
	WindowStart    string `json:"window_start"`
	WindowEnd      string `json:"window_end"`
	CandidateClass string `json:"candidate_class_id,omitempty"`
	ExistingClass  string `json:"existing_class_id,omitempty"`
}

type scheduleValidationDetails struct {
	Field  string      `json:"field,omitempty"`
	Reason string      `json:"reason"`
	Value  interface{} `json:"value,omitempty"`
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
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return nil, newScheduleValidationError("星期参数无效，应为 1 到 7", "day_of_week", "out_of_range", dayOfWeek)
	}
	if !startDate.Before(endDate) {
		return nil, newScheduleValidationError("开始日期必须早于结束日期", "date_range", "start_not_before_end", map[string]string{
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
		})
	}

	// Validate slot exists and get times
	slot, err := s.timeSlotRepo.FindByID(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, newScheduleValidationError("未找到对应时间段", "slot_id", "not_found", slotID)
	}
	if slot.SchoolID != schoolID {
		return nil, newScheduleValidationError("该时间段不属于当前学校", "slot_id", "school_mismatch", slotID)
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
			return nil, newScheduleValidationError("未找到对应教室", "classroom_id", "not_found", *classroomID)
		}
		if classroom == nil {
			return nil, newScheduleValidationError("未找到对应教室", "classroom_id", "not_found", *classroomID)
		}
		if classroom.SchoolID != schoolID {
			return nil, newScheduleValidationError("该教室不属于当前学校", "classroom_id", "school_mismatch", *classroomID)
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
				return nil, newScheduleValidationError("未找到对应教师", "teacher_id", "not_found", teacherID)
			}
			if teacher == nil {
				return nil, newScheduleValidationError("未找到对应教师", "teacher_id", "not_found", teacherID)
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

	candidateSessions, err := buildSessionsForSchedule(
		courseID,
		classID,
		tid,
		slotID,
		classroomID,
		dayOfWeek,
		location,
		startDate,
		endDate,
		*slot,
	)
	if err != nil {
		return nil, err
	}
	for i := range candidateSessions {
		if err := s.ensureNoSessionConflict(ctx, candidateSessions[i], ""); err != nil {
			return nil, err
		}
	}

	now := time.Now()
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
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, err
	}

	for i := range candidateSessions {
		session := candidateSessions[i]
		session.ID = uuid.New().String()
		session.CreatedAt = now
		session.UpdatedAt = now
		if err := s.sessionRepo.Create(ctx, &session); err != nil {
			return nil, fmt.Errorf("failed to create session for %s: %v", session.StartsAt, err)
		}
	}

	return schedule, nil
}

func (s *ScheduleService) DeleteSchedule(ctx context.Context, id string) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if schedule == nil {
		return nil
	}

	if err := s.deleteGeneratedSessionsBySchedule(ctx, *schedule); err != nil {
		return err
	}

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
		return newScheduleValidationError("生成范围开始时间必须早于结束时间", "date_range", "start_not_before_end", map[string]string{
			"start": start.Format(time.RFC3339),
			"end":   end.Format(time.RFC3339),
		})
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
				return newScheduleValidationError("时间段开始时间格式不正确", "start_time", "invalid_time_format", slot.StartTime)
			}
			endH, endM, err := parseTime(slot.EndTime)
			if err != nil {
				return newScheduleValidationError("时间段结束时间格式不正确", "end_time", "invalid_time_format", slot.EndTime)
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

func buildSessionsForSchedule(
	courseID string,
	classID string,
	teacherID *string,
	slotID string,
	classroomID *string,
	dayOfWeek int,
	location string,
	startDate time.Time,
	endDate time.Time,
	slot domain.TimeSlot,
) ([]domain.CourseSession, error) {
	startH, startM, err := parseTime(slot.StartTime)
	if err != nil {
		return nil, newScheduleValidationError("时间段开始时间格式不正确", "start_time", "invalid_time_format", slot.StartTime)
	}
	endH, endM, err := parseTime(slot.EndTime)
	if err != nil {
		return nil, newScheduleValidationError("时间段结束时间格式不正确", "end_time", "invalid_time_format", slot.EndTime)
	}

	sessions := make([]domain.CourseSession, 0)
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		weekday := int(d.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != dayOfWeek {
			continue
		}

		startsAt := time.Date(
			d.Year(),
			d.Month(),
			d.Day(),
			startH,
			startM,
			0,
			0,
			d.Location(),
		)
		endsAt := time.Date(
			d.Year(),
			d.Month(),
			d.Day(),
			endH,
			endM,
			0,
			0,
			d.Location(),
		)

		sessions = append(sessions, domain.CourseSession{
			CourseID:    courseID,
			ClassID:     classID,
			TeacherID:   teacherID,
			SlotID:      slotID,
			ClassroomID: classroomID,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			Location:    location,
			Source:      "system",
		})
	}

	return sessions, nil
}

// Teacher Adjustment

func (s *ScheduleService) UpdateSession(ctx context.Context, sessionID string, newSlotID string, newDate time.Time, newLocation string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return newScheduleValidationError("未找到对应课次", "session_id", "not_found", sessionID)
	}

	// Get new slot details
	slot, err := s.timeSlotRepo.FindByID(ctx, newSlotID)
	if err != nil {
		return err
	}
	if slot == nil {
		return newScheduleValidationError("未找到对应时间段", "slot_id", "not_found", newSlotID)
	}

	startH, startM, err := parseTime(slot.StartTime)
	if err != nil {
		return newScheduleValidationError("时间段开始时间格式不正确", "start_time", "invalid_time_format", slot.StartTime)
	}
	endH, endM, err := parseTime(slot.EndTime)
	if err != nil {
		return newScheduleValidationError("时间段结束时间格式不正确", "end_time", "invalid_time_format", slot.EndTime)
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
			overlapStart, overlapEnd := overlapDateRange(startDate, endDate, sch.StartDate, sch.EndDate)
			resourceLabel := "资源"
			resourceType := "resource"
			resourceID := ""
			switch scope {
			case "teacher":
				resourceLabel = "教师"
				resourceType = "teacher"
				if sch.TeacherID != nil {
					resourceID = *sch.TeacherID
				}
			case "class":
				resourceLabel = "班级"
				resourceType = "class"
				resourceID = sch.ClassID
			case "classroom":
				resourceLabel = "教室"
				resourceType = "classroom"
				if sch.ClassroomID != nil {
					resourceID = *sch.ClassroomID
				}
			}
			message := fmt.Sprintf(
				"%s在%s %s-%s（%s 至 %s）已有排课",
				resourceLabel,
				weekdayLabel(dayOfWeek),
				minuteToClock(startMinute),
				minuteToClock(endMinute),
				overlapStart.Format("2006-01-02"),
				overlapEnd.Format("2006-01-02"),
			)
			return newScheduleConflictError(message, scheduleConflictDetails{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Date:         overlapStart.Format("2006-01-02"),
				TimeWindow:   fmt.Sprintf("%s-%s", minuteToClock(startMinute), minuteToClock(endMinute)),
				WindowStart:  overlapStart.Format(time.RFC3339),
				WindowEnd:    overlapEnd.Format(time.RFC3339),
			})
		}
	}
	return nil
}

func (s *ScheduleService) ensureNoSessionConflict(ctx context.Context, session domain.CourseSession, excludeSessionID string) error {
	allSessions, err := s.sessionRepo.ListBetween(ctx, session.StartsAt, session.EndsAt)
	if err != nil {
		return err
	}

	for _, existing := range allSessions {
		if existing.ID == excludeSessionID {
			continue
		}
		if !timeRangesOverlapTime(session.StartsAt, session.EndsAt, existing.StartsAt, existing.EndsAt) {
			continue
		}
		timeWindow := fmt.Sprintf(
			"%s %s-%s",
			session.StartsAt.Format("2006-01-02"),
			session.StartsAt.Format("15:04"),
			session.EndsAt.Format("15:04"),
		)
		if existing.ClassID == session.ClassID {
			return newScheduleConflictError(
				fmt.Sprintf("班级在%s已有课程安排", timeWindow),
				scheduleConflictDetails{
					ResourceType:   "class",
					ResourceID:     session.ClassID,
					Date:           session.StartsAt.Format("2006-01-02"),
					TimeWindow:     fmt.Sprintf("%s-%s", session.StartsAt.Format("15:04"), session.EndsAt.Format("15:04")),
					WindowStart:    session.StartsAt.Format(time.RFC3339),
					WindowEnd:      session.EndsAt.Format(time.RFC3339),
					CandidateClass: session.ClassID,
					ExistingClass:  existing.ClassID,
				},
			)
		}
		if session.TeacherID != nil && existing.TeacherID != nil && *session.TeacherID == *existing.TeacherID {
			return newScheduleConflictError(
				fmt.Sprintf("教师在%s已有课程安排", timeWindow),
				scheduleConflictDetails{
					ResourceType:   "teacher",
					ResourceID:     *session.TeacherID,
					Date:           session.StartsAt.Format("2006-01-02"),
					TimeWindow:     fmt.Sprintf("%s-%s", session.StartsAt.Format("15:04"), session.EndsAt.Format("15:04")),
					WindowStart:    session.StartsAt.Format(time.RFC3339),
					WindowEnd:      session.EndsAt.Format(time.RFC3339),
					CandidateClass: session.ClassID,
					ExistingClass:  existing.ClassID,
				},
			)
		}
		if session.ClassroomID != nil && existing.ClassroomID != nil && *session.ClassroomID == *existing.ClassroomID {
			return newScheduleConflictError(
				fmt.Sprintf("教室在%s已被占用", timeWindow),
				scheduleConflictDetails{
					ResourceType:   "classroom",
					ResourceID:     *session.ClassroomID,
					Date:           session.StartsAt.Format("2006-01-02"),
					TimeWindow:     fmt.Sprintf("%s-%s", session.StartsAt.Format("15:04"), session.EndsAt.Format("15:04")),
					WindowStart:    session.StartsAt.Format(time.RFC3339),
					WindowEnd:      session.EndsAt.Format(time.RFC3339),
					CandidateClass: session.ClassID,
					ExistingClass:  existing.ClassID,
				},
			)
		}
	}

	return nil
}

func (s *ScheduleService) deleteGeneratedSessionsBySchedule(ctx context.Context, schedule domain.CourseSchedule) error {
	loc := schedule.StartDate.Location()
	startDate := dateAtLocation(schedule.StartDate, loc)
	endDate := dateAtLocation(schedule.EndDate, loc)

	start := startDate.AddDate(0, 0, -2)
	end := endDate.AddDate(0, 0, 3)
	existingSessions, err := s.sessionRepo.ListBetween(ctx, start, end)
	if err != nil {
		return err
	}
	if len(existingSessions) == 0 {
		return nil
	}

	idsToDelete := make([]string, 0)
	for _, session := range existingSessions {
		if session.Source != "system" {
			continue
		}
		if session.CourseID != schedule.CourseID || session.ClassID != schedule.ClassID || session.SlotID != schedule.SlotID {
			continue
		}
		if !sameOptionalString(session.TeacherID, schedule.TeacherID) {
			continue
		}
		if !sameOptionalString(session.ClassroomID, schedule.ClassroomID) {
			continue
		}

		sessionDate := dateAtLocation(session.StartsAt, loc)
		if sessionDate.Before(startDate) || sessionDate.After(endDate) {
			continue
		}
		weekday := int(session.StartsAt.In(loc).Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != schedule.DayOfWeek {
			continue
		}
		idsToDelete = append(idsToDelete, session.ID)
	}

	return s.sessionRepo.DeleteByIDs(ctx, idsToDelete)
}

func dateAtLocation(ts time.Time, loc *time.Location) time.Time {
	inLoc := ts.In(loc)
	return time.Date(inLoc.Year(), inLoc.Month(), inLoc.Day(), 0, 0, 0, 0, loc)
}

func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
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
		return domain.TimeSlot{}, newScheduleValidationError("未找到对应时间段", "slot_id", "not_found", slotID)
	}
	slotMap[slotID] = *resolved
	return *resolved, nil
}

func slotRangeMinutes(slot domain.TimeSlot) (int, int, error) {
	startH, startM, err := parseTime(slot.StartTime)
	if err != nil {
		return 0, 0, newScheduleValidationError("时间段开始时间格式不正确", "start_time", "invalid_time_format", slot.StartTime)
	}
	endH, endM, err := parseTime(slot.EndTime)
	if err != nil {
		return 0, 0, newScheduleValidationError("时间段结束时间格式不正确", "end_time", "invalid_time_format", slot.EndTime)
	}
	startMinute := startH*60 + startM
	endMinute := endH*60 + endM
	if endMinute <= startMinute {
		return 0, 0, newScheduleValidationError("时间段结束时间必须晚于开始时间", "time_range", "end_not_after_start", map[string]string{
			"start_time": slot.StartTime,
			"end_time":   slot.EndTime,
		})
	}
	return startMinute, endMinute, nil
}

func overlapDateRange(aStart, aEnd, bStart, bEnd time.Time) (time.Time, time.Time) {
	start := aStart
	if bStart.After(start) {
		start = bStart
	}
	end := aEnd
	if bEnd.Before(end) {
		end = bEnd
	}
	return start, end
}

func weekdayLabel(day int) string {
	labels := map[int]string{
		1: "周一",
		2: "周二",
		3: "周三",
		4: "周四",
		5: "周五",
		6: "周六",
		7: "周日",
	}
	if v, ok := labels[day]; ok {
		return v
	}
	return fmt.Sprintf("周%d", day)
}

func minuteToClock(v int) string {
	h := v / 60
	m := v % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func dateRangesOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return !aStart.After(bEnd) && !bStart.After(aEnd)
}

func timeRangesOverlapMinute(aStart, aEnd, bStart, bEnd int) bool {
	return aStart < bEnd && bStart < aEnd
}

func timeRangesOverlapTime(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func newScheduleConflictError(message string, details scheduleConflictDetails) error {
	return &AppError{
		Code:    ErrScheduleConflict.Error(),
		Status:  http.StatusConflict,
		Message: message,
		Details: details,
		Cause:   ErrScheduleConflict,
	}
}

func newScheduleValidationError(message, field, reason string, value interface{}) error {
	return &AppError{
		Code:    ErrScheduleValidation.Error(),
		Status:  http.StatusBadRequest,
		Message: message,
		Details: scheduleValidationDetails{
			Field:  field,
			Reason: reason,
			Value:  value,
		},
		Cause: ErrScheduleValidation,
	}
}
