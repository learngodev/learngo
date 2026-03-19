package school

import (
	"context"
)

// SchoolRepository handles school persistence.
type SchoolRepository interface {
	List(ctx context.Context) ([]School, error)
}
