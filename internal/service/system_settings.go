package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// AdminSystemSwitch represents a feature toggle in the admin console.
type AdminSystemSwitch struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Enabled          bool     `json:"enabled"`
	LastUpdatedLabel string   `json:"last_updated_label"`
	Responsible      string   `json:"responsible"`
	IconName         string   `json:"icon"`
	Tags             []string `json:"tags"`
	Environment      string   `json:"environment"`
}

// AdminSystemParameter represents a configurable runtime parameter.
type AdminSystemParameter struct {
	ID               string `json:"id"`
	Key              string `json:"key"`
	Value            string `json:"value"`
	Scope            string `json:"scope"`
	Description      string `json:"description"`
	LastUpdatedLabel string `json:"last_updated_label"`
	Locked           bool   `json:"locked"`
}

// AdminSystemBroadcastStatus enumerates broadcast states.
type AdminSystemBroadcastStatus string

const (
	AdminSystemBroadcastScheduled AdminSystemBroadcastStatus = "scheduled"
	AdminSystemBroadcastSent      AdminSystemBroadcastStatus = "sent"
	AdminSystemBroadcastDraft     AdminSystemBroadcastStatus = "draft"
)

// AdminSystemBroadcast represents announcement message metadata.
type AdminSystemBroadcast struct {
	ID             string                     `json:"id"`
	Title          string                     `json:"title"`
	MessagePreview string                     `json:"message_preview"`
	Status         AdminSystemBroadcastStatus `json:"status"`
	TargetLabel    string                     `json:"target_label"`
	ScheduleLabel  string                     `json:"schedule_label"`
	CreatedBy      string                     `json:"created_by"`
	Pinned         bool                       `json:"pinned"`
}

// AdminSystemAuditLog captures admin modifications for auditing.
type AdminSystemAuditLog struct {
	Category  string `json:"category"`
	Action    string `json:"action"`
	Operator  string `json:"operator"`
	TimeLabel string `json:"time_label"`
	Detail    string `json:"detail"`

	createdAt time.Time
}

var (
	// ErrSystemSwitchNotFound indicates toggle is missing.
	ErrSystemSwitchNotFound = errors.New("system switch not found")
	// ErrSystemParameterNotFound indicates parameter is missing.
	ErrSystemParameterNotFound = errors.New("system parameter not found")
	// ErrSystemBroadcastNotFound indicates broadcast is missing.
	ErrSystemBroadcastNotFound = errors.New("system broadcast not found")
)

// AdminSystemService manages in-memory system configuration for demo purposes.
type AdminSystemService struct {
	mu         sync.RWMutex
	switches   map[string][]AdminSystemSwitch
	parameters map[string][]AdminSystemParameter
	broadcasts map[string][]AdminSystemBroadcast
	auditLogs  map[string][]AdminSystemAuditLog
}

// NewAdminSystemService constructs a service with seed data.
func NewAdminSystemService() *AdminSystemService {
	return &AdminSystemService{
		switches:   make(map[string][]AdminSystemSwitch),
		parameters: make(map[string][]AdminSystemParameter),
		broadcasts: make(map[string][]AdminSystemBroadcast),
		auditLogs:  make(map[string][]AdminSystemAuditLog),
	}
}

// ListSwitches returns switches for a school.
func (s *AdminSystemService) ListSwitches(schoolID string) []AdminSystemSwitch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureSchoolLocked(schoolID)
	return cloneSwitches(s.switches[schoolID])
}

// UpdateSwitchState toggles a switch and records audit log.
func (s *AdminSystemService) UpdateSwitchState(schoolID, switchID string, enabled bool, operator string) (*AdminSystemSwitch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchoolLocked(schoolID)
	items := s.switches[schoolID]
	for idx := range items {
		if items[idx].ID == switchID {
			items[idx].Enabled = enabled
			if operator == "" {
				operator = "系统管理员"
			}
			items[idx].LastUpdatedLabel = fmt.Sprintf("最近更新：%s · 由 %s", time.Now().Format("2006-01-02 15:04"), operator)
			s.switches[schoolID][idx] = items[idx]
			s.appendAuditLocked(schoolID, "系统开关", fmt.Sprintf("更新 %s", items[idx].Title), operator, fmt.Sprintf("已%s「%s」", statusVerb(enabled), items[idx].Title))
			updated := items[idx]
			return &updated, nil
		}
	}
	return nil, ErrSystemSwitchNotFound
}

// ListParameters returns runtime parameters for a school.
func (s *AdminSystemService) ListParameters(schoolID string) []AdminSystemParameter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureSchoolLocked(schoolID)
	return cloneParameters(s.parameters[schoolID])
}

// UpdateParameter changes parameter value.
func (s *AdminSystemService) UpdateParameter(schoolID, parameterID, value, operator string) (*AdminSystemParameter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchoolLocked(schoolID)
	items := s.parameters[schoolID]
	trimmed := strings.TrimSpace(value)
	if operator == "" {
		operator = "系统管理员"
	}
	for idx := range items {
		if items[idx].ID == parameterID {
			if items[idx].Locked {
				return nil, fmt.Errorf("parameter %s is locked", items[idx].Key)
			}
			items[idx].Value = trimmed
			items[idx].LastUpdatedLabel = fmt.Sprintf("更新于 %s", time.Now().Format("2006-01-02 15:04"))
			s.parameters[schoolID][idx] = items[idx]
			s.appendAuditLocked(schoolID, "平台参数", fmt.Sprintf("修改 %s", items[idx].Key), operator, fmt.Sprintf("更新为 %s", trimmed))
			updated := items[idx]
			return &updated, nil
		}
	}
	return nil, ErrSystemParameterNotFound
}

// ListBroadcasts returns announcements for a school.
func (s *AdminSystemService) ListBroadcasts(schoolID string) []AdminSystemBroadcast {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureSchoolLocked(schoolID)
	return cloneBroadcasts(s.broadcasts[schoolID])
}

// UpdateBroadcast updates broadcast status or pin state.
func (s *AdminSystemService) UpdateBroadcast(schoolID, broadcastID string, status *AdminSystemBroadcastStatus, pinned *bool, operator string) (*AdminSystemBroadcast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchoolLocked(schoolID)
	if operator == "" {
		operator = "系统管理员"
	}
	items := s.broadcasts[schoolID]
	for idx := range items {
		if items[idx].ID != broadcastID {
			continue
		}

		if status != nil {
			items[idx].Status = *status
			switch *status {
			case AdminSystemBroadcastSent:
				items[idx].ScheduleLabel = fmt.Sprintf("发送时间：%s", time.Now().Format("01-02 15:04"))
			case AdminSystemBroadcastScheduled:
				items[idx].ScheduleLabel = fmt.Sprintf("计划发送：%s", time.Now().Format("01-02 15:04"))
			case AdminSystemBroadcastDraft:
				items[idx].ScheduleLabel = fmt.Sprintf("草稿保存：%s", time.Now().Format("01-02 15:04"))
			}
		}

		if pinned != nil {
			items[idx].Pinned = *pinned
		}

		s.broadcasts[schoolID][idx] = items[idx]
		var changes []string
		if status != nil {
			changes = append(changes, fmt.Sprintf("更新状态为 %s", string(*status)))
		}
		if pinned != nil {
			if *pinned {
				changes = append(changes, "置顶")
			} else {
				changes = append(changes, "取消置顶")
			}
		}
		s.appendAuditLocked(schoolID, "通知广播", fmt.Sprintf("更新公告 %s", items[idx].Title), operator, strings.Join(changes, "；"))
		updated := items[idx]
		return &updated, nil
	}
	return nil, ErrSystemBroadcastNotFound
}

// ListAuditLogs returns audit logs sorted by time desc. limit <=0 returns all.
func (s *AdminSystemService) ListAuditLogs(schoolID string, limit int) []AdminSystemAuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureSchoolLocked(schoolID)
	logs := cloneAuditLogs(s.auditLogs[schoolID])
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].createdAt.After(logs[j].createdAt)
	})
	if limit > 0 && len(logs) > limit {
		return logs[:limit]
	}
	return logs
}

func (s *AdminSystemService) ensureSchoolLocked(schoolID string) {
	if schoolID == "" {
		schoolID = "default"
	}
	if _, ok := s.switches[schoolID]; !ok {
		s.switches[schoolID] = cloneSwitches(defaultSystemSwitches)
	}
	if _, ok := s.parameters[schoolID]; !ok {
		s.parameters[schoolID] = cloneParameters(defaultSystemParameters)
	}
	if _, ok := s.broadcasts[schoolID]; !ok {
		s.broadcasts[schoolID] = cloneBroadcasts(defaultSystemBroadcasts)
	}
	if _, ok := s.auditLogs[schoolID]; !ok {
		s.auditLogs[schoolID] = cloneAuditLogs(defaultSystemAuditLogs)
	}
}

func (s *AdminSystemService) appendAuditLocked(schoolID, category, action, operator, detail string) {
	logs := s.auditLogs[schoolID]
	now := time.Now()
	entry := AdminSystemAuditLog{
		Category:  category,
		Action:    action,
		Operator:  operator,
		Detail:    detail,
		TimeLabel: now.Format("01-02 15:04"),
		createdAt: now,
	}
	logs = append([]AdminSystemAuditLog{entry}, logs...)
	if len(logs) > 200 {
		logs = logs[:200]
	}
	s.auditLogs[schoolID] = logs
}

func statusVerb(enabled bool) string {
	if enabled {
		return "启用"
	}
	return "停用"
}

func cloneSwitches(values []AdminSystemSwitch) []AdminSystemSwitch {
	out := make([]AdminSystemSwitch, len(values))
	copy(out, values)
	return out
}

func cloneParameters(values []AdminSystemParameter) []AdminSystemParameter {
	out := make([]AdminSystemParameter, len(values))
	copy(out, values)
	return out
}

func cloneBroadcasts(values []AdminSystemBroadcast) []AdminSystemBroadcast {
	out := make([]AdminSystemBroadcast, len(values))
	copy(out, values)
	return out
}

func cloneAuditLogs(values []AdminSystemAuditLog) []AdminSystemAuditLog {
	out := make([]AdminSystemAuditLog, len(values))
	copy(out, values)
	return out
}

var defaultSystemSwitches = []AdminSystemSwitch{
	{
		ID:               "switch-maintenance",
		Title:            "夜间维护模式",
		Description:      "在每日 23:30 - 06:30 内限制学生端访问，教师端保持可用。",
		Enabled:          true,
		LastUpdatedLabel: "最近更新：2024-10-10 · 由 系统管理员",
		Responsible:      "运维团队",
		IconName:         "nightlight",
		Tags:             []string{"计划任务", "自动恢复"},
		Environment:      "生产环境",
	},
	{
		ID:               "switch-assignment-reminder",
		Title:            "作业提交提醒推送",
		Description:      "距离截止时间 1 小时自动推送提醒至学生端与家长端。",
		Enabled:          true,
		LastUpdatedLabel: "最近更新：2024-09-28 · 由 教务处",
		Responsible:      "教务处",
		IconName:         "notifications_active",
		Tags:             []string{"消息服务"},
		Environment:      "生产环境",
	},
	{
		ID:               "switch-ai-assist",
		Title:            "实验特性：AI 批改建议",
		Description:      "允许教师端使用智能批改建议，需单独开通权限。",
		Enabled:          false,
		LastUpdatedLabel: "最近更新：2024-10-18 · 由 创新组",
		Responsible:      "创新组",
		IconName:         "auto_awesome",
		Tags:             []string{"实验功能", "需申请"},
		Environment:      "预览环境",
	},
}

var defaultSystemParameters = []AdminSystemParameter{
	{
		ID:               "param-max-upload",
		Key:              "MAX_UPLOAD_SIZE_MB",
		Value:            "80",
		Scope:            "文件上传服务",
		Description:      "限制单个文件上传大小，单位 MB。",
		LastUpdatedLabel: "更新于 2024-09-30",
		Locked:           true,
	},
	{
		ID:               "param-timezone",
		Key:              "DEFAULT_TIMEZONE",
		Value:            "Asia/Shanghai",
		Scope:            "全局时区",
		Description:      "用于课程表、作业截止时间的默认时区。",
		LastUpdatedLabel: "更新于 2024-08-20",
	},
	{
		ID:               "param-session-timeout",
		Key:              "SESSION_IDLE_TIMEOUT",
		Value:            "30m",
		Scope:            "认证服务",
		Description:      "后台管理端空闲自动退出时长。",
		LastUpdatedLabel: "更新于 2024-10-05",
	},
	{
		ID:               "param-feature-flags",
		Key:              "FEATURE_FLAGS",
		Value:            "ai_grading,live_classroom",
		Scope:            "功能开关",
		Description:      "为指定院系开启的实验功能列表。",
		LastUpdatedLabel: "更新于 2024-10-18",
	},
}

var defaultSystemBroadcasts = []AdminSystemBroadcast{
	{
		ID:             "broadcast-oct-maintenance",
		Title:          "10 月系统升级通知",
		MessagePreview: "定于 10-28 02:00 - 03:00 进行数据库维护，期间服务可能短暂波动。",
		Status:         AdminSystemBroadcastScheduled,
		TargetLabel:    "全体教师、管理员",
		ScheduleLabel:  "计划发送：10-25 09:00",
		CreatedBy:      "系统管理员",
		Pinned:         true,
	},
	{
		ID:             "broadcast-ai-feature",
		Title:          "作业批改新功能上线",
		MessagePreview: "AI 批改建议将在试点院系开放试用，如需参与请向教务处申请。",
		Status:         AdminSystemBroadcastDraft,
		TargetLabel:    "教师端 · 试点院系",
		ScheduleLabel:  "草稿保存：10-19 16:20",
		CreatedBy:      "创新组",
		Pinned:         false,
	},
	{
		ID:             "broadcast-sept-report",
		Title:          "9 月例行巡检完成",
		MessagePreview: "服务性能已恢复正常，如遇问题可提交工单。",
		Status:         AdminSystemBroadcastSent,
		TargetLabel:    "全体用户",
		ScheduleLabel:  "发送时间：09-30 07:30",
		CreatedBy:      "运维团队",
		Pinned:         false,
	},
}

var defaultSystemAuditLogs = []AdminSystemAuditLog{
	{
		Category:  "平台参数",
		Action:    "修改 MAX_UPLOAD_SIZE_MB",
		Operator:  "系统管理员",
		Detail:    "将文件上传限制调整为 80MB，并同步至 CDN 缓存。",
		TimeLabel: time.Now().AddDate(0, 0, -5).Format("01-02 15:04"),
		createdAt: time.Now().AddDate(0, 0, -5),
	},
	{
		Category:  "安全策略",
		Action:    "启用夜间维护模式",
		Operator:  "运维团队",
		Detail:    "设置自动开启与关闭时间，已推送至状态页。",
		TimeLabel: time.Now().AddDate(0, 0, -3).Format("01-02 15:04"),
		createdAt: time.Now().AddDate(0, 0, -3),
	},
	{
		Category:  "通知广播",
		Action:    "发布 9 月巡检完成公告",
		Operator:  "运维团队",
		Detail:    "公告同步至网页端公告栏与短信渠道。",
		TimeLabel: time.Now().AddDate(0, 0, -1).Format("01-02 15:04"),
		createdAt: time.Now().AddDate(0, 0, -1),
	},
}
