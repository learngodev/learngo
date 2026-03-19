package shared

import "errors"

// ErrNotFound indicates repository query returned no rows.
var ErrNotFound = errors.New("repository: not found")

// Role defines platform roles.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

// AccountStatus indicates account availability state.
type AccountStatus string

const (
	AccountStatusActive                AccountStatus = "active"
	AccountStatusLocked                AccountStatus = "locked"
	AccountStatusPasswordResetRequired AccountStatus = "password_reset_required"
)
