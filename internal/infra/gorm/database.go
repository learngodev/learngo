package gorminfra

import (
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"learn-go/internal/biz/ai"
	"learn-go/internal/biz/assignment"
	"learn-go/internal/biz/conversation"
	"learn-go/internal/biz/course"
	"learn-go/internal/biz/identity"
	"learn-go/internal/biz/note"
	"learn-go/internal/biz/notification"
	"learn-go/internal/biz/school"
	"learn-go/internal/biz/storage"
	"learn-go/internal/biz/system"
	"learn-go/internal/config"
)

// Open initializes the shared GORM instance for the application.
func Open(cfg config.AppConfig) (*gorm.DB, error) {
	dialector, err := resolveDialector(cfg)
	if err != nil {
		return nil, err
	}

	return gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
}

// AutoMigrate applies schema changes and compatibility fixes.
func AutoMigrate(db *gorm.DB) error {
	migrator := db.Migrator()
	if migrator.HasTable("ai_chat_messages") {
		if err := migrator.DropTable("ai_chat_messages"); err != nil {
			return err
		}
	}
	if migrator.HasTable("ai_chat_sessions") {
		if err := migrator.DropTable("ai_chat_sessions"); err != nil {
			return err
		}
	}

	if err := db.AutoMigrate(
		&school.School{},
		&identity.Account{},
		&identity.Teacher{},
		&identity.Student{},
		&identity.StudentReminder{},
		&identity.TeacherStudentLink{},
		&school.Department{},
		&school.Class{},
		&course.Course{},
		&course.CourseChapter{},
		&course.CourseChapterAttachment{},
		&course.CourseStudent{},
		&course.CourseTeacher{},
		&course.CourseSlot{},
		&course.CourseSchedule{},
		&course.CourseSession{},
		&assignment.Assignment{},
		&assignment.AssignmentQuestion{},
		&assignment.AssignmentSubmission{},
		&assignment.SubmissionItem{},
		&assignment.SubmissionComment{},
		&conversation.Conversation{},
		&conversation.ConversationMember{},
		&conversation.Message{},
		&conversation.MessageReceipt{},
		&note.Note{},
		&note.NoteComment{},
		&storage.OssCredential{},
		&storage.OssPolicy{},
		&storage.OssAuditLog{},
		&system.SystemSwitch{},
		&system.SystemParameter{},
		&system.SystemBroadcast{},
		&system.SystemAuditLog{},
		&ai.AIAgentSetting{},
		&ai.AIAgentSettingAudit{},
		&ai.AIUsageLog{},
		&identity.PasswordResetToken{},
		&course.TimeSlot{},
		&course.Classroom{},
		&storage.File{},
		&assignment.AssignmentAttachment{},
		&assignment.SubmissionAttachment{},
		&notification.Notification{},
	); err != nil {
		return err
	}

	return ensurePartialUniqueIndexes(db)
}

func ensurePartialUniqueIndexes(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres", "sqlite":
		if err := db.Exec("DROP INDEX IF EXISTS idx_accounts_identifier").Error; err != nil {
			return err
		}
		return db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_identifier ON accounts (identifier) WHERE deleted_at IS NULL").Error
	default:
		return nil
	}
}

func resolveDialector(cfg config.AppConfig) (gorm.Dialector, error) {
	switch driver := strings.ToLower(cfg.DatabaseDriver); driver {
	case "postgres", "postgresql":
		return postgres.Open(cfg.DatabaseDSN), nil
	case "sqlite", "sqlite3", "":
		return sqlite.Open(cfg.DatabaseDSN), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DatabaseDriver)
	}
}
