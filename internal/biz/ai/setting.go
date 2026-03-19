package ai

import "time"

// AIAgentSetting stores per-school AI configuration.
type AIAgentSetting struct {
	ID                      string     `gorm:"primaryKey;size:36"`
	SchoolID                string     `gorm:"size:36;index"`
	Provider                AIProvider `gorm:"size:32"`
	Model                   string     `gorm:"size:128"`
	APIKey                  string     `gorm:"size:256"`
	BaseURL                 string     `gorm:"size:256"`
	Temperature             float32    `gorm:"type:real"`
	TopP                    float32    `gorm:"type:real"`
	MaxOutputTokens         int        `gorm:"default:0"`
	MaxDailyRequests        int        `gorm:"default:0"`
	MaxConcurrentRequests   int        `gorm:"default:0"`
	MaxConversationMessages int        `gorm:"default:0"`
	SystemPrompt            string     `gorm:"type:text"`
	VisionEnabled           bool       `gorm:"default:false"`
	UpdatedBy               string     `gorm:"size:36"`
	UpdatedByName           string     `gorm:"size:128"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// AIAgentSettingAudit tracks administrative changes to AI settings.
type AIAgentSettingAudit struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SchoolID     string    `gorm:"size:36;index"`
	OperatorID   string    `gorm:"size:36"`
	OperatorName string    `gorm:"size:128"`
	Action       string    `gorm:"size:64"`
	Detail       string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"index"`
}
