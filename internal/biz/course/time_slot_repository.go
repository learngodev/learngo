package course

import "context"

// TimeSlotRepository manages class time slots.
type TimeSlotRepository interface {
	Create(ctx context.Context, slot *TimeSlot) error
	Update(ctx context.Context, slot *TimeSlot) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, schoolID string) ([]TimeSlot, error)
	FindByID(ctx context.Context, id string) (*TimeSlot, error)
}

// CourseSlotRepository manages course slots.
type CourseSlotRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]CourseSlot, error)
}
