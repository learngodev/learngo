package service

import (
	"encoding/json"
	assignmentbiz "learn-go/internal/biz/assignment"
	"strings"
	"time"
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
func ToAssignmentType(v string) assignmentbiz.AssignmentType {
	switch strings.ToLower(v) {
	case string(assignmentbiz.AssignmentExam):
		return assignmentbiz.AssignmentExam
	default:
		return assignmentbiz.AssignmentHomework
	}
}

// ToQuestionType maps string to QuestionType.
func ToQuestionType(v string) assignmentbiz.QuestionType {
	switch strings.ToLower(v) {
	case string(assignmentbiz.QuestionChoice):
		return assignmentbiz.QuestionChoice
	case string(assignmentbiz.QuestionJudgement):
		return assignmentbiz.QuestionJudgement
	case string(assignmentbiz.QuestionEssay):
		return assignmentbiz.QuestionEssay
	default:
		return assignmentbiz.QuestionFill
	}
}
