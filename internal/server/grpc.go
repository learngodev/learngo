package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
	"google.golang.org/grpc"

	"learn-go/internal/config"
	"learn-go/internal/domain/identity"
	apigrpc "learn-go/internal/interface/grpcpb"
	"learn-go/internal/interface/grpcserver"
	"learn-go/pkg/logger"
)

// GRPCServer 承载原生 gRPC 服务，以及供 Flutter Web 使用的 gRPC-Web 网关。
type GRPCServer struct {
	grpc   *grpc.Server
	addr   string
	webSrv *http.Server
	log    *logger.Logger
}

// NewGRPCServer 创建 gRPC 服务并注册会话与通知服务。
func NewGRPCServer(
	cfg config.AppConfig,
	conversations apigrpc.ConversationServiceServer,
	notifications apigrpc.NotificationServiceServer,
	log *logger.Logger,
) *GRPCServer {
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpcserver.StreamAuthInterceptor(cfg.JWTSecret, []string{string(identity.RoleStudent), string(identity.RoleTeacher), string(identity.RoleAdmin)})),
	)
	apigrpc.RegisterConversationServiceServer(grpcServer, conversations)
	apigrpc.RegisterNotificationServiceServer(grpcServer, notifications)

	return &GRPCServer{
		grpc:   grpcServer,
		addr:   fmt.Sprintf(":%s", cfg.GRPCPort),
		webSrv: &http.Server{Addr: fmt.Sprintf(":%s", cfg.GRPCWebPort), Handler: newGRPCWebHandler(grpcServer)},
		log:    log,
	}
}

// Start 监听 gRPC 端口，并在后台启动 gRPC 与 gRPC-Web 服务；端口监听失败时返回错误。
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	go func() {
		s.log.Printf("starting grpc server on %s", s.addr)
		if err := s.grpc.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			s.log.Printf("grpc server stopped: %v", err)
		}
	}()

	go func() {
		s.log.Printf("starting grpc-web server on %s", s.webSrv.Addr)
		if err := s.webSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Printf("grpc-web server stopped: %v", err)
		}
	}()

	return nil
}

// Stop 优雅关闭 gRPC 服务并关闭 gRPC-Web 网关。
func (s *GRPCServer) Stop() {
	s.grpc.GracefulStop()
	_ = s.webSrv.Close()
}

func newGRPCWebHandler(grpcServer *grpc.Server) http.Handler {
	wrapped := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			// Web 端使用 Bearer Token，不依赖 cookie；这里放开跨域由上层网关/部署策略约束。
			return true
		}),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wrapped.IsGrpcWebRequest(r) || wrapped.IsAcceptableGrpcCorsRequest(r) || wrapped.IsGrpcWebSocketRequest(r) {
			wrapped.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return cors.New(cors.Options{
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
	}).Handler(handler)
}
