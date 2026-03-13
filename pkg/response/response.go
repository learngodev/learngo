package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var localizedMessages = map[string]string{
	"missing authorization":                         "未检测到登录凭证，请先登录后重试。",
	"missing token":                                 "未检测到登录凭证，请先登录后重试。",
	"invalid token":                                 "登录状态已失效，请重新登录后重试。",
	"invalid refresh token":                         "登录状态已失效，请重新登录后重试。",
	"invalid token claims":                          "登录状态已失效，请重新登录后重试。",
	"invalid token subject":                         "登录状态已失效，请重新登录后重试。",
	"invalid reset token":                           "重置密码链接无效，请重新申请。",
	"reset token expired":                           "重置密码链接已过期，请重新申请。",
	"insufficient role":                             "当前账号没有权限执行此操作。",
	"forbidden":                                     "当前账号没有权限执行此操作。",
	"account locked":                                "账号已被锁定，请联系管理员。",
	"password reset required":                       "请先重置密码后再登录。",
	"password reset unavailable":                    "当前账号暂不支持重置密码，请联系管理员。",
	"invalid credentials":                           "账号或密码错误，请检查后重试。",
	"missing account context":                       "无法识别当前登录账号，请重新登录后重试。",
	"invalid account context":                       "无法识别当前登录账号，请重新登录后重试。",
	"invalid role":                                  "角色参数无效，请检查后重试。",
	"invalid request":                               "请求参数不正确，请检查后重试。",
	"invalid request body":                          "请求参数格式不正确，请检查后重试。",
	"invalid request payload":                       "请求参数格式不正确，请检查后重试。",
	"validation failed":                             "提交的数据不符合要求，请检查后重试。",
	"validation error":                              "提交的数据不符合要求，请检查后重试。",
	"school id required":                            "缺少学校标识，请刷新页面后重试。",
	"invalid school id":                             "学校标识无效，请检查后重试。",
	"id required":                                   "缺少必要标识，请检查后重试。",
	"account id is required":                        "缺少账号标识，请检查后重试。",
	"class id is required":                          "缺少班级标识，请检查后重试。",
	"department id is required":                     "缺少部门标识，请检查后重试。",
	"credential id is required":                     "缺少凭证标识，请检查后重试。",
	"policy id is required":                         "缺少策略标识，请检查后重试。",
	"file id required":                              "缺少文件标识，请检查后重试。",
	"file is required":                              "请先选择要上传的文件。",
	"invalid file size":                             "文件大小不合法，请检查后重试。",
	"file name is required":                         "缺少文件名，请检查后重试。",
	"file name too long":                            "文件名过长，请缩短后重试。",
	"file type is required":                         "缺少文件类型，请检查后重试。",
	"file type too long":                            "文件类型标识过长，请检查后重试。",
	"file name mismatch":                            "文件名与上传内容不一致，请检查后重试。",
	"file type mismatch":                            "文件类型与上传内容不一致，请检查后重试。",
	"file size mismatch":                            "文件大小与上传内容不一致，请检查后重试。",
	"file not found":                                "未找到对应文件，请刷新后重试。",
	"time slot not found":                           "未找到对应的时间段，请确认后重试。",
	"student profile not found":                     "未找到学生档案，请联系管理员确认账号信息。",
	"teacher profile not found":                     "未找到教师档案，请联系管理员确认账号信息。",
	"teacher profile required":                      "未找到教师档案，请联系管理员确认账号信息。",
	"student profile required":                      "未找到学生档案，请联系管理员确认账号信息。",
	"course access denied":                          "你没有该课程的访问权限。",
	"course not found":                              "未找到对应课程，请刷新后重试。",
	"course not owned":                              "你没有该课程的管理权限。",
	"chapter not found":                             "未找到对应章节，请刷新后重试。",
	"chapter or file not found":                     "未找到对应章节或文件，请刷新后重试。",
	"missing course id":                             "缺少课程标识，请检查后重试。",
	"missing course/chapter id":                     "缺少课程或章节标识，请检查后重试。",
	"missing course/chapter/file id":                "缺少课程、章节或文件标识，请检查后重试。",
	"missing assignment id":                         "缺少作业标识，请检查后重试。",
	"missing assignment target":                     "缺少作业分配目标，请检查后重试。",
	"missing assignment or submission id":           "缺少作业或提交记录标识，请检查后重试。",
	"assignment not found":                          "未找到对应作业，请刷新后重试。",
	"assignment not accessible":                     "你没有该作业的访问权限。",
	"submission not found":                          "未找到提交记录，请刷新后重试。",
	"submission forbidden":                          "你没有该提交记录的访问权限。",
	"submission already graded":                     "该提交已评分，不能重复评分。",
	"note not found":                                "未找到对应笔记，请刷新后重试。",
	"missing note id":                               "缺少笔记标识，请检查后重试。",
	"not allowed to access note":                    "你没有该笔记的访问权限。",
	"not allowed to view comments":                  "你没有查看评论的权限。",
	"not allowed to comment":                        "你没有发表评论的权限。",
	"not allowed to send message":                   "你没有发送消息的权限。",
	"not allowed to view messages":                  "你没有查看消息的权限。",
	"not allowed to create conversation":            "你没有创建会话的权限。",
	"not allowed to mark read":                      "你没有操作消息已读状态的权限。",
	"conversation not found":                        "未找到对应会话，请刷新后重试。",
	"conversation or message not found":             "未找到对应会话或消息，请刷新后重试。",
	"missing conversation id":                       "缺少会话标识，请检查后重试。",
	"invalid conversation":                          "会话参数无效，请检查后重试。",
	"invalid reminder":                              "提醒参数无效，请检查后重试。",
	"missing reminder id":                           "缺少提醒标识，请检查后重试。",
	"reminder not found":                            "未找到对应提醒，请刷新后重试。",
	"reminders not found":                           "未找到对应提醒，请刷新后重试。",
	"schedule conflict":                             "课表时间冲突，请调整时间后重试。",
	"invalid schedule":                              "课表参数无效，请检查后重试。",
	"invalid time range":                            "时间范围无效，请检查后重试。",
	"invalid status":                                "状态参数无效，请检查后重试。",
	"unsupported action":                            "不支持的操作类型，请检查后重试。",
	"unsupported provider":                          "不支持的服务提供方，请检查后重试。",
	"invalid configuration":                         "配置参数无效，请检查后重试。",
	"department id required when class id provided": "指定班级时必须同时指定部门。",
	"department scope unassigned cannot be combined with department id": "部门范围与部门标识参数冲突，请检查后重试。",
	"class scope unassigned cannot be combined with class id":           "班级范围与班级标识参数冲突，请检查后重试。",
	"invalid department scope":                                          "部门范围参数无效，请检查后重试。",
	"invalid class scope":                                               "班级范围参数无效，请检查后重试。",
}

var localizedAliases = map[string]string{
	"school_id required":            "school id required",
	"school_id_required":            "school id required",
	"invalid_request":               "invalid request",
	"time_slot_not_found":           "time slot not found",
	"student_profile_not_found":     "student profile not found",
	"teacher_profile_not_found":     "teacher profile not found",
	"course_not_found":              "course not found",
	"course_not_owned":              "course not owned",
	"missing_assignment_target":     "missing assignment target",
	"file_name is required":         "file name is required",
	"file_name too long":            "file name too long",
	"file_type is required":         "file type is required",
	"file_type too long":            "file type too long",
	"file_name mismatch":            "file name mismatch",
	"file_type mismatch":            "file type mismatch",
	"file_size mismatch":            "file size mismatch",
	"invalid school_id":             "invalid school id",
	"account_ids required":          "account ids required",
	"failed to sign download token": "unable to sign download token",
}

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
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fallbackByStatus(status)
	}

	normalized := normalizeMessage(trimmed)
	if alias, ok := localizedAliases[normalized]; ok {
		normalized = alias
	}

	if value, ok := localizedMessages[normalized]; ok {
		return value
	}

	if byRule := localizeByRule(status, normalized); byRule != "" {
		return byRule
	}

	if strings.HasPrefix(normalized, "failed to") {
		return "操作失败，请稍后重试。"
	}
	if strings.HasPrefix(normalized, "unable to ") || strings.HasPrefix(normalized, "failed to ") {
		return "操作失败，请稍后重试。"
	}

	if containsChinese(trimmed) {
		return trimmed
	}

	return fallbackByStatus(status)
}

func localizeByRule(status int, normalized string) string {
	if strings.Contains(normalized, "token") {
		if strings.Contains(normalized, "expired") {
			return "登录状态已过期，请重新登录后重试。"
		}
		if strings.Contains(normalized, "invalid") {
			return "登录状态已失效，请重新登录后重试。"
		}
	}

	if strings.Contains(normalized, "not found") {
		if status == http.StatusForbidden {
			return "未找到对应账号档案，请联系管理员确认账号信息。"
		}
		return "未找到对应数据，请刷新后重试。"
	}

	if strings.Contains(normalized, "required") {
		return "缺少必要参数，请检查后重试。"
	}

	if strings.Contains(normalized, "invalid") {
		return "请求参数无效，请检查后重试。"
	}

	if strings.Contains(normalized, "forbidden") ||
		strings.Contains(normalized, "not allowed") {
		return "当前账号没有权限执行此操作。"
	}

	if strings.Contains(normalized, "conflict") {
		return "当前操作发生冲突，请刷新页面后重试。"
	}

	return ""
}

func normalizeMessage(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", " ", "-", " ")
	lower = replacer.Replace(lower)
	return strings.Join(strings.Fields(lower), " ")
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
