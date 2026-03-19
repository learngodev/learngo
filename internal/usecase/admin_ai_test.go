package service

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	identitybiz "learn-go/internal/biz/identity"
	sharedbiz "learn-go/internal/biz/shared"
	"strings"
	"testing"
)

func TestExecuteBatchOperations_GeneratedNumber(t *testing.T) {
	repo := newMemoryAccountRepo()
	svc := newTestAdminService(repo)

	schoolID := uuid.New()
	ops := []AIOperation{
		{
			Action: "create_student",
			Data: json.RawMessage(`{
				"name": "Student A",
				"password": "password",
				"number": "generated_or_placeholder"
			}`),
		},
		{
			Action: "create_student",
			Data: json.RawMessage(`{
				"name": "Student B",
				"password": "password",
				"number": ""
			}`),
		},
		{
			Action: "create_teacher",
			Data: json.RawMessage(`{
				"name": "Teacher A",
				"password": "password",
				"number": "generated_or_placeholder"
			}`),
		},
	}

	results, err := svc.ExecuteBatchOperations(context.Background(), schoolID, ops)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify accounts were created with generated numbers
	if len(repo.accounts) != 3 {
		t.Fatalf("expected 3 accounts created, got %d", len(repo.accounts))
	}

	for _, acc := range repo.accounts {
		if acc.Identifier == "generated_or_placeholder" {
			t.Errorf("account %s has placeholder identifier", acc.DisplayName)
		}
		if acc.Identifier == "" {
			t.Errorf("account %s has empty identifier", acc.DisplayName)
		}
		if acc.Role == "student" && !strings.HasPrefix(acc.Identifier, "S") {
			t.Errorf("student account %s identifier %s should start with S", acc.DisplayName, acc.Identifier)
		}
		if acc.Role == "teacher" && !strings.HasPrefix(acc.Identifier, "T") {
			t.Errorf("teacher account %s identifier %s should start with T", acc.DisplayName, acc.Identifier)
		}
	}
}

func TestExecuteBatchOperations_NameLookup(t *testing.T) {
	repo := newMemoryAccountRepo()
	schoolID := uuid.New()

	// Setup existing account
	repo.accounts["acc1"] = &identitybiz.Account{
		ID:          "acc1",
		SchoolID:    schoolID.String(),
		Role:        sharedbiz.RoleStudent,
		Status:      sharedbiz.AccountStatusActive,
		DisplayName: "Zhang San",
		Identifier:  "S123",
	}

	svc := newTestAdminService(repo)

	ops := []AIOperation{
		{
			Action: "lock_account",
			Data: json.RawMessage(`{
				"name": "Zhang San"
			}`),
		},
	}

	results, err := svc.ExecuteBatchOperations(context.Background(), schoolID, ops)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !strings.Contains(results[0], "已锁定账号 S123") {
		t.Errorf("expected result to contain '已锁定账号 S123', got %s", results[0])
	}

	if repo.accounts["acc1"].Status != sharedbiz.AccountStatusLocked {
		t.Errorf("expected account to be locked")
	}
}
