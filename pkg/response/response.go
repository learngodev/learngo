package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var localizedMessages = map[string]string{
	CodeMissingAuthorization:                    "未检测到登录凭证，请先登录后重试。",
	CodeMissingToken:                            "未检测到登录凭证，请先登录后重试。",
	CodeInvalidToken:                            "登录状态已失效，请重新登录后重试。",
	CodeInvalidRefreshToken:                     "登录状态已失效，请重新登录后重试。",
	CodeInvalidTokenClaims:                      "登录状态已失效，请重新登录后重试。",
	CodeInvalidTokenSubject:                     "登录状态已失效，请重新登录后重试。",
	CodeInvalidResetToken:                       "重置密码链接无效，请重新申请。",
	CodeResetTokenExpired:                       "重置密码链接已过期，请重新申请。",
	CodeInsufficientRole:                        "当前账号没有权限执行此操作。",
	CodeForbidden:                               "当前账号没有权限执行此操作。",
	CodeAccountLocked:                           "账号已被锁定，请联系管理员。",
	CodePasswordResetRequired:                   "请先重置密码后再登录。",
	CodePasswordResetUnavailable:                "当前账号暂不支持重置密码，请联系管理员。",
	CodeInvalidCredentials:                      "账号或密码错误，请检查后重试。",
	CodeMissingAccountContext:                   "无法识别当前登录账号，请重新登录后重试。",
	CodeInvalidAccountContext:                   "无法识别当前登录账号，请重新登录后重试。",
	CodeInvalidRole:                             "角色参数无效，请检查后重试。",
	CodeInvalidRequest:                          "请求参数不正确，请检查后重试。",
	CodeInvalidRequestBody:                      "请求参数格式不正确，请检查后重试。",
	CodeInvalidRequestPayload:                   "请求参数格式不正确，请检查后重试。",
	CodeValidationFailed:                        "提交的数据不符合要求，请检查后重试。",
	CodeValidationError:                         "提交的数据不符合要求，请检查后重试。",
	CodeSchoolIDRequired:                        "缺少学校标识，请刷新页面后重试。",
	CodeInvalidSchoolID:                         "学校标识无效，请检查后重试。",
	CodeIDRequired:                              "缺少必要标识，请检查后重试。",
	CodeAccountIDRequired:                       "缺少账号标识，请检查后重试。",
	CodeAccountNotFound:                         "未找到对应账号，请联系管理员确认账号信息。",
	CodeClassIDRequired:                         "缺少班级标识，请检查后重试。",
	CodeDepartmentIDRequired:                    "缺少部门标识，请检查后重试。",
	CodeCredentialIDRequired:                    "缺少凭证标识，请检查后重试。",
	CodePolicyIDRequired:                        "缺少策略标识，请检查后重试。",
	CodeFileIDRequired:                          "缺少文件标识，请检查后重试。",
	CodeFileIsRequired:                          "请先选择要上传的文件。",
	CodeInvalidFileSize:                         "文件大小不合法，请检查后重试。",
	CodeFileNameRequired:                        "缺少文件名，请检查后重试。",
	CodeFileNameTooLong:                         "文件名过长，请缩短后重试。",
	CodeFileTypeRequired:                        "缺少文件类型，请检查后重试。",
	CodeFileTypeTooLong:                         "文件类型标识过长，请检查后重试。",
	CodeFileNameMismatch:                        "文件名与上传内容不一致，请检查后重试。",
	CodeFileTypeMismatch:                        "文件类型与上传内容不一致，请检查后重试。",
	CodeFileSizeMismatch:                        "文件大小与上传内容不一致，请检查后重试。",
	CodeFileNotFound:                            "未找到对应文件，请刷新后重试。",
	CodeTimeSlotNotFound:                        "未找到对应的时间段，请确认后重试。",
	CodeStudentProfileNotFound:                  "未找到学生档案，请联系管理员确认账号信息。",
	CodeTeacherProfileNotFound:                  "未找到教师档案，请联系管理员确认账号信息。",
	CodeTeacherProfileRequired:                  "未找到教师档案，请联系管理员确认账号信息。",
	CodeStudentProfileRequired:                  "未找到学生档案，请联系管理员确认账号信息。",
	CodeCourseAccessDenied:                      "你没有该课程的访问权限。",
	CodeCourseNotFound:                          "未找到对应课程，请刷新后重试。",
	CodeCourseNotOwned:                          "你没有该课程的管理权限。",
	CodeChapterNotFound:                         "未找到对应章节，请刷新后重试。",
	CodeChapterOrFileNotFound:                   "未找到对应章节或文件，请刷新后重试。",
	CodeMissingCourseID:                         "缺少课程标识，请检查后重试。",
	CodeMissingCourseChapterID:                  "缺少课程或章节标识，请检查后重试。",
	CodeMissingCourseChapterFileID:              "缺少课程、章节或文件标识，请检查后重试。",
	CodeMissingAssignmentID:                     "缺少作业标识，请检查后重试。",
	CodeMissingAssignmentTarget:                 "缺少作业分配目标，请检查后重试。",
	CodeMissingAssignmentOrSubmissionID:         "缺少作业或提交记录标识，请检查后重试。",
	CodeAssignmentNotFound:                      "未找到对应作业，请刷新后重试。",
	CodeAssignmentNotAccessible:                 "你没有该作业的访问权限。",
	CodeSubmissionNotFound:                      "未找到提交记录，请刷新后重试。",
	CodeSubmissionForbidden:                     "你没有该提交记录的访问权限。",
	CodeSubmissionAlreadyGraded:                 "该提交已评分，不能重复评分。",
	CodeNoteNotFound:                            "未找到对应笔记，请刷新后重试。",
	CodeMissingNoteID:                           "缺少笔记标识，请检查后重试。",
	CodeNotAllowedToAccessNote:                  "你没有该笔记的访问权限。",
	CodeNotAllowedToViewComments:                "你没有查看评论的权限。",
	CodeNotAllowedToComment:                     "你没有发表评论的权限。",
	CodeNotAllowedToSendMessage:                 "你没有发送消息的权限。",
	CodeNotAllowedToViewMessages:                "你没有查看消息的权限。",
	CodeNotAllowedToCreateConversation:          "你没有创建会话的权限。",
	CodeNotAllowedToMarkRead:                    "你没有操作消息已读状态的权限。",
	CodeConversationNotFound:                    "未找到对应会话，请刷新后重试。",
	CodeConversationOrMessageNotFound:           "未找到对应会话或消息，请刷新后重试。",
	CodeMissingConversationID:                   "缺少会话标识，请检查后重试。",
	CodeInvalidConversation:                     "会话参数无效，请检查后重试。",
	CodeInvalidReminder:                         "提醒参数无效，请检查后重试。",
	CodeMissingReminderID:                       "缺少提醒标识，请检查后重试。",
	CodeReminderNotFound:                        "未找到对应提醒，请刷新后重试。",
	CodeRemindersNotFound:                       "未找到对应提醒，请刷新后重试。",
	CodeScheduleConflict:                        "课表时间冲突，请调整时间后重试。",
	CodeInvalidSchedule:                         "课表参数无效，请检查后重试。",
	CodeInvalidTimeRange:                        "时间范围无效，请检查后重试。",
	CodeInvalidStatus:                           "状态参数无效，请检查后重试。",
	CodeUnsupportedAction:                       "不支持的操作类型，请检查后重试。",
	CodeUnsupportedProvider:                     "不支持的服务提供方，请检查后重试。",
	CodeInvalidConfiguration:                    "配置参数无效，请检查后重试。",
	CodeFailedToResolveSchoolContext:            "无法识别当前学校上下文，请刷新后重试。",
	CodeUnableToResolveTeacherProfile:           "无法识别教师档案，请联系管理员确认账号信息。",
	CodeUnableToExportGrades:                    "导出成绩失败，请稍后重试。",
	CodeSchoolIDIsRequired:                      "缺少学校标识，请刷新页面后重试。",
	CodeUnableToLoadFile:                        "加载文件信息失败，请稍后重试。",
	CodeUnableToRelayUploadExistingFile:         "续传文件失败，请稍后重试。",
	CodeUnableToRelayUpload:                     "上传文件失败，请稍后重试。",
	CodeUnableToInitializeUpload:                "初始化上传失败，请稍后重试。",
	CodeUnableToGetDownloadInfo:                 "获取下载信息失败，请稍后重试。",
	CodeUnableToOpenDownloadStream:              "下载文件失败，请稍后重试。",
	CodeUnableToSignDownloadToken:               "下载令牌生成失败，请稍后重试。",
	CodeAdminAccountNotFound:                    "未找到对应账号，请刷新后重试。",
	CodeAdminAccountRoleNotSupported:            "该账号角色不支持此操作。",
	CodeAdminAccountAlreadyLocked:               "账号已处于锁定状态。",
	CodeAdminAccountNotLocked:                   "账号当前未锁定。",
	CodeAdminPasswordResetPending:               "账号存在待处理的密码重置流程。",
	CodeAdminAccountOperationFailed:             "账号操作失败，请稍后重试。",
	CodeDepartmentIDRequiredWhenClassIDProvided: "指定班级时必须同时指定部门。",
	CodeDepartmentScopeUnassignedCannotCombineWithDepartmentID: "部门范围与部门标识参数冲突，请检查后重试。",
	CodeClassScopeUnassignedCannotCombineWithClassID:           "班级范围与班级标识参数冲突，请检查后重试。",
	CodeInvalidDepartmentScope:                                 "部门范围参数无效，请检查后重试。",
	CodeInvalidClassScope:                                      "班级范围参数无效，请检查后重试。",
}

// Success writes a JSON success response with payload.
func Success(ctx *gin.Context, status int, payload interface{}) {
	ctx.JSON(status, gin.H{"success": true, "data": payload})
}

// Error writes an error response with message and optional details.
func Error(ctx *gin.Context, status int, message string, details interface{}) {
	friendlyMessage := localizeMessage(status, message)
	_ = details

	ctx.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    normalizeCode(status, message),
			"message": friendlyMessage,
		},
	})
}

// ErrorWithCode writes an error response with explicit stable code.
func ErrorWithCode(ctx *gin.Context, status int, code string, details interface{}) {
	ErrorWithCodeMessage(ctx, status, code, "", details)
}

// ErrorWithCodeMessage writes an error response with explicit stable code and optional preferred message.
func ErrorWithCodeMessage(
	ctx *gin.Context,
	status int,
	code string,
	preferredMessage string,
	details interface{},
) {
	messageSource := strings.TrimSpace(preferredMessage)
	if messageSource == "" {
		messageSource = code
	}
	friendlyMessage := localizeMessage(status, messageSource)
	_ = details

	ctx.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    normalizeCode(status, code),
			"message": friendlyMessage,
		},
	})
}

func localizeMessage(status int, message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fallbackByStatus(status)
	}

	if value, ok := localizedMessages[trimmed]; ok {
		return value
	}

	if containsChinese(trimmed) {
		return trimmed
	}

	if status < http.StatusInternalServerError {
		// Keep explicit service/validation messages for 4xx responses.
		return trimmed
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

	if _, isString := details.(string); !isString {
		return details
	}

	text := strings.TrimSpace(fmt.Sprint(details))
	if text == "" || text == "<nil>" {
		return nil
	}
	return text
}

func normalizeCode(status int, code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return fallbackCodeByStatus(status)
	}
	if _, ok := localizedMessages[trimmed]; ok {
		return trimmed
	}
	return fallbackCodeByStatus(status)
}

func fallbackCodeByStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "resource not found"
	case http.StatusConflict:
		return "conflict"
	default:
		if status >= http.StatusInternalServerError {
			return "internal error"
		}
		return "request failed"
	}
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
