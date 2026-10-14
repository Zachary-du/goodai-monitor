package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

// AuditRunner defines the interface for running security audits.
type AuditRunner interface {
	RunAudit(ctx context.Context, rawURL string, userID int64) (*model.SecurityAuditReport, error)
}

// StandardAuditEngine is the production orchestrator for multi-vector security audits.
type StandardAuditEngine struct {
	DialFunc DialContextFunc
	Timeout  time.Duration
}

// NewAuditEngine creates a new StandardAuditEngine.
func NewAuditEngine() *StandardAuditEngine {
	return &StandardAuditEngine{
		Timeout: 6 * time.Second,
	}
}

// RunAudit is the convenience package function using the default audit engine.
func RunAudit(ctx context.Context, rawURL string, userID int64) (*model.SecurityAuditReport, error) {
	return NewAuditEngine().RunAudit(ctx, rawURL, userID)
}

// RunAudit validates SSRF, executes concurrent inspections across the 4 pillars, and computes the scorecard.
func (e *StandardAuditEngine) RunAudit(ctx context.Context, rawURL string, userID int64) (*model.SecurityAuditReport, error) {
	startTime := time.Now()

	// 1. SSRF and URL validation
	if err := ValidateTargetURL(rawURL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRestrictedTargetURL, err)
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid URL syntax", ErrRestrictedTargetURL)
	}

	host := parsedURL.Hostname()

	// 2. Resolve target IP
	targetIP := host
	if net.ParseIP(host) == nil {
		lookupCtx, lookupCancel := context.WithTimeout(ctx, 3*time.Second)
		ips, err := net.DefaultResolver.LookupIP(lookupCtx, "ip", host)
		lookupCancel()
		if err == nil && len(ips) > 0 {
			targetIP = ips[0].String()
		}
	}

	// Create scoped context with overall timeout
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	auditCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Safe HTTP client
	client := NewSafeHTTPClient(5 * time.Second)

	// 3. Concurrently run the 4 pillar probes
	var wg sync.WaitGroup
	wg.Add(4)

	var (
		mu       sync.Mutex
		findings []model.FindingItem
	)

	// Pillar 1: AI Governance
	go func() {
		defer wg.Done()
		items, err := AuditAIGovernance(auditCtx, client, parsedURL)
		if err == nil {
			mu.Lock()
			findings = append(findings, items...)
			mu.Unlock()
		}
	}()

	// Pillar 2: Security Headers
	go func() {
		defer wg.Done()
		items, err := AuditHeaders(auditCtx, client, parsedURL)
		if err == nil {
			mu.Lock()
			findings = append(findings, items...)
			mu.Unlock()
		}
	}()

	// Pillar 3: TLS Cryptography
	go func() {
		defer wg.Done()
		items, err := AuditTLSWithDialer(auditCtx, parsedURL, e.DialFunc)
		if err == nil {
			mu.Lock()
			findings = append(findings, items...)
			mu.Unlock()
		}
	}()

	// Pillar 4: Exposure Probes
	go func() {
		defer wg.Done()
		items, err := AuditExposures(auditCtx, client, parsedURL)
		if err == nil {
			mu.Lock()
			findings = append(findings, items...)
			mu.Unlock()
		}
	}()

	wg.Wait()

	// 4. Calculate domain scores, composite score, and grade
	compositeScore, grade, gradeLabel, domainScores := CalculateScores(findings)

	createdAt := time.Now().UTC()
	execMs := time.Since(startTime).Milliseconds()

	// 5. Generate cryptographic attestation hash
	hashPayload := fmt.Sprintf("%s|%s|%d|%s|%d|%s", parsedURL.String(), targetIP, compositeScore, grade, userID, createdAt.Format(time.RFC3339Nano))
	hashBytes := sha256.Sum256([]byte(hashPayload))
	attestationHash := hex.EncodeToString(hashBytes[:])

	report := &model.SecurityAuditReport{
		UserID:         userID,
		TargetURL:      parsedURL.String(),
		TargetHost:     host,
		TargetIP:       targetIP,
		CompositeScore: compositeScore,
		Grade:          grade,
		GradeLabel:     gradeLabel,
		CreatedAt:      createdAt,
		ExecutionMs:    execMs,
		DomainScores:   domainScores,
		Findings:       findings,
		Attestation:    attestationHash,
	}

	return report, nil
}
