package storage

import (
	"context"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func setupTestSecurityDB(t *testing.T) (*DB, *SecurityStore) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	userStore, err := NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to create user store: %v", err)
	}

	// Create test user 1
	_, err = userStore.CreateUser(context.Background(), "user1@example.com", "Password123!")
	if err != nil {
		t.Fatalf("failed to create test user 1: %v", err)
	}

	// Create test user 2
	_, err = userStore.CreateUser(context.Background(), "user2@example.com", "Password123!")
	if err != nil {
		t.Fatalf("failed to create test user 2: %v", err)
	}

	secStore, err := NewSecurityStore(db)
	if err != nil {
		t.Fatalf("failed to create security store: %v", err)
	}

	return db, secStore
}

func TestSecurityStore_CRUD_And_Isolation(t *testing.T) {
	db, secStore := setupTestSecurityDB(t)
	defer db.Close()

	ctx := context.Background()

	report1 := &model.SecurityAuditReport{
		UserID:         1,
		TargetURL:      "https://goodai.co.nz",
		TargetHost:     "goodai.co.nz",
		TargetIP:       "158.179.29.27",
		CompositeScore: 92,
		Grade:          "A",
		GradeLabel:     "Strong Defense Posture",
		CreatedAt:      time.Now().UTC(),
		ExecutionMs:    280,
		Attestation:    "hash123",
		DomainScores: []model.CategoryBreakdown{
			{DomainKey: "ai_governance", DomainName: "AI Bot & Scraper Governance", Score: 21, MaxScore: 25, Status: "Strong"},
			{DomainKey: "headers", DomainName: "HTTP Security Headers Posture", Score: 21, MaxScore: 25, Status: "Strong"},
			{DomainKey: "tls", DomainName: "Transport & TLS Cryptography", Score: 25, MaxScore: 25, Status: "Hardened"},
			{DomainKey: "exposure", DomainName: "Exposure & Surface Shield", Score: 25, MaxScore: 25, Status: "Hardened"},
		},
		Findings: []model.FindingItem{
			{ID: "ai-1", Domain: "ai_governance", CheckName: "Robots", Severity: model.SeverityPass},
		},
	}

	if err := secStore.SaveReport(ctx, report1); err != nil {
		t.Fatalf("failed to save report: %v", err)
	}

	if report1.ID <= 0 {
		t.Errorf("expected generated report ID > 0, got %d", report1.ID)
	}

	// Retrieve by user 1
	got, err := secStore.GetReport(ctx, report1.ID, 1)
	if err != nil {
		t.Fatalf("failed to get report for user 1: %v", err)
	}
	if got.CompositeScore != 92 || got.TargetHost != "goodai.co.nz" {
		t.Errorf("mismatched report content: %+v", got)
	}
	if len(got.DomainScores) != 4 || len(got.Findings) != 1 {
		t.Errorf("expected domain scores and findings to be unmarshaled properly")
	}

	// User isolation: user 2 must NOT access user 1's report
	_, err = secStore.GetReport(ctx, report1.ID, 2)
	if err == nil {
		t.Errorf("expected user 2 access to user 1 report to fail, but succeeded")
	}

	// List reports for user 1
	list1, err := secStore.ListReports(ctx, 1, 10)
	if err != nil {
		t.Fatalf("failed to list reports for user 1: %v", err)
	}
	if len(list1) != 1 {
		t.Errorf("expected 1 report for user 1, got %d", len(list1))
	}

	// List reports for user 2
	list2, err := secStore.ListReports(ctx, 2, 10)
	if err != nil {
		t.Fatalf("failed to list reports for user 2: %v", err)
	}
	if len(list2) != 0 {
		t.Errorf("expected 0 reports for user 2, got %d", len(list2))
	}

	// Delete report
	if err := secStore.DeleteReport(ctx, report1.ID, 1); err != nil {
		t.Fatalf("failed to delete report: %v", err)
	}

	_, err = secStore.GetReport(ctx, report1.ID, 1)
	if err == nil {
		t.Errorf("expected report to be deleted, but found")
	}
}
