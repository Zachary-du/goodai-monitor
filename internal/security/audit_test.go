package security

import (
	"context"
	"errors"
	"testing"
)

func TestRunAudit_SSRFRejection(t *testing.T) {
	restrictedTargets := []string{
		"http://127.0.0.1",
		"http://127.0.0.1:8080/admin",
		"http://10.0.0.1",
		"http://192.168.1.1",
		"http://169.254.169.254/latest/meta-data/",
		"http://localhost:3000",
		"ftp://example.com",
	}

	for _, target := range restrictedTargets {
		_, err := RunAudit(context.Background(), target, 1)
		if err == nil {
			t.Errorf("expected target %s to be rejected by SSRF guard, but succeeded", target)
		} else if !errors.Is(err, ErrRestrictedTargetURL) {
			t.Errorf("expected ErrRestrictedTargetURL for %s, got: %v", target, err)
		}
	}
}

func TestStandardAuditEngine_Execution(t *testing.T) {
	// A mock test verifying engine initialization and options
	engine := NewAuditEngine()
	if engine == nil {
		t.Fatalf("expected non-nil engine")
	}
	if engine.Timeout <= 0 {
		t.Errorf("expected positive default timeout")
	}
}
