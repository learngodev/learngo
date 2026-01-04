package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

var (
	ErrOssPrimaryCredentialDeletion = errors.New("cannot delete primary credential")
)

// AdminOssService manages administrator OSS configuration operations.
type AdminOssService struct {
	credentials repository.OssCredentialRepository
	policies    repository.OssPolicyRepository
	audits      repository.OssAuditRepository
	accounts    repository.AccountRepository
}

// NewAdminOssService constructs an AdminOssService.
func NewAdminOssService(
	credentials repository.OssCredentialRepository,
	policies repository.OssPolicyRepository,
	audits repository.OssAuditRepository,
	accounts repository.AccountRepository,
) *AdminOssService {
	return &AdminOssService{
		credentials: credentials,
		policies:    policies,
		audits:      audits,
		accounts:    accounts,
	}
}

// AdminOssCredentialView represents credential data returned to clients.
type AdminOssCredentialView struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Endpoint             string     `json:"endpoint"`
	InternalEndpoint     string     `json:"internal_endpoint"`
	Region               string     `json:"region"`
	Bucket               string     `json:"bucket"`
	DirectoryPrefix      string     `json:"directory_prefix"`
	AccessKeyMasked      string     `json:"access_key_masked"`
	AllowPublicRead      bool       `json:"allow_public_read"`
	AllowMultipartUpload bool       `json:"allow_multipart_upload"`
	UseRelayUpload       bool       `json:"use_relay_upload"`
	IsPrimary            bool       `json:"is_primary"`
	Active               bool       `json:"active"`
	CreatedAt            time.Time  `json:"created_at"`
	LastRotatedAt        *time.Time `json:"last_rotated_at,omitempty"`
}

// AdminOssPolicyView represents policy data returned to clients.
type AdminOssPolicyView struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	AppliesTo     string                 `json:"applies_to"`
	Status        domain.OssPolicyStatus `json:"status"`
	LastUpdatedAt time.Time              `json:"last_updated_at"`
}

// AdminOssAuditLogView represents audit log data returned to clients.
type AdminOssAuditLogView struct {
	ID           string    `json:"id"`
	Action       string    `json:"action"`
	OperatorName string    `json:"operator"`
	Detail       string    `json:"detail"`
	CreatedAt    time.Time `json:"created_at"`
}

// UpdateOssCredentialInput collects optional updates for a credential.
type UpdateOssCredentialInput struct {
	SchoolID             string
	CredentialID         string
	Name                 *string
	Endpoint             *string
	InternalEndpoint     *string
	Region               *string
	Bucket               *string
	DirectoryPrefix      *string
	AccessKeyID          *string
	AccessKeySecret      *string
	AccessKeyDisplay     *string
	AllowPublicRead      *bool
	AllowMultipartUpload *bool
	UseRelayUpload       *bool
	Active               *bool
	IsPrimary            *bool
	OperatorID           string
}

// CreateOssCredentialInput collects fields required to create a credential.
type CreateOssCredentialInput struct {
	SchoolID             string
	Name                 string
	Endpoint             string
	InternalEndpoint     string
	Region               string
	Bucket               string
	DirectoryPrefix      string
	AccessKeyID          string
	AccessKeySecret      string
	AccessKeyDisplay     string
	AllowPublicRead      bool
	AllowMultipartUpload bool
	UseRelayUpload       bool
	Active               bool
	IsPrimary            bool
	OperatorID           string
}

// UpdateOssPolicyStatusInput collects updates for policy status.
type UpdateOssPolicyStatusInput struct {
	SchoolID   string
	PolicyID   string
	Status     domain.OssPolicyStatus
	OperatorID string
}

// CreateOssPolicyInput collects fields required to create a policy.
type CreateOssPolicyInput struct {
	SchoolID    string
	Name        string
	Description string
	AppliesTo   string
	Status      domain.OssPolicyStatus
	OperatorID  string
}

// ListCredentials returns all credentials for a school.
func (s *AdminOssService) ListCredentials(ctx context.Context, schoolID string) ([]AdminOssCredentialView, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}
	creds, err := s.credentials.List(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	out := make([]AdminOssCredentialView, 0, len(creds))
	for _, cred := range creds {
		out = append(out, toOssCredentialView(&cred))
	}
	return out, nil
}

// UpdateCredential applies partial updates to a credential.
func (s *AdminOssService) UpdateCredential(ctx context.Context, input UpdateOssCredentialInput) (*AdminOssCredentialView, error) {
	if strings.TrimSpace(input.SchoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if strings.TrimSpace(input.CredentialID) == "" {
		return nil, errors.New("credential_id required")
	}

	updates := make(map[string]any)
	var changeSummary []string

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		updates["name"] = trimmed
		changeSummary = append(changeSummary, fmt.Sprintf("名称→%s", trimmed))
	}
	if input.Endpoint != nil {
		trimmed := strings.TrimSpace(*input.Endpoint)
		updates["endpoint"] = trimmed
		changeSummary = append(changeSummary, fmt.Sprintf("Endpoint→%s", trimmed))
	}
	if input.InternalEndpoint != nil {
		trimmed := strings.TrimSpace(*input.InternalEndpoint)
		updates["internal_endpoint"] = trimmed
		if trimmed == "" {
			changeSummary = append(changeSummary, "清空内网Endpoint")
		} else {
			changeSummary = append(changeSummary, fmt.Sprintf("内网Endpoint→%s", trimmed))
		}
	}
	if input.Region != nil {
		trimmed := strings.TrimSpace(*input.Region)
		updates["region"] = trimmed
		changeSummary = append(changeSummary, fmt.Sprintf("区域→%s", trimmed))
	}
	if input.Bucket != nil {
		trimmed := strings.TrimSpace(*input.Bucket)
		updates["bucket"] = trimmed
		changeSummary = append(changeSummary, fmt.Sprintf("Bucket→%s", trimmed))
	}
	if input.DirectoryPrefix != nil {
		prefix := strings.TrimSpace(*input.DirectoryPrefix)
		updates["directory_prefix"] = prefix
		changeSummary = append(changeSummary, fmt.Sprintf("目录前缀→%s", prefix))
	}
	if input.AccessKeyID != nil {
		updates["access_key_id"] = strings.TrimSpace(*input.AccessKeyID)
		now := time.Now()
		updates["last_rotated_at"] = &now
		changeSummary = append(changeSummary, "更新AccessKeyID并刷新轮换时间")
	}
	if input.AccessKeySecret != nil {
		updates["access_key_secret"] = strings.TrimSpace(*input.AccessKeySecret)
		now := time.Now()
		updates["last_rotated_at"] = &now
		changeSummary = append(changeSummary, "更新AccessKeySecret并刷新轮换时间")
	}
	if input.AccessKeyDisplay != nil {
		display := strings.TrimSpace(*input.AccessKeyDisplay)
		updates["access_key_display"] = display
		now := time.Now()
		updates["last_rotated_at"] = &now
		changeSummary = append(changeSummary, "更新访问凭证并刷新轮换时间")
	}
	if input.AllowPublicRead != nil {
		updates["allow_public_read"] = *input.AllowPublicRead
		if *input.AllowPublicRead {
			changeSummary = append(changeSummary, "开启公开只读")
		} else {
			changeSummary = append(changeSummary, "关闭公开只读")
		}
	}
	if input.AllowMultipartUpload != nil {
		updates["allow_multipart_upload"] = *input.AllowMultipartUpload
		if *input.AllowMultipartUpload {
			changeSummary = append(changeSummary, "开启分片上传")
		} else {
			changeSummary = append(changeSummary, "关闭分片上传")
		}
	}
	if input.UseRelayUpload != nil {
		updates["use_relay_upload"] = *input.UseRelayUpload
		if *input.UseRelayUpload {
			changeSummary = append(changeSummary, "启用服务端中继上传")
		} else {
			changeSummary = append(changeSummary, "关闭服务端中继上传")
		}
	}
	if input.Active != nil {
		updates["active"] = *input.Active
		if *input.Active {
			changeSummary = append(changeSummary, "启用凭证")
		} else {
			changeSummary = append(changeSummary, "停用凭证")
		}
	}

	var cred *domain.OssCredential
	var err error

	if len(updates) > 0 {
		cred, err = s.credentials.Update(ctx, input.CredentialID, input.SchoolID, updates)
		if err != nil {
			return nil, err
		}
	}

	if input.IsPrimary != nil {
		if *input.IsPrimary {
			cred, err = s.credentials.SetPrimary(ctx, input.CredentialID, input.SchoolID)
			if err != nil {
				return nil, err
			}
			changeSummary = append(changeSummary, "设为主凭证并保持启用")
		} else {
			cred, err = s.credentials.Update(ctx, input.CredentialID, input.SchoolID, map[string]any{"is_primary": false})
			if err != nil {
				return nil, err
			}
			changeSummary = append(changeSummary, "取消主凭证标记")
		}
	}

	if cred == nil {
		cred, err = s.credentials.GetByID(ctx, input.CredentialID, input.SchoolID)
		if err != nil {
			return nil, err
		}
	}

	if err := s.recordAudit(ctx, input.SchoolID, input.OperatorID, "更新访问凭证", changeSummary); err != nil {
		return nil, err
	}

	view := toOssCredentialView(cred)
	return &view, nil
}

// CreateCredential adds a new OSS credential for a school.
func (s *AdminOssService) CreateCredential(ctx context.Context, input CreateOssCredentialInput) (*AdminOssCredentialView, error) {
	schoolID := strings.TrimSpace(input.SchoolID)
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("name required")
	}
	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		return nil, errors.New("endpoint required")
	}
	internalEndpoint := strings.TrimSpace(input.InternalEndpoint)
	region := strings.TrimSpace(input.Region)
	if region == "" {
		return nil, errors.New("region required")
	}
	bucket := strings.TrimSpace(input.Bucket)
	if bucket == "" {
		return nil, errors.New("bucket required")
	}

	directoryPrefix := strings.TrimSpace(input.DirectoryPrefix)
	accessKeyID := strings.TrimSpace(input.AccessKeyID)
	accessKeySecret := strings.TrimSpace(input.AccessKeySecret)
	accessKeyDisplay := strings.TrimSpace(input.AccessKeyDisplay)

	credential := &domain.OssCredential{
		ID:                   uuid.NewString(),
		SchoolID:             schoolID,
		Name:                 name,
		Endpoint:             endpoint,
		InternalEndpoint:     internalEndpoint,
		Region:               region,
		Bucket:               bucket,
		DirectoryPrefix:      directoryPrefix,
		AccessKeyID:          accessKeyID,
		AccessKeySecret:      accessKeySecret,
		AccessKeyDisplay:     accessKeyDisplay,
		AllowPublicRead:      input.AllowPublicRead,
		AllowMultipartUpload: input.AllowMultipartUpload,
		UseRelayUpload:       input.UseRelayUpload,
		Active:               input.Active,
		IsPrimary:            input.IsPrimary,
	}

	if !credential.Active && credential.IsPrimary {
		credential.Active = true
	}

	if err := s.credentials.Create(ctx, credential); err != nil {
		return nil, err
	}

	stored := credential
	if input.IsPrimary {
		updated, err := s.credentials.SetPrimary(ctx, credential.ID, schoolID)
		if err != nil {
			return nil, err
		}
		stored = updated
	}

	detail := []string{fmt.Sprintf("创建访问凭证 %s", stored.Name)}
	if stored.IsPrimary {
		detail = append(detail, "设为主凭证")
	}
	if strings.TrimSpace(stored.InternalEndpoint) != "" {
		detail = append(detail, "配置内网Endpoint")
	}
	if stored.AllowPublicRead {
		detail = append(detail, "开启公开只读")
	}
	if stored.AllowMultipartUpload {
		detail = append(detail, "开启分片上传")
	}
	if stored.UseRelayUpload {
		detail = append(detail, "启用服务端中继上传")
	}

	if err := s.recordAudit(ctx, schoolID, input.OperatorID, "创建访问凭证", detail); err != nil {
		return nil, err
	}

	view := toOssCredentialView(stored)
	return &view, nil
}

// DeleteCredential removes a credential that is not primary.
func (s *AdminOssService) DeleteCredential(ctx context.Context, schoolID, credentialID, operatorID string) error {
	schoolID = strings.TrimSpace(schoolID)
	credentialID = strings.TrimSpace(credentialID)
	if schoolID == "" {
		return errors.New("school_id required")
	}
	if credentialID == "" {
		return errors.New("credential_id required")
	}

	cred, err := s.credentials.GetByID(ctx, credentialID, schoolID)
	if err != nil {
		return err
	}
	if cred.IsPrimary {
		return ErrOssPrimaryCredentialDeletion
	}

	if err := s.credentials.Delete(ctx, credentialID, schoolID); err != nil {
		return err
	}

	detail := []string{fmt.Sprintf("删除访问凭证 %s", cred.Name)}
	return s.recordAudit(ctx, schoolID, operatorID, "删除访问凭证", detail)
}

// ListPolicies returns all policies for a school.
func (s *AdminOssService) ListPolicies(ctx context.Context, schoolID string) ([]AdminOssPolicyView, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}
	records, err := s.policies.List(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	out := make([]AdminOssPolicyView, 0, len(records))
	for _, policy := range records {
		out = append(out, toOssPolicyView(&policy))
	}
	return out, nil
}

// CreatePolicy adds a new OSS policy definition.
func (s *AdminOssService) CreatePolicy(ctx context.Context, input CreateOssPolicyInput) (*AdminOssPolicyView, error) {
	schoolID := strings.TrimSpace(input.SchoolID)
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("name required")
	}
	appliesTo := strings.TrimSpace(input.AppliesTo)
	if appliesTo == "" {
		return nil, errors.New("applies_to required")
	}

	description := strings.TrimSpace(input.Description)
	status := input.Status
	if status == "" {
		status = domain.OssPolicyStatusEnabled
	}
	switch status {
	case domain.OssPolicyStatusEnabled, domain.OssPolicyStatusReadOnly, domain.OssPolicyStatusDisabled:
	default:
		return nil, errors.New("invalid status")
	}

	now := time.Now()
	policy := &domain.OssPolicy{
		ID:            uuid.NewString(),
		SchoolID:      schoolID,
		Name:          name,
		Description:   description,
		AppliesTo:     appliesTo,
		Status:        status,
		LastUpdatedAt: now,
	}

	if err := s.policies.Create(ctx, policy); err != nil {
		return nil, err
	}

	detail := []string{fmt.Sprintf("创建安全策略 %s", policy.Name), fmt.Sprintf("状态→%s", policy.Status)}
	if err := s.recordAudit(ctx, schoolID, input.OperatorID, "创建安全策略", detail); err != nil {
		return nil, err
	}

	view := toOssPolicyView(policy)
	return &view, nil
}

// DeletePolicy removes an OSS policy.
func (s *AdminOssService) DeletePolicy(ctx context.Context, schoolID, policyID, operatorID string) error {
	schoolID = strings.TrimSpace(schoolID)
	policyID = strings.TrimSpace(policyID)
	if schoolID == "" {
		return errors.New("school_id required")
	}
	if policyID == "" {
		return errors.New("policy_id required")
	}

	policy, err := s.policies.GetByID(ctx, policyID, schoolID)
	if err != nil {
		return err
	}

	if err := s.policies.Delete(ctx, policyID, schoolID); err != nil {
		return err
	}

	detail := []string{fmt.Sprintf("删除安全策略 %s", policy.Name)}
	return s.recordAudit(ctx, schoolID, operatorID, "删除安全策略", detail)
}

// UpdatePolicyStatus updates policy status and logs the change.
func (s *AdminOssService) UpdatePolicyStatus(ctx context.Context, input UpdateOssPolicyStatusInput) (*AdminOssPolicyView, error) {
	if strings.TrimSpace(input.SchoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if strings.TrimSpace(input.PolicyID) == "" {
		return nil, errors.New("policy_id required")
	}
	if input.Status == "" {
		return nil, errors.New("status required")
	}

	policy, err := s.policies.UpdateStatus(ctx, input.PolicyID, input.SchoolID, input.Status)
	if err != nil {
		return nil, err
	}

	detail := fmt.Sprintf("策略状态更新为 %s", string(input.Status))
	if err := s.recordAudit(ctx, input.SchoolID, input.OperatorID, "更新安全策略", []string{detail}); err != nil {
		return nil, err
	}

	view := toOssPolicyView(policy)
	return &view, nil
}

// ListAuditLogs returns recent audit logs for a school.
func (s *AdminOssService) ListAuditLogs(ctx context.Context, schoolID string, limit int) ([]AdminOssAuditLogView, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}
	logs, err := s.audits.ListRecent(ctx, schoolID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]AdminOssAuditLogView, 0, len(logs))
	for _, entry := range logs {
		out = append(out, AdminOssAuditLogView{
			ID:           entry.ID,
			Action:       entry.Action,
			OperatorName: entry.OperatorName,
			Detail:       entry.Detail,
			CreatedAt:    entry.CreatedAt,
		})
	}
	return out, nil
}

func (s *AdminOssService) recordAudit(ctx context.Context, schoolID, operatorID, action string, details []string) error {
	if schoolID == "" {
		return errors.New("school_id required for audit log")
	}
	operatorName := "系统管理员"
	if operatorID != "" {
		account, err := s.accounts.FindByID(ctx, operatorID)
		if err == nil && account != nil && strings.TrimSpace(account.DisplayName) != "" {
			operatorName = account.DisplayName
		}
	}

	detail := strings.Join(details, "；")
	log := &domain.OssAuditLog{
		ID:           uuid.NewString(),
		SchoolID:     schoolID,
		Action:       action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Detail:       detail,
		CreatedAt:    time.Now(),
	}
	return s.audits.Create(ctx, log)
}

func toOssCredentialView(cred *domain.OssCredential) AdminOssCredentialView {
	var masked string
	if trimmed := strings.TrimSpace(cred.AccessKeyDisplay); trimmed != "" {
		masked = maskAccessKey(trimmed)
	}
	return AdminOssCredentialView{
		ID:                   cred.ID,
		Name:                 cred.Name,
		Endpoint:             cred.Endpoint,
		InternalEndpoint:     cred.InternalEndpoint,
		Region:               cred.Region,
		Bucket:               cred.Bucket,
		DirectoryPrefix:      cred.DirectoryPrefix,
		AccessKeyMasked:      masked,
		AllowPublicRead:      cred.AllowPublicRead,
		AllowMultipartUpload: cred.AllowMultipartUpload,
		UseRelayUpload:       cred.UseRelayUpload,
		IsPrimary:            cred.IsPrimary,
		Active:               cred.Active,
		CreatedAt:            cred.CreatedAt,
		LastRotatedAt:        cred.LastRotatedAt,
	}
}

func toOssPolicyView(policy *domain.OssPolicy) AdminOssPolicyView {
	return AdminOssPolicyView{
		ID:            policy.ID,
		Name:          policy.Name,
		Description:   policy.Description,
		AppliesTo:     policy.AppliesTo,
		Status:        policy.Status,
		LastUpdatedAt: policy.LastUpdatedAt,
	}
}

func maskAccessKey(v string) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	ln := len(runes)
	if ln <= 4 {
		return "****"
	}
	if ln <= 6 {
		return string(runes[:2]) + strings.Repeat("*", ln-4) + string(runes[ln-2:])
	}
	prefix := string(runes[:4])
	suffix := string(runes[ln-2:])
	return prefix + strings.Repeat("*", ln-6) + suffix
}
