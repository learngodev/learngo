package app

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
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

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
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

	authService := service.NewAuthService(accountRepo, passwordResetRepo, cfg)
	aiModel := service.NewOpenAIChatModel()
	streamHub := realtime.NewHub()
	notificationHub := realtime.NewNotificationHub()
	notificationService := service.NewNotificationService(notificationRepo, notificationHub)
	fileService := service.NewFileService(db, ossCredentialRepo)
	assignmentService := service.NewAssignmentService(assignmentRepo, submissionRepo, submissionCommentRepo, studentRepo, notificationService, fileService)

	adminService := service.NewAdminService(accountRepo, teacherRepo, studentRepo, departmentRepo, classRepo, teacherStudentRepo, aiModel, aiSettingRepo)
	teacherPortalService := service.NewTeacherPortalService(teacherRepo, assignmentRepo, submissionRepo, studentRepo, courseSessionRepo, courseRepo, classRepo, courseSlotRepo, accountRepo, courseScheduleRepo)
	studentPortalService := service.NewStudentPortalService(studentRepo, assignmentRepo, submissionRepo, courseRepo, courseSlotRepo, courseSessionRepo, teacherRepo, accountRepo, studentReminderRepo)
	conversationService := service.NewConversationService(conversationRepo, messageRepo, receiptRepo, accountRepo)
	noteService := service.NewNoteService(noteRepo, accountRepo)
	noteCommentService := service.NewNoteCommentService(noteRepo, noteCommentRepo, accountRepo)
	ossService := service.NewAdminOssService(ossCredentialRepo, ossPolicyRepo, ossAuditRepo, accountRepo)
	systemService := service.NewAdminSystemService(systemSwitchRepo, systemParameterRepo, systemBroadcastRepo, systemAuditRepo)
	aiSettingsService := service.NewAISettingsService(aiSettingRepo, aiAuditRepo, accountRepo)
	aiGradingService := service.NewAIGradingService(aiSettingRepo, aiUsageLogRepo, aiModel)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	engine.Use(middleware.CORS())

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpcserver.StreamAuthInterceptor(cfg.JWTSecret, []string{string(domain.RoleStudent), string(domain.RoleTeacher), string(domain.RoleAdmin)})),
	)
	apigrpc.RegisterConversationServiceServer(grpcServer, grpcserver.NewConversationServer(conversationService, streamHub))
	apigrpc.RegisterNotificationServiceServer(grpcServer, grpcserver.NewNotificationServer(notificationHub))

	schoolService := service.NewSchoolService(schoolRepo, timeSlotRepo)
	courseService := service.NewCourseService(courseRepo, courseStudentRepo, courseTeacherRepo, studentRepo, teacherRepo)
	scheduleService := service.NewScheduleService(timeSlotRepo, courseScheduleRepo, courseSessionRepo, courseRepo, teacherRepo, classroomRepo)
	classroomService := service.NewClassroomService(classroomRepo)

	handler := apihandlers.NewHandler(authService, adminService, assignmentService, teacherPortalService, studentPortalService, conversationService, noteService, noteCommentService, ossService, systemService, aiSettingsService, aiGradingService, schoolService, courseService, scheduleService, classroomService, fileService, notificationService, streamHub)

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

	// gRPC-Web (for Flutter Web)
	grpcWebAddr := fmt.Sprintf(":%s", a.cfg.GRPCWebPort)
	wrapped := grpcweb.WrapServer(
		a.grpc,
		grpcweb.WithOriginFunc(func(origin string) bool {
			// Web 端使用 Bearer Token，不依赖 cookie；这里放开跨域由上层网关/部署策略约束。
			return true
		}),
	)

	grpcWebHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wrapped.IsGrpcWebRequest(r) || wrapped.IsAcceptableGrpcCorsRequest(r) || wrapped.IsGrpcWebSocketRequest(r) {
			wrapped.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	grpcWebServer := &http.Server{Addr: grpcWebAddr, Handler: cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
			"Authorization",
			"X-Grpc-Web",
			"X-User-Agent",
			"Grpc-Timeout",
		},
		ExposedHeaders: []string{"Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin"},
	}).Handler(grpcWebHandler)}

	go func() {
		a.log.Printf("starting grpc-web server on %s", grpcWebAddr)
		if err := grpcWebServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Printf("grpc-web server stopped: %v", err)
		}
	}()
	defer func() {
		a.grpc.GracefulStop()
		_ = lis.Close()
		_ = grpcWebServer.Close()
	}()

	address := fmt.Sprintf(":%s", a.cfg.HTTPPort)
	a.log.Printf("starting http server on %s", address)
	return a.engine.Run(address)
}

func migrate(db *gorm.DB) error {
	// Legacy AI chat tables are removed from the product.
	// Drop them if they still exist (messages first due to FK dependencies).
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
		&domain.School{},
		&domain.Account{},
		&domain.Teacher{},
		&domain.Student{},
		&domain.StudentReminder{},
		&domain.TeacherStudentLink{},
		&domain.Department{},
		&domain.Class{},
		&domain.ClassTeacher{},
		&domain.Course{},
		&domain.CourseStudent{},
		&domain.CourseTeacher{},
		&domain.CourseSlot{},
		&domain.CourseSchedule{},
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
		&domain.AIUsageLog{},
		&domain.PasswordResetToken{},
		&domain.TimeSlot{},
		&domain.Classroom{},
		&domain.File{},
		&domain.AssignmentAttachment{},
		&domain.SubmissionAttachment{},
	); err != nil {
		return err
	}

	return ensurePartialUniqueIndexes(db)
}

func ensurePartialUniqueIndexes(db *gorm.DB) error {
	// GORM soft-delete keeps rows with deleted_at set, but a plain UNIQUE index on
	// identifier will still block re-registration. Replace the auto-created index
	// with a partial unique index that applies only to active rows.
	//
	// Postgres and modern SQLite both support partial indexes.
	dialect := db.Dialector.Name()
	switch dialect {
	case "postgres", "sqlite":
		// Drop the old unique index created by gorm:"uniqueIndex".
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
