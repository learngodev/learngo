package app

import (
	"fmt"
	"strings"
	"time"

	"learn-go/internal/config"
	"learn-go/internal/domain/aiagent"
	"learn-go/internal/domain/coursework"
	"learn-go/internal/domain/curriculum"
	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/library"
	"learn-go/internal/domain/messaging"
	"learn-go/internal/domain/notebook"
	"learn-go/internal/domain/notification"
	"learn-go/internal/domain/organization"
	"learn-go/internal/domain/roster"
	"learn-go/internal/domain/scheduling"
	"learn-go/internal/domain/storage"
	"learn-go/internal/domain/system"
	apihandlers "learn-go/internal/handler"
	"learn-go/internal/infra/objectstorage"
	"learn-go/internal/infra/realtime"
	gormrepo "learn-go/internal/infra/repository/gorm"
	"learn-go/internal/infra/security"
	"learn-go/internal/interface/grpcserver"
	"learn-go/internal/interface/imstream"
	"learn-go/internal/server"
	"learn-go/internal/usecase"
	"learn-go/pkg/logger"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Application 负责装配各个用例与传输层。
type Application struct {
	db   *gorm.DB
	log  *logger.Logger
	http *server.HTTPServer
	grpc *server.GRPCServer
}

// New 创建应用并准备好所有依赖。
func New() (*Application, error) {
	cfg := config.Load()

	log := logger.New()

	dialector, err := resolveDialector(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	accountRepo := gormrepo.NewAccountStore(db)
	teacherRepo := gormrepo.NewTeacherStore(db)
	studentRepo := gormrepo.NewStudentStore(db)
	departmentRepo := gormrepo.NewDepartmentStore(db)
	classRepo := gormrepo.NewClassStore(db)
	courseRepo := gormrepo.NewCourseStore(db)
	courseChapterRepo := gormrepo.NewCourseChapterStore(db)
	courseSlotRepo := gormrepo.NewCourseSlotStore(db)
	courseSessionRepo := gormrepo.NewCourseSessionStore(db)
	teacherStudentRepo := gormrepo.NewTeacherStudentStore(db)
	assignmentRepo := gormrepo.NewAssignmentStore(db)
	submissionRepo := gormrepo.NewSubmissionStore(db)
	submissionCommentRepo := gormrepo.NewSubmissionCommentStore(db)
	studentReminderRepo := gormrepo.NewStudentReminderStore(db)
	noteRepo := gormrepo.NewNoteStore(db)
	noteCommentRepo := gormrepo.NewNoteCommentStore(db)
	conversationRepo := gormrepo.NewConversationStore(db)
	messageRepo := gormrepo.NewMessageStore(db)
	receiptRepo := gormrepo.NewMessageReceiptStore(db)
	ossCredentialRepo := gormrepo.NewOssCredentialStore(db)
	ossPolicyRepo := gormrepo.NewOssPolicyStore(db)
	ossAuditRepo := gormrepo.NewOssAuditStore(db)
	aiSettingRepo := gormrepo.NewAIAgentSettingStore(db)
	aiAuditRepo := gormrepo.NewAIAgentSettingAuditStore(db)
	aiUsageLogRepo := gormrepo.NewAIUsageLogStore(db)
	systemSwitchRepo := gormrepo.NewSystemSwitchStore(db)
	systemParameterRepo := gormrepo.NewSystemParameterStore(db)
	systemBroadcastRepo := gormrepo.NewSystemBroadcastStore(db)
	systemAuditRepo := gormrepo.NewSystemAuditStore(db)
	passwordResetRepo := gormrepo.NewPasswordResetTokenStore(db)
	schoolRepo := gormrepo.NewSchoolStore(db)
	timeSlotRepo := gormrepo.NewTimeSlotRepository(db)
	courseStudentRepo := gormrepo.NewCourseStudentStore(db)
	courseTeacherRepo := gormrepo.NewCourseTeacherStore(db)
	courseScheduleRepo := gormrepo.NewCourseScheduleStore(db)
	classroomRepo := gormrepo.NewClassroomRepository(db)
	notificationRepo := gormrepo.NewNotificationStore(db)
	resourceRepo := gormrepo.NewResourceStore(db)
	resourceFileRepo := gormrepo.NewResourceFileStore(db)
	resourceFavoriteRepo := gormrepo.NewResourceFavoriteStore(db)

	fileRepo := gormrepo.NewFileStore(db)
	attachmentRepo := gormrepo.NewAttachmentStore(db)

	passwordHasher := security.NewBcryptHasher()
	tokenIssuer := security.NewJWTIssuer(cfg.JWTSecret, cfg.RefreshSecret, cfg.TokenTTL, cfg.RefreshTokenTTL)

	authUsecase := usecase.NewAuthUsecase(accountRepo, passwordResetRepo, tokenIssuer, passwordHasher, time.Duration(cfg.PasswordResetTokenTTL)*time.Second)
	aiModel := usecase.NewOpenAIChatModel()
	streamHub := imstream.NewHub()
	notificationHub := realtime.NewNotificationHub()
	notificationUsecase := usecase.NewNotificationUsecase(notificationRepo, notificationHub)
	fileUsecase := usecase.NewFileUsecase(fileRepo, attachmentRepo, ossCredentialRepo, objectstorage.NewAliyunFactory())
	assignmentUsecase := usecase.NewAssignmentUsecase(assignmentRepo, submissionRepo, submissionCommentRepo, studentRepo, notificationUsecase, fileUsecase)

	adminUsecase := usecase.NewAdminUsecase(accountRepo, teacherRepo, studentRepo, departmentRepo, classRepo, courseScheduleRepo, teacherStudentRepo, aiModel, aiSettingRepo, passwordHasher)
	teacherPortalUsecase := usecase.NewTeacherPortalUsecase(teacherRepo, assignmentRepo, submissionRepo, studentRepo, courseSessionRepo, courseRepo, classRepo, courseSlotRepo, accountRepo, courseScheduleRepo, courseChapterRepo, courseStudentRepo, courseTeacherRepo)
	studentPortalUsecase := usecase.NewStudentPortalUsecase(studentRepo, assignmentRepo, submissionRepo, courseRepo, courseChapterRepo, courseSlotRepo, courseSessionRepo, teacherRepo, accountRepo, studentReminderRepo)
	conversationUsecase := usecase.NewConversationUsecase(conversationRepo, messageRepo, receiptRepo, accountRepo)
	noteUsecase := usecase.NewNoteUsecase(noteRepo, accountRepo)
	noteCommentUsecase := usecase.NewNoteCommentUsecase(noteRepo, noteCommentRepo, accountRepo)
	ossUsecase := usecase.NewAdminOssUsecase(ossCredentialRepo, ossPolicyRepo, ossAuditRepo, accountRepo)
	systemUsecase := usecase.NewAdminSystemUsecase(systemSwitchRepo, systemParameterRepo, systemBroadcastRepo, systemAuditRepo)
	aiSettingsUsecase := usecase.NewAISettingsUsecase(aiSettingRepo, aiAuditRepo, accountRepo)
	aiGradingUsecase := usecase.NewAIGradingUsecase(aiSettingRepo, aiUsageLogRepo, aiModel)

	schoolUsecase := usecase.NewSchoolUsecase(schoolRepo, timeSlotRepo)
	courseUsecase := usecase.NewCourseUsecase(courseRepo, courseStudentRepo, courseTeacherRepo, studentRepo, teacherRepo)
	scheduleUsecase := usecase.NewScheduleUsecase(timeSlotRepo, courseScheduleRepo, courseSessionRepo, courseRepo, teacherRepo, classroomRepo)
	classroomUsecase := usecase.NewClassroomUsecase(classroomRepo)
	resourceUsecase := usecase.NewResourceUsecase(resourceRepo, resourceFileRepo, resourceFavoriteRepo, teacherRepo, fileUsecase, departmentRepo, fileRepo)

	httpHandler := apihandlers.NewHandler(cfg.JWTSecret, authUsecase, adminUsecase, assignmentUsecase, teacherPortalUsecase, studentPortalUsecase, conversationUsecase, noteUsecase, noteCommentUsecase, ossUsecase, systemUsecase, aiSettingsUsecase, aiGradingUsecase, schoolUsecase, courseUsecase, scheduleUsecase, classroomUsecase, fileUsecase, notificationUsecase, streamHub, resourceUsecase)

	httpServer := server.NewHTTPServer(cfg, httpHandler, log)
	grpcServer := server.NewGRPCServer(
		cfg,
		grpcserver.NewConversationServer(conversationUsecase, streamHub),
		grpcserver.NewNotificationServer(notificationHub),
		log,
	)

	return &Application{db: db, log: log, http: httpServer, grpc: grpcServer}, nil
}

// Run 在后台启动 gRPC 服务，并阻塞运行 HTTP 服务；HTTP 服务退出时一并关闭 gRPC 服务。
func (a *Application) Run() error {
	if err := a.grpc.Start(); err != nil {
		return err
	}
	defer a.grpc.Stop()

	return a.http.Start()
}

func migrate(db *gorm.DB) error {
	// 旧版 AI 聊天相关的表已从产品中移除。
	// 若仍存在则删除（由于外键依赖，先删除 messages 表）。
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
		&organization.School{},
		&identity.Account{},
		&roster.Teacher{},
		&roster.Student{},
		&roster.StudentReminder{},
		&roster.TeacherStudentLink{},
		&organization.Department{},
		&organization.Class{},
		&curriculum.Course{},
		&curriculum.CourseChapter{},
		&curriculum.CourseChapterAttachment{},
		&curriculum.CourseStudent{},
		&curriculum.CourseTeacher{},
		&scheduling.CourseSlot{},
		&scheduling.CourseSchedule{},
		&scheduling.CourseSession{},
		&coursework.Assignment{},
		&coursework.AssignmentQuestion{},
		&coursework.AssignmentSubmission{},
		&coursework.SubmissionItem{},
		&coursework.SubmissionComment{},
		&messaging.Conversation{},
		&messaging.ConversationMember{},
		&messaging.Message{},
		&messaging.MessageReceipt{},
		&notebook.Note{},
		&notebook.NoteComment{},
		&storage.OssCredential{},
		&storage.OssPolicy{},
		&storage.OssAuditLog{},
		&system.SystemSwitch{},
		&system.SystemParameter{},
		&system.SystemBroadcast{},
		&system.SystemAuditLog{},
		&aiagent.AIAgentSetting{},
		&aiagent.AIAgentSettingAudit{},
		&aiagent.AIUsageLog{},
		&identity.PasswordResetToken{},
		&scheduling.TimeSlot{},
		&organization.Classroom{},
		&storage.File{},
		&coursework.AssignmentAttachment{},
		&coursework.SubmissionAttachment{},
		&notification.Notification{},
		&library.Resource{},
		&library.ResourceFile{},
		&library.ResourceFavorite{},
	); err != nil {
		return err
	}

	if err := ensureAccountStatusIntegrity(db); err != nil {
		return err
	}

	return ensurePartialUniqueIndexes(db)
}

func ensureAccountStatusIntegrity(db *gorm.DB) error {
	dialect := db.Dialector.Name()

	// 在施加约束前，先规范化历史遗留或非法的值。
	if err := db.Exec(
		"UPDATE accounts SET status = 'active' WHERE status IS NULL OR TRIM(status) = '' OR status NOT IN ('active','locked','password_reset_required')",
	).Error; err != nil {
		return err
	}

	switch dialect {
	case "postgres":
		if err := db.Exec("ALTER TABLE accounts ALTER COLUMN status SET DEFAULT 'active'").Error; err != nil {
			return err
		}
		if err := db.Exec("ALTER TABLE accounts ALTER COLUMN status SET NOT NULL").Error; err != nil {
			return err
		}
		if err := db.Exec("ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_status_check").Error; err != nil {
			return err
		}
		if err := db.Exec("ALTER TABLE accounts ADD CONSTRAINT accounts_status_check CHECK (status IN ('active','locked','password_reset_required'))").Error; err != nil {
			return err
		}
	}

	return nil
}

func ensurePartialUniqueIndexes(db *gorm.DB) error {
	// GORM 软删除会保留设置了 deleted_at 的行，但 identifier 上的普通 UNIQUE 索引
	// 仍会阻止重新注册。因此将自动创建的索引替换为
	// 仅作用于未删除行的部分唯一索引。
	//
	// Postgres 和较新的 SQLite 都支持部分索引。
	dialect := db.Dialector.Name()
	switch dialect {
	case "postgres", "sqlite":
		// 删除由 gorm:"uniqueIndex" 创建的旧唯一索引。
		if err := db.Exec("DROP INDEX IF EXISTS idx_accounts_identifier").Error; err != nil {
			return err
		}
		return db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_identifier ON accounts (identifier) WHERE deleted_at IS NULL").Error
	default:
		return nil
	}
}

func resolveDialector(cfg config.AppConfig) (gorm.Dialector, error) {
	driver := strings.ToLower(cfg.DatabaseDriver)
	switch driver {
	case "postgres", "postgresql":
		return postgres.Open(cfg.DatabaseDSN), nil
	case "sqlite", "sqlite3", "":
		return sqlite.Open(cfg.DatabaseDSN), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DatabaseDriver)
	}
}
