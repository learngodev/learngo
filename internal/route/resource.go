package route

import (
	"github.com/gin-gonic/gin"

	"learn-go/internal/handler"
)

// RegisterResourceRoutes 注册资源相关路由。
func RegisterResourceRoutes(api *gin.RouterGroup, h *handler.Handler, teacherGuard, studentGuard gin.HandlerFunc) {
	// Teacher routes
	teacher := api.Group("/teacher", teacherGuard)
	{
		teacher.POST("/resources", h.CreateTeacherResource)
		teacher.GET("/resources", h.ListTeacherResources)
		teacher.GET("/resources/:id", h.GetTeacherResource)
		teacher.PATCH("/resources/:id", h.UpdateTeacherResource)
		teacher.DELETE("/resources/:id", h.DeleteTeacherResource)
		teacher.POST("/resources/:id/files", h.AddFileToResource)
		teacher.DELETE("/resources/:id/files/:fileID", h.RemoveFileFromResource)
	}

	// Student routes (accessible by both students and teachers)
	student := api.Group("", studentGuard)
	{
		student.GET("/resources", h.BrowseResources)
		student.GET("/resources/:id", h.GetResourceDetail)
		student.POST("/resources/:id/favorite", h.ToggleFavorite)
		student.DELETE("/resources/:id/favorite", h.ToggleFavorite)
		student.GET("/resources/favorites", h.ListFavorites)
		student.GET("/resources/:id/files/:fileID/download", h.DownloadResourceFile)
	}
}
