package usecase

import (
	"encoding/json"
	"strings"
	"time"

	"learn-go/internal/domain"
)

// TimeISO8601 helps parse ISO8601 timestamps from JSON.
type TimeISO8601 struct {
	Time time.Time
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *TimeISO8601) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// ToAssignmentType maps string to AssignmentType.
func ToAssignmentType(v string) domain.AssignmentType {
	switch strings.ToLower(v) {
	case string(domain.AssignmentExam):
		return domain.AssignmentExam
	default:
		return domain.AssignmentHomework
	}
}

// ToQuestionType maps string to QuestionType.
func ToQuestionType(v string) domain.QuestionType {
	switch strings.ToLower(v) {
	case string(domain.QuestionChoice):
		return domain.QuestionChoice
	case string(domain.QuestionJudgement):
		return domain.QuestionJudgement
	case string(domain.QuestionEssay):
		return domain.QuestionEssay
	default:
		return domain.QuestionFill
	}
}
