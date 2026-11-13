package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"learn-go/internal/config"
	"learn-go/internal/domain/identity"
	"learn-go/internal/handler"
	"learn-go/internal/route"
	"learn-go/pkg/logger"
	"learn-go/pkg/middleware"
)

// HTTPServer 承载 Gin 引擎，对外提供 REST API 与 WebSocket。
type HTTPServer struct {
	srv *http.Server
	log *logger.Logger
}

// NewHTTPServer 创建 Gin 引擎、挂载中间件与路由，并返回 HTTPServer。
func NewHTTPServer(cfg config.AppConfig, h *handler.Handler, log *logger.Logger) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	if cfg.Environment == "local" || cfg.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	engine.Use(middleware.CORS())

	adminGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(identity.RoleAdmin)}})
	teacherGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(identity.RoleTeacher), string(identity.RoleAdmin)}})
	studentGuard := middleware.JWTAuth(middleware.AuthConfig{Secret: cfg.JWTSecret, AllowedRoles: []string{string(identity.RoleStudent), string(identity.RoleTeacher), string(identity.RoleAdmin)}})

	route.Register(engine, h, adminGuard, teacherGuard, studentGuard)

	return &HTTPServer{
		srv: &http.Server{Addr: fmt.Sprintf(":%s", cfg.HTTPPort), Handler: engine.Handler()},
		log: log,
	}
}

// Start 阻塞运行 HTTP 服务，直到出错或被 Stop 关闭。
func (s *HTTPServer) Start() error {
	s.log.Printf("starting http server on %s", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// Stop 优雅关闭 HTTP 服务。
func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
