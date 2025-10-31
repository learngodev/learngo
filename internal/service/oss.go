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
    ID                  string     `json:"id"`
    Name                string     `json:"name"`
    Endpoint            string     `json:"endpoint"`
    Region              string     `json:"region"`
    Bucket              string     `json:"bucket"`
    DirectoryPrefix     string     `json:"directory_prefix"`
    AccessKeyMasked     string     `json:"access_key_masked"`
    AllowPublicRead     bool       `json:"allow_public_read"`
    AllowMultipartUpload bool      `json:"allow_multipart_upload"`
    IsPrimary           bool       `json:"is_primary"`
    Active              bool       `json:"active"`
    CreatedAt           time.Time  `json:"created_at"`
    LastRotatedAt       *time.Time `json:"last_rotated_at,omitempty"`
}

// AdminOssPolicyView represents policy data returned to clients.
type AdminOssPolicyView struct {
    ID            string                `json:"id"`
    Name          string                `json:"name"`
    Description   string                `json:"description"`
    AppliesTo     string                `json:"applies_to"`
    Status        domain.OssPolicyStatus `json:"status"`
    LastUpdatedAt time.Time             `json:"last_updated_at"`
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
    SchoolID            string
    CredentialID        string
    Name                *string
    Endpoint            *string
    Region              *string
    Bucket              *string
    DirectoryPrefix     *string
    AccessKeyDisplay    *string
    AllowPublicRead     *bool
    AllowMultipartUpload *bool
    Active              *bool
    IsPrimary           *bool
    OperatorID          string
}

// UpdateOssPolicyStatusInput collects updates for policy status.
type UpdateOssPolicyStatusInput struct {
    SchoolID   string
    PolicyID   string
    Status     domain.OssPolicyStatus
    OperatorID string
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
        ID:                  cred.ID,
        Name:                cred.Name,
        Endpoint:            cred.Endpoint,
        Region:              cred.Region,
        Bucket:              cred.Bucket,
        DirectoryPrefix:     cred.DirectoryPrefix,
        AccessKeyMasked:     masked,
        AllowPublicRead:     cred.AllowPublicRead,
        AllowMultipartUpload: cred.AllowMultipartUpload,
        IsPrimary:           cred.IsPrimary,
        Active:              cred.Active,
        CreatedAt:           cred.CreatedAt,
        LastRotatedAt:       cred.LastRotatedAt,
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
*** End of File
