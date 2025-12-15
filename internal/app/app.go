package app

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	apigrpc "learn-go/internal/api/grpcpb"
	"learn-go/internal/api/grpcserver"
	apihandlers "learn-go/internal/api/http"
	"learn-go/internal/config"
	"learn-go/internal/domain"
	"learn-go/internal/realtime"
	gormrepo "learn-go/internal/repository/gorm"
	"learn-go/internal/service"
	"learn-go/pkg/logger"
	"learn-go/pkg/middleware"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Application wires up services and transports.
type Application struct {
	cfg    config.AppConfig
	engine *gin.Engine
	db     *gorm.DB
	log    *logger.Logger
	grpc   *grpc.Server
}

// New creates the application, preparing dependencies.
func New() (*Application, error) {
	cfg := config.Load()

	log := logger.New()

	gin.SetMode(gin.ReleaseMode)
	if cfg.Environment == "local" || cfg.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	}

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
	teachingAssignmentRepo := gormrepo.NewTeachingAssignmentStore(db)
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
	aiSessionRepo := gormrepo.NewAIChatSessionStore(db)
	aiMessageRepo := gormrepo.NewAIChatMessageStore(db)
	systemSwitchRepo := gormrepo.NewSystemSwitchStore(db)
	systemParameterRepo := gormrepo.NewSystemParameterStore(db)
	systemBroadcastRepo := gormrepo.NewSystemBroadcastStore(db)
	systemAuditRepo := gormrepo.NewSystemAuditStore(db)
	passwordResetRepo := gormrepo.NewPasswordResetTokenStore(db)
	schoolRepo := gormrepo.NewSchoolStore(db)
	timeSlotRepo := gormrepo.NewTimeSlotRepository(db)

	authService := service.NewAuthService(accountRepo, passwordResetRepo, cfg)
	adminService := service.NewAdminService(accountRepo, teacherRepo, studentRepo, departmentRepo, classRepo, teacherStudentRepo)
	assignmentService := service.NewAssignmentService(assignmentRepo, submissionRepo, submissionCommentRepo)
	teacherPortalService := service.NewTeacherPortalService(teacherRepo, assignmentRepo, submissionRepo, studentRepo, courseSessionRepo, courseRepo, classRepo, courseSlotRepo, accountRepo)
	studentPortalService := service.NewStudentPortalService(studentRepo, assignmentRepo, submissionRepo, courseRepo, courseSlotRepo, courseSessionRepo, teacherRepo, accountRepo, studentReminderRepo)
	conversationService := service.NewConversationService(conversationRepo, messageRepo, receiptRepo, accountRepo)
	noteService := service.NewNoteService(noteRepo, accountRepo)
	noteCommentService := service.NewNoteCommentService(noteRepo, noteCommentRepo, accountRepo)
	ossService := service.NewAdminOssService(ossCredentialRepo, ossPolicyRepo, ossAuditRepo, accountRepo)
	systemService := service.NewAdminSystemService(systemSwitchRepo, systemParameterRepo, systemBroadcastRepo, systemAuditRepo)
	aiModel := service.NewOpenAIChatModel()
	aiService := service.NewAIAssistantService(aiSettingRepo, aiAuditRepo, aiSessionRepo, aiMessageRepo, accountRepo, aiModel, studentPortalService, teacherPortalService, assignmentService)
	aiGradingService := service.NewAIGradingService(aiSettingRepo, aiModel)
	streamHub := realtime.NewHub()

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpcserver.StreamAuthInterceptor(cfg.JWTSecret, []string{string(domain.RoleStudent), string(domain.RoleTeacher), string(domain.RoleAdmin)})),
	)
	apigrpc.RegisterConversationServiceServer(grpcServer, grpcserver.NewConversationServer(conversationService, streamHub))

	schoolService := service.NewSchoolService(schoolRepo, timeSlotRepo)
	courseService := service.NewCourseService(courseRepo, teachingAssignmentRepo)

	handler := apihandlers.NewHandler(authService, adminService, assignmentService, teacherPortalService, studentPortalService, conversationService, noteService, noteCommentService, ossService, systemService, aiService, aiGradingService, schoolService, courseService, streamHub)

	adminGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(domain.RoleAdmin)}})
	teacherGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(domain.RoleTeacher), string(domain.RoleAdmin)}})
	studentGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(domain.RoleStudent), string(domain.RoleTeacher), string(domain.RoleAdmin)}})

	handler.RegisterRoutes(engine, adminGuard, teacherGuard, studentGuard)

	return &Application{cfg: cfg, engine: engine, db: db, log: log, grpc: grpcServer}, nil
}

// Run starts the HTTP server.
func (a *Application) Run() error {
	grpcAddr := fmt.Sprintf(":%s", a.cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	go func() {
		a.log.Printf("starting grpc server on %s", grpcAddr)
		if err := a.grpc.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			a.log.Printf("grpc server stopped: %v", err)
		}
	}()
	defer func() {
		a.grpc.GracefulStop()
		_ = lis.Close()
	}()

	address := fmt.Sprintf(":%s", a.cfg.HTTPPort)
	a.log.Printf("starting http server on %s", address)
	return a.engine.Run(address)
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.School{},
		&domain.Account{},
		&domain.Teacher{},
		&domain.Student{},
		&domain.StudentReminder{},
		&domain.TeacherStudentLink{},
		&domain.Department{},
		&domain.Class{},
		&domain.Course{},
		&domain.CourseSlot{},
		&domain.CourseSession{},
		&domain.Assignment{},
		&domain.AssignmentQuestion{},
		&domain.AssignmentSubmission{},
		&domain.SubmissionItem{},
		&domain.SubmissionComment{},
		&domain.Conversation{},
		&domain.ConversationMember{},
		&domain.Message{},
		&domain.MessageReceipt{},
		&domain.Note{},
		&domain.NoteComment{},
		&domain.OssCredential{},
		&domain.OssPolicy{},
		&domain.OssAuditLog{},
		&domain.SystemSwitch{},
		&domain.SystemParameter{},
		&domain.SystemBroadcast{},
		&domain.SystemAuditLog{},
		&domain.AIAgentSetting{},
		&domain.AIAgentSettingAudit{},
		&domain.AIChatSession{},
		&domain.AIChatMessage{},
		&domain.PasswordResetToken{},
		&domain.TimeSlot{},
	)
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
