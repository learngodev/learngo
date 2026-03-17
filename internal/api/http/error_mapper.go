package http

import (
	"errors"
	nethttp "net/http"

	"learn-go/internal/service"
	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type mappedError struct {
	status int
	code   string
}

func (h *Handler) respondServiceError(
	c *gin.Context,
	err error,
	fallbackStatus int,
	fallbackCode string,
) {
	if err == nil {
		return
	}

	var appErr *service.AppError
	if errors.As(err, &appErr) {
		response.ErrorWithCodeMessage(
			c,
			appErr.Status,
			appErr.Code,
			appErr.Message,
			appErr.Details,
		)
		return
	}

	if mapped, ok := mapServiceError(err); ok {
		response.ErrorWithCodeMessage(
			c,
			mapped.status,
			mapped.code,
			err.Error(),
			err.Error(),
		)
		return
	}

	response.ErrorWithCodeMessage(
		c,
		fallbackStatus,
		fallbackCode,
		err.Error(),
		err.Error(),
	)
}

func mapServiceError(err error) (mappedError, bool) {
	switch {
	case errors.Is(err, service.ErrAccountLocked):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeAccountLocked}, true
	case errors.Is(err, service.ErrPasswordResetRequired):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodePasswordResetRequired}, true
	case errors.Is(err, service.ErrInvalidCredentials):
		return mappedError{status: nethttp.StatusUnauthorized, code: response.CodeInvalidCredentials}, true
	case errors.Is(err, service.ErrInvalidRefreshToken):
		return mappedError{status: nethttp.StatusUnauthorized, code: response.CodeInvalidRefreshToken}, true
	case errors.Is(err, service.ErrPasswordResetUnavailable):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodePasswordResetUnavailable}, true
	case errors.Is(err, service.ErrPasswordResetTokenInvalid):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidResetToken}, true
	case errors.Is(err, service.ErrPasswordResetTokenExpired):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeResetTokenExpired}, true

	case errors.Is(err, service.ErrScheduleValidation):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidSchedule}, true
	case errors.Is(err, service.ErrScheduleConflict):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeScheduleConflict}, true

	case errors.Is(err, service.ErrAdminAccountNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeAdminAccountNotFound}, true
	case errors.Is(err, service.ErrAdminAccountRoleNotSupported):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeAdminAccountRoleNotSupported}, true
	case errors.Is(err, service.ErrAdminAccountAlreadyLocked):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeAdminAccountAlreadyLocked}, true
	case errors.Is(err, service.ErrAdminAccountNotLocked):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeAdminAccountNotLocked}, true
	case errors.Is(err, service.ErrAdminPasswordResetPending):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeAdminPasswordResetPending}, true
	case errors.Is(err, service.ErrAdminBatchAccountIDsRequired):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeAccountIDsRequired}, true
	case errors.Is(err, service.ErrAdminBatchActionUnsupported):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeUnsupportedAction}, true
	case errors.Is(err, service.ErrOssPrimaryCredentialDeletion):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeUnableToDeleteCredential}, true

	case errors.Is(err, service.ErrTeacherProfileNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeTeacherProfileNotFound}, true
	case errors.Is(err, service.ErrStudentProfileNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeStudentProfileNotFound}, true
	case errors.Is(err, service.ErrTeacherCourseAccessDenied),
		errors.Is(err, service.ErrCourseAccessDenied),
		errors.Is(err, service.ErrTeacherAssignmentForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeCourseAccessDenied}, true

	case errors.Is(err, service.ErrAssignmentNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeAssignmentNotFound}, true
	case errors.Is(err, service.ErrSubmissionNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSubmissionNotFound}, true
	case errors.Is(err, service.ErrSubmissionForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeSubmissionForbidden}, true
	case errors.Is(err, service.ErrSubmissionAlreadyGraded):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeSubmissionAlreadyGraded}, true
	case errors.Is(err, service.ErrScoreOutOfRange):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidScore}, true

	case errors.Is(err, service.ErrNoteNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeNoteNotFound}, true
	case errors.Is(err, service.ErrNoteForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeNotAllowedToAccessNote}, true
	case errors.Is(err, service.ErrNoteCommentNotAllowed):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeNotAllowedToComment}, true

	case errors.Is(err, service.ErrConversationNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeConversationNotFound}, true
	case errors.Is(err, service.ErrConversationForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeForbidden}, true
	case errors.Is(err, service.ErrConversationInvalid):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidConversation}, true

	case errors.Is(err, service.ErrStudentReminderNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeReminderNotFound}, true
	case errors.Is(err, service.ErrStudentReminderInvalid):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidReminder}, true

	case errors.Is(err, service.ErrSystemSwitchNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSystemSwitchNotFound}, true
	case errors.Is(err, service.ErrSystemParameterNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSystemParameterNotFound}, true
	case errors.Is(err, service.ErrSystemBroadcastNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSystemBroadcastNotFound}, true

	case errors.Is(err, gorm.ErrRecordNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeResourceNotFound}, true
	}

	return mappedError{}, false
}
