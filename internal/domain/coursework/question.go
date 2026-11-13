package coursework

// QuestionType 枚举支持的题型。
type QuestionType string

const (
	QuestionFill      QuestionType = "fill"
	QuestionChoice    QuestionType = "choice"
	QuestionJudgement QuestionType = "judge"
	QuestionEssay     QuestionType = "essay"
)

// AssignmentQuestion 保存题目内容。
type AssignmentQuestion struct {
	ID           string       `gorm:"primaryKey;size:36"`
	AssignmentID string       `gorm:"size:36;index"`
	Type         QuestionType `gorm:"size:16"`
	Prompt       string       `gorm:"type:text"`
	Options      string       `gorm:"type:text"` // 选择题选项，JSON 编码
	Answer       string       `gorm:"type:text"`
	Score        float64
	OrderIndex   int
}
