package response

import (
	"net/http"
	"testing"
)

func TestLocalizeMessage_MappedAndAlias(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		want    string
	}{
		{
			name:    "direct mapped key",
			status:  http.StatusUnauthorized,
			message: "invalid token",
			want:    "登录状态已失效，请重新登录后重试。",
		},
		{
			name:    "snake case alias",
			status:  http.StatusBadRequest,
			message: "school_id_required",
			want:    "缺少学校标识，请刷新页面后重试。",
		},
		{
			name:    "underscore alias",
			status:  http.StatusBadRequest,
			message: "invalid_request",
			want:    "请求参数不正确，请检查后重试。",
		},
		{
			name:    "space canonicalization",
			status:  http.StatusBadRequest,
			message: "  FILE_NAME   TOO_LONG  ",
			want:    "文件名过长，请缩短后重试。",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := localizeMessage(tc.status, tc.message)
			if got != tc.want {
				t.Fatalf("localizeMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLocalizeMessage_RulesAndFallback(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		want    string
	}{
		{
			name:    "not found rule",
			status:  http.StatusNotFound,
			message: "course section not found",
			want:    "未找到对应数据，请刷新后重试。",
		},
		{
			name:    "required rule",
			status:  http.StatusBadRequest,
			message: "xyz required",
			want:    "缺少必要参数，请检查后重试。",
		},
		{
			name:    "failed to prefix",
			status:  http.StatusInternalServerError,
			message: "failed_to_sync_remote",
			want:    "操作失败，请稍后重试。",
		},
		{
			name:    "status fallback",
			status:  http.StatusConflict,
			message: "totally unknown message",
			want:    "当前操作发生冲突，请刷新页面后重试。",
		},
		{
			name:    "keep chinese message",
			status:  http.StatusBadRequest,
			message: "参数非法，请重试",
			want:    "参数非法，请重试",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := localizeMessage(tc.status, tc.message)
			if got != tc.want {
				t.Fatalf("localizeMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeMessage(t *testing.T) {
	got := normalizeMessage("  INVALID_REQUEST-BODY  ")
	want := "invalid request body"
	if got != want {
		t.Fatalf("normalizeMessage() = %q, want %q", got, want)
	}
}
