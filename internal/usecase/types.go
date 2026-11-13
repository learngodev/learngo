package usecase

import (
	"encoding/json"
	"strings"
	"time"

	"learn-go/internal/domain/coursework"
)

// TimeISO8601 用于从 JSON 中解析 ISO8601 时间戳。
type TimeISO8601 struct {
	Time time.Time
}

// UnmarshalJSON 实现 json.Unmarshaler。
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

// ToAssignmentType 将字符串转换为 AssignmentType。
func ToAssignmentType(v string) coursework.AssignmentType {
	switch strings.ToLower(v) {
	case string(coursework.AssignmentExam):
		return coursework.AssignmentExam
	default:
		return coursework.AssignmentHomework
	}
}

// ToQuestionType 将字符串转换为 QuestionType。
func ToQuestionType(v string) coursework.QuestionType {
	switch strings.ToLower(v) {
	case string(coursework.QuestionChoice):
		return coursework.QuestionChoice
	case string(coursework.QuestionJudgement):
		return coursework.QuestionJudgement
	case string(coursework.QuestionEssay):
		return coursework.QuestionEssay
	default:
		return coursework.QuestionFill
	}
}
