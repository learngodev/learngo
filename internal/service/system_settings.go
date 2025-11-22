package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
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

// AdminSystemService manages system configuration with persistence.
type AdminSystemService struct {
	switchRepo    repository.SystemSwitchRepository
	parameterRepo repository.SystemParameterRepository
	broadcastRepo repository.SystemBroadcastRepository
	auditRepo     repository.SystemAuditRepository

	seedMu sync.Mutex
	seeded map[string]bool
}

// NewAdminSystemService constructs a service backed by repositories.
func NewAdminSystemService(
	switches repository.SystemSwitchRepository,
	parameters repository.SystemParameterRepository,
	broadcasts repository.SystemBroadcastRepository,
	audits repository.SystemAuditRepository,
) *AdminSystemService {
	return &AdminSystemService{
		switchRepo:    switches,
		parameterRepo: parameters,
		broadcastRepo: broadcasts,
		auditRepo:     audits,
		seeded:        make(map[string]bool),
	}
}

// ListSwitches returns switches for a school.
func (s *AdminSystemService) ListSwitches(ctx context.Context, schoolID string) ([]AdminSystemSwitch, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	switches, err := s.switchRepo.List(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	out := make([]AdminSystemSwitch, len(switches))
	for i := range switches {
		out[i] = mapSwitchModel(switches[i])
	}
	return out, nil
}

// UpdateSwitchState toggles a switch and records audit log.
func (s *AdminSystemService) UpdateSwitchState(ctx context.Context, schoolID, switchID string, enabled bool, operator string) (*AdminSystemSwitch, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	if operator == "" {
		operator = "系统管理员"
	}
	label := fmt.Sprintf("最近更新：%s · 由 %s", time.Now().Format("2006-01-02 15:04"), operator)
	updated, err := s.switchRepo.UpdateFields(ctx, schoolID, switchID, map[string]any{
		"enabled":            enabled,
		"last_updated_label": label,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSystemSwitchNotFound
		}
		return nil, err
	}
	if err := s.appendAudit(ctx, schoolID, "系统开关", fmt.Sprintf("更新 %s", updated.Title), operator, fmt.Sprintf("已%s「%s」", statusVerb(enabled), updated.Title)); err != nil {
		return nil, err
	}
	mapped := mapSwitchModel(*updated)
	return &mapped, nil
}

// ListParameters returns runtime parameters for a school.
func (s *AdminSystemService) ListParameters(ctx context.Context, schoolID string) ([]AdminSystemParameter, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	params, err := s.parameterRepo.List(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	out := make([]AdminSystemParameter, len(params))
	for i := range params {
		out[i] = mapParameterModel(params[i])
	}
	return out, nil
}

// UpdateParameter changes parameter value.
func (s *AdminSystemService) UpdateParameter(ctx context.Context, schoolID, parameterID, value, operator string) (*AdminSystemParameter, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	if operator == "" {
		operator = "系统管理员"
	}
	trimmed := strings.TrimSpace(value)
	param, err := s.parameterRepo.Get(ctx, schoolID, parameterID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSystemParameterNotFound
		}
		return nil, err
	}
	if param.Locked {
		return nil, fmt.Errorf("parameter %s is locked", param.Key)
	}
	label := fmt.Sprintf("更新于 %s", time.Now().Format("2006-01-02 15:04"))
	updated, err := s.parameterRepo.UpdateFields(ctx, schoolID, parameterID, map[string]any{
		"value":              trimmed,
		"last_updated_label": label,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSystemParameterNotFound
		}
		return nil, err
	}
	if err := s.appendAudit(ctx, schoolID, "平台参数", fmt.Sprintf("修改 %s", updated.Key), operator, fmt.Sprintf("更新为 %s", trimmed)); err != nil {
		return nil, err
	}
	mapped := mapParameterModel(*updated)
	return &mapped, nil
}

// ListBroadcasts returns announcements for a school.
func (s *AdminSystemService) ListBroadcasts(ctx context.Context, schoolID string) ([]AdminSystemBroadcast, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	broadcasts, err := s.broadcastRepo.List(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	out := make([]AdminSystemBroadcast, len(broadcasts))
	for i := range broadcasts {
		out[i] = mapBroadcastModel(broadcasts[i])
	}
	return out, nil
}

// UpdateBroadcast updates broadcast status or pin state.
func (s *AdminSystemService) UpdateBroadcast(ctx context.Context, schoolID, broadcastID string, status *AdminSystemBroadcastStatus, pinned *bool, operator string) (*AdminSystemBroadcast, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	if operator == "" {
		operator = "系统管理员"
	}
	existing, err := s.broadcastRepo.Get(ctx, schoolID, broadcastID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSystemBroadcastNotFound
		}
		return nil, err
	}
	updates := make(map[string]any)
	var changes []string
	if status != nil {
		updates["status"] = string(*status)
		updates["schedule_label"] = buildBroadcastScheduleLabel(*status)
		changes = append(changes, fmt.Sprintf("更新状态为 %s", string(*status)))
	}
	if pinned != nil {
		updates["pinned"] = *pinned
		if *pinned {
			changes = append(changes, "置顶")
		} else {
			changes = append(changes, "取消置顶")
		}
	}
	if len(updates) == 0 {
		mapped := mapBroadcastModel(*existing)
		return &mapped, nil
	}
	updated, err := s.broadcastRepo.UpdateFields(ctx, schoolID, broadcastID, updates)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSystemBroadcastNotFound
		}
		return nil, err
	}
	if len(changes) > 0 {
		if err := s.appendAudit(ctx, schoolID, "通知广播", fmt.Sprintf("更新公告 %s", updated.Title), operator, strings.Join(changes, "；")); err != nil {
			return nil, err
		}
	}
	mapped := mapBroadcastModel(*updated)
	return &mapped, nil
}

// ListAuditLogs returns audit logs sorted by time desc. limit <=0 returns all.
func (s *AdminSystemService) ListAuditLogs(ctx context.Context, schoolID string, limit int) ([]AdminSystemAuditLog, error) {
	schoolID = normalizeSchoolID(schoolID)
	if err := s.ensureDefaults(ctx, schoolID); err != nil {
		return nil, err
	}
	logs, err := s.auditRepo.List(ctx, schoolID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]AdminSystemAuditLog, len(logs))
	for i := range logs {
		out[i] = AdminSystemAuditLog{
			Category:  logs[i].Category,
			Action:    logs[i].Action,
			Operator:  logs[i].Operator,
			Detail:    logs[i].Detail,
			TimeLabel: logs[i].TimeLabel,
			createdAt: logs[i].CreatedAt,
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].createdAt.After(out[j].createdAt)
	})
	if limit > 0 && len(out) > limit {
		return out[:limit], nil
	}
	return out, nil
}

func (s *AdminSystemService) appendAudit(ctx context.Context, schoolID, category, action, operator, detail string) error {
	if operator == "" {
		operator = "系统管理员"
	}
	entry := &domain.SystemAuditLog{
		SchoolID:  schoolID,
		Category:  category,
		Action:    action,
		Operator:  operator,
		Detail:    detail,
		CreatedAt: time.Now(),
	}
	entry.TimeLabel = entry.CreatedAt.Format("01-02 15:04")
	return s.auditRepo.Create(ctx, entry)
}

func statusVerb(enabled bool) string {
	if enabled {
		return "启用"
	}
	return "停用"
}

func normalizeSchoolID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "default"
	}
	return id
}

func mapSwitchModel(model domain.SystemSwitch) AdminSystemSwitch {
	return AdminSystemSwitch{
		ID:               model.ID,
		Title:            model.Title,
		Description:      model.Description,
		Enabled:          model.Enabled,
		LastUpdatedLabel: model.LastUpdatedLabel,
		Responsible:      model.Responsible,
		IconName:         model.IconName,
		Tags:             parseTags(model.Tags),
		Environment:      model.Environment,
	}
}

func mapParameterModel(model domain.SystemParameter) AdminSystemParameter {
	return AdminSystemParameter{
		ID:               model.ID,
		Key:              model.Key,
		Value:            model.Value,
		Scope:            model.Scope,
		Description:      model.Description,
		LastUpdatedLabel: model.LastUpdatedLabel,
		Locked:           model.Locked,
	}
}

func mapBroadcastModel(model domain.SystemBroadcast) AdminSystemBroadcast {
	return AdminSystemBroadcast{
		ID:             model.ID,
		Title:          model.Title,
		MessagePreview: model.MessagePreview,
		Status:         AdminSystemBroadcastStatus(model.Status),
		TargetLabel:    model.TargetLabel,
		ScheduleLabel:  model.ScheduleLabel,
		CreatedBy:      model.CreatedBy,
		Pinned:         model.Pinned,
	}
}

func parseTags(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return strings.Join(tags, ",")
}

func buildBroadcastScheduleLabel(status AdminSystemBroadcastStatus) string {
	now := time.Now().Format("01-02 15:04")
	switch status {
	case AdminSystemBroadcastSent:
		return fmt.Sprintf("发送时间：%s", now)
	case AdminSystemBroadcastScheduled:
		return fmt.Sprintf("计划发送：%s", now)
	default:
		return fmt.Sprintf("草稿保存：%s", now)
	}
}

func (s *AdminSystemService) ensureDefaults(ctx context.Context, schoolID string) error {
	s.seedMu.Lock()
	if s.seeded[schoolID] {
		s.seedMu.Unlock()
		return nil
	}
	s.seedMu.Unlock()

	if err := s.switchRepo.EnsureDefaults(ctx, schoolID, seedSwitchModels()); err != nil {
		return err
	}
	if err := s.parameterRepo.EnsureDefaults(ctx, schoolID, seedParameterModels()); err != nil {
		return err
	}
	if err := s.broadcastRepo.EnsureDefaults(ctx, schoolID, seedBroadcastModels()); err != nil {
		return err
	}
	if err := s.auditRepo.EnsureDefaults(ctx, schoolID, seedAuditLogModels()); err != nil {
		return err
	}

	s.seedMu.Lock()
	s.seeded[schoolID] = true
	s.seedMu.Unlock()
	return nil
}

func seedSwitchModels() []domain.SystemSwitch {
	now := time.Now()
	out := make([]domain.SystemSwitch, len(defaultSystemSwitches))
	for i, sw := range defaultSystemSwitches {
		out[i] = domain.SystemSwitch{
			ID:               sw.ID,
			Title:            sw.Title,
			Description:      sw.Description,
			Enabled:          sw.Enabled,
			LastUpdatedLabel: sw.LastUpdatedLabel,
			Responsible:      sw.Responsible,
			IconName:         sw.IconName,
			Tags:             joinTags(sw.Tags),
			Environment:      sw.Environment,
			SortOrder:        i + 1,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
	return out
}

func seedParameterModels() []domain.SystemParameter {
	now := time.Now()
	out := make([]domain.SystemParameter, len(defaultSystemParameters))
	for i, param := range defaultSystemParameters {
		out[i] = domain.SystemParameter{
			ID:               param.ID,
			Key:              param.Key,
			Value:            param.Value,
			Scope:            param.Scope,
			Description:      param.Description,
			LastUpdatedLabel: param.LastUpdatedLabel,
			Locked:           param.Locked,
			SortOrder:        i + 1,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
	return out
}

func seedBroadcastModels() []domain.SystemBroadcast {
	now := time.Now()
	out := make([]domain.SystemBroadcast, len(defaultSystemBroadcasts))
	for i, b := range defaultSystemBroadcasts {
		out[i] = domain.SystemBroadcast{
			ID:             b.ID,
			Title:          b.Title,
			MessagePreview: b.MessagePreview,
			Status:         string(b.Status),
			TargetLabel:    b.TargetLabel,
			ScheduleLabel:  b.ScheduleLabel,
			CreatedBy:      b.CreatedBy,
			Pinned:         b.Pinned,
			SortOrder:      i + 1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	return out
}

func seedAuditLogModels() []domain.SystemAuditLog {
	out := make([]domain.SystemAuditLog, len(defaultSystemAuditLogs))
	for i, log := range defaultSystemAuditLogs {
		out[i] = domain.SystemAuditLog{
			Category:  log.Category,
			Action:    log.Action,
			Operator:  log.Operator,
			Detail:    log.Detail,
			TimeLabel: log.TimeLabel,
			CreatedAt: log.createdAt,
		}
	}
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
