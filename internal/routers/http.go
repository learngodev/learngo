package routers

import (
	"github.com/gin-gonic/gin"

	"learn-go/internal/handler/http"
)

// RegisterHTTPRoutes centralizes HTTP route aggregation.
func RegisterHTTPRoutes(
	engine *gin.Engine,
	handler *http.Handler,
	adminGuard gin.HandlerFunc,
	teacherGuard gin.HandlerFunc,
	studentGuard gin.HandlerFunc,
) {
	handler.RegisterRoutes(engine, adminGuard, teacherGuard, studentGuard)
}
