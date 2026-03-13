package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Success writes a JSON success response with payload.
func Success(ctx *gin.Context, status int, payload interface{}) {
	ctx.JSON(status, gin.H{"success": true, "data": payload})
}

// Error writes an error response with message and optional details.
func Error(ctx *gin.Context, status int, message string, details interface{}) {
	friendlyMessage := localizeMessage(status, message)
	friendlyDetails := sanitizeDetails(status, details)

	ctx.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"message": friendlyMessage,
			"details": friendlyDetails,
		},
	})
}

func localizeMessage(status int, message string) string {
	normalized := strings.TrimSpace(strings.ToLower(message))

	mapped := map[string]string{
		"missing authorization":          "未检测到登录凭证，请先登录后重试。",
		"invalid token":                  "登录状态已失效，请重新登录后重试。",
		"invalid token claims":           "登录状态已失效，请重新登录后重试。",
		"invalid token subject":          "登录状态已失效，请重新登录后重试。",
		"insufficient role":              "当前账号没有权限执行此操作。",
		"missing account context":        "无法识别当前登录账号，请重新登录后重试。",
		"invalid request":                "请求参数不正确，请检查后重试。",
		"invalid request body":           "请求参数格式不正确，请检查后重试。",
		"validation failed":              "提交的数据不符合要求，请检查后重试。",
		"validation error":               "提交的数据不符合要求，请检查后重试。",
		"school_id required":             "缺少学校标识，请刷新页面后重试。",
		"school_id_required":             "缺少学校标识，请刷新页面后重试。",
		"id required":                    "缺少必要标识，请检查后重试。",
		"time_slot_not_found":            "未找到对应的时间段，请确认后重试。",
		"student_profile_not_found":      "未找到学生档案，请联系管理员确认账号信息。",
		"teacher profile not found":      "未找到教师档案，请联系管理员确认账号信息。",
		"student profile not found":      "未找到学生档案，请联系管理员确认账号信息。",
		"course access denied":           "你没有该课程的访问权限。",
		"chapter not found":              "未找到对应章节，请刷新后重试。",
		"chapter or file not found":      "未找到对应章节或文件，请刷新后重试。",
		"missing course id":              "缺少课程标识，请检查后重试。",
		"missing course/chapter id":      "缺少课程或章节标识，请检查后重试。",
		"missing course/chapter/file id": "缺少课程、章节或文件标识，请检查后重试。",
		"no fields to update":            "未检测到可更新内容，请修改后再提交。",
		"schedule conflict":              "课表时间冲突，请调整时间后重试。",
		"invalid schedule":               "课表参数无效，请检查后重试。",
	}
	if value, ok := mapped[normalized]; ok {
		return value
	}

	if strings.HasPrefix(normalized, "failed_to_") {
		return "操作失败，请稍后重试。"
	}
	if strings.HasPrefix(normalized, "unable to ") || strings.HasPrefix(normalized, "failed to ") {
		return "操作失败，请稍后重试。"
	}

	if strings.TrimSpace(message) != "" && containsChinese(message) {
		return message
	}

	return fallbackByStatus(status)
}

func sanitizeDetails(status int, details interface{}) interface{} {
	if details == nil {
		return nil
	}

	if status >= http.StatusInternalServerError {
		// Do not expose internal implementation details to clients.
		return nil
	}

	text := strings.TrimSpace(fmt.Sprint(details))
	if text == "" || text == "<nil>" {
		return nil
	}

	if containsChinese(text) {
		return text
	}

	return nil
}

func fallbackByStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "请求参数有误，请检查后重试。"
	case http.StatusUnauthorized:
		return "登录状态已失效，请重新登录。"
	case http.StatusForbidden:
		return "当前账号没有权限执行此操作。"
	case http.StatusNotFound:
		return "未找到请求的数据，请刷新后重试。"
	case http.StatusConflict:
		return "当前操作发生冲突，请刷新页面后重试。"
	default:
		if status >= http.StatusInternalServerError {
			return "服务器开小差了，请稍后重试。"
		}
		return "请求失败，请稍后重试。"
	}
}

func containsChinese(value string) bool {
	for _, r := range value {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
