package handler

import (
	"errors"
	"learn-go/internal/domain"
	nethttp "net/http"

	"learn-go/internal/usecase"
	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
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

	var appErr *usecase.AppError
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
	case errors.Is(err, usecase.ErrAccountLocked):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeAccountLocked}, true
	case errors.Is(err, usecase.ErrPasswordResetRequired):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodePasswordResetRequired}, true
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return mappedError{status: nethttp.StatusUnauthorized, code: response.CodeInvalidCredentials}, true
	case errors.Is(err, usecase.ErrInvalidRefreshToken):
		return mappedError{status: nethttp.StatusUnauthorized, code: response.CodeInvalidRefreshToken}, true
	case errors.Is(err, usecase.ErrPasswordResetUnavailable):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodePasswordResetUnavailable}, true
	case errors.Is(err, usecase.ErrPasswordResetTokenInvalid):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidResetToken}, true
	case errors.Is(err, usecase.ErrPasswordResetTokenExpired):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeResetTokenExpired}, true

	case errors.Is(err, usecase.ErrScheduleValidation):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidSchedule}, true
	case errors.Is(err, usecase.ErrScheduleConflict):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeScheduleConflict}, true

	case errors.Is(err, usecase.ErrAdminAccountNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeAdminAccountNotFound}, true
	case errors.Is(err, usecase.ErrAdminAccountRoleNotSupported):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeAdminAccountRoleNotSupported}, true
	case errors.Is(err, usecase.ErrAdminAccountAlreadyLocked):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeAdminAccountAlreadyLocked}, true
	case errors.Is(err, usecase.ErrAdminAccountNotLocked):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeAdminAccountNotLocked}, true
	case errors.Is(err, usecase.ErrAdminPasswordResetPending):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeAdminPasswordResetPending}, true
	case errors.Is(err, usecase.ErrAdminBatchAccountIDsRequired):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeAccountIDsRequired}, true
	case errors.Is(err, usecase.ErrAdminBatchActionUnsupported):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeUnsupportedAction}, true
	case errors.Is(err, usecase.ErrOssPrimaryCredentialDeletion):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeUnableToDeleteCredential}, true

	case errors.Is(err, usecase.ErrTeacherProfileNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeTeacherProfileNotFound}, true
	case errors.Is(err, usecase.ErrStudentProfileNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeStudentProfileNotFound}, true
	case errors.Is(err, usecase.ErrTeacherCourseAccessDenied),
		errors.Is(err, usecase.ErrCourseAccessDenied),
		errors.Is(err, usecase.ErrTeacherAssignmentForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeCourseAccessDenied}, true

	case errors.Is(err, usecase.ErrAssignmentNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeAssignmentNotFound}, true
	case errors.Is(err, usecase.ErrSubmissionNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSubmissionNotFound}, true
	case errors.Is(err, usecase.ErrSubmissionForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeSubmissionForbidden}, true
	case errors.Is(err, usecase.ErrSubmissionAlreadyGraded):
		return mappedError{status: nethttp.StatusConflict, code: response.CodeSubmissionAlreadyGraded}, true
	case errors.Is(err, usecase.ErrScoreOutOfRange):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidScore}, true

	case errors.Is(err, usecase.ErrNoteNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeNoteNotFound}, true
	case errors.Is(err, usecase.ErrNoteForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeNotAllowedToAccessNote}, true
	case errors.Is(err, usecase.ErrNoteCommentNotAllowed):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeNotAllowedToComment}, true

	case errors.Is(err, usecase.ErrConversationNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeConversationNotFound}, true
	case errors.Is(err, usecase.ErrConversationForbidden):
		return mappedError{status: nethttp.StatusForbidden, code: response.CodeForbidden}, true
	case errors.Is(err, usecase.ErrConversationInvalid):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidConversation}, true

	case errors.Is(err, usecase.ErrStudentReminderNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeReminderNotFound}, true
	case errors.Is(err, usecase.ErrStudentReminderInvalid):
		return mappedError{status: nethttp.StatusBadRequest, code: response.CodeInvalidReminder}, true

	case errors.Is(err, usecase.ErrSystemSwitchNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSystemSwitchNotFound}, true
	case errors.Is(err, usecase.ErrSystemParameterNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSystemParameterNotFound}, true
	case errors.Is(err, usecase.ErrSystemBroadcastNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeSystemBroadcastNotFound}, true

	case errors.Is(err, domain.ErrNotFound):
		return mappedError{status: nethttp.StatusNotFound, code: response.CodeResourceNotFound}, true
	}

	return mappedError{}, false
}
