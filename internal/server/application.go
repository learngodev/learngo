package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	apigrpc "learn-go/api/grpcpb"
	sharedbiz "learn-go/internal/biz/shared"
	"learn-go/internal/config"
	gormrepo "learn-go/internal/data/gorm"
	grpcserver "learn-go/internal/handler/grpc"
	apihandlers "learn-go/internal/handler/http"
	gorminfra "learn-go/internal/infra/gorm"
	"learn-go/internal/realtime"
	"learn-go/internal/routers"
	"learn-go/internal/usecase"
	"learn-go/pkg/logger"
	"learn-go/pkg/middleware"
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

	db, err := gorminfra.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := gorminfra.AutoMigrate(db); err != nil {
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

	authService := service.NewAuthService(accountRepo, passwordResetRepo, cfg)
	aiModel := service.NewOpenAIChatModel()
	streamHub := realtime.NewHub()
	notificationHub := realtime.NewNotificationHub()
	notificationService := service.NewNotificationService(notificationRepo, notificationHub)
	fileService := service.NewFileService(db, ossCredentialRepo)
	assignmentService := service.NewAssignmentService(assignmentRepo, submissionRepo, submissionCommentRepo, studentRepo, notificationService, fileService)

	adminService := service.NewAdminService(accountRepo, teacherRepo, studentRepo, departmentRepo, classRepo, courseScheduleRepo, teacherStudentRepo, aiModel, aiSettingRepo)
	teacherPortalService := service.NewTeacherPortalService(teacherRepo, assignmentRepo, submissionRepo, studentRepo, courseSessionRepo, courseRepo, classRepo, courseSlotRepo, accountRepo, courseScheduleRepo, courseChapterRepo, courseStudentRepo, courseTeacherRepo)
	studentPortalService := service.NewStudentPortalService(studentRepo, assignmentRepo, submissionRepo, courseRepo, courseChapterRepo, courseSlotRepo, courseSessionRepo, teacherRepo, accountRepo, studentReminderRepo)
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
		grpc.StreamInterceptor(grpcserver.StreamAuthInterceptor(cfg.JWTSecret, []string{string(sharedbiz.RoleStudent), string(sharedbiz.RoleTeacher), string(sharedbiz.RoleAdmin)})),
	)
	apigrpc.RegisterConversationServiceServer(grpcServer, grpcserver.NewConversationServer(conversationService, streamHub))
	apigrpc.RegisterNotificationServiceServer(grpcServer, grpcserver.NewNotificationServer(notificationHub))

	schoolService := service.NewSchoolService(schoolRepo, timeSlotRepo)
	courseService := service.NewCourseService(courseRepo, courseStudentRepo, courseTeacherRepo, studentRepo, teacherRepo)
	scheduleService := service.NewScheduleService(timeSlotRepo, courseScheduleRepo, courseSessionRepo, courseRepo, teacherRepo, classroomRepo)
	classroomService := service.NewClassroomService(classroomRepo)

	handler := apihandlers.NewHandler(cfg.JWTSecret, authService, adminService, assignmentService, teacherPortalService, studentPortalService, conversationService, noteService, noteCommentService, ossService, systemService, aiSettingsService, aiGradingService, schoolService, courseService, scheduleService, classroomService, fileService, notificationService, streamHub)

	adminGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(sharedbiz.RoleAdmin)}})
	teacherGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(sharedbiz.RoleTeacher), string(sharedbiz.RoleAdmin)}})
	studentGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(sharedbiz.RoleStudent), string(sharedbiz.RoleTeacher), string(sharedbiz.RoleAdmin)}})

	routers.RegisterHTTPRoutes(engine, handler, adminGuard, teacherGuard, studentGuard)

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
