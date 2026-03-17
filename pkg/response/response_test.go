package response

import (
	"net/http"
	"testing"
)

func TestLocalizeMessage_MappedAndFallback(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		want    string
	}{
		{
			name:    "direct mapped key",
			status:  http.StatusUnauthorized,
			message: CodeInvalidToken,
			want:    "登录状态已失效，请重新登录后重试。",
		},
		{
			name:    "unknown key keeps explicit 4xx message",
			status:  http.StatusBadRequest,
			message: "school_id_required",
			want:    "school_id_required",
		},
		{
			name:    "unknown key fallback for 5xx",
			status:  http.StatusInternalServerError,
			message: "db connection timeout",
			want:    "服务器开小差了，请稍后重试。",
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
