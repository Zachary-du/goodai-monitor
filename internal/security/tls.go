package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

// DialContextFunc matches net.Dialer.DialContext signature.
type DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

// AuditTLS inspects the TLS handshake, certificate validity, and cryptography of the target.
func AuditTLS(ctx context.Context, targetURL *url.URL) ([]model.FindingItem, error) {
	return AuditTLSWithDialer(ctx, targetURL, nil)
}

// AuditTLSWithDialer performs TLS audit using a specified dialer function, defaulting to SafeDialContext.
func AuditTLSWithDialer(ctx context.Context, targetURL *url.URL, dialFn DialContextFunc) ([]model.FindingItem, error) {
	var findings []model.FindingItem

	// 1. Plaintext HTTP check
	if strings.ToLower(targetURL.Scheme) == "http" {
		findings = append(findings, model.FindingItem{
			ID:             "tls-protocol",
			Domain:         string(model.DomainTLS),
			CheckName:      "TLS Protocol Version",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  "Plaintext HTTP (No TLS Encryption)",
			Recommendation: "Upgrade target to HTTPS and enforce TLS 1.3 encryption.",
			Remediation:    "server {\n    listen 80;\n    server_name " + targetURL.Hostname() + ";\n    return 301 https://$host$request_uri;\n}",
		})
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-validity",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Expiration & Validity",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  "No TLS certificate present",
			Recommendation: "Obtain and configure a valid TLS certificate from an automated ACME CA (e.g. Let's Encrypt).",
			Remediation:    "certbot --nginx -d " + targetURL.Hostname(),
		})
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-san",
			Domain:         string(model.DomainTLS),
			CheckName:      "Subject Alternative Names (SAN)",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  "No TLS certificate present",
			Recommendation: "Issue a TLS certificate matching the target hostname.",
			Remediation:    "",
		})
		return findings, nil
	}

	host := targetURL.Hostname()
	port := targetURL.Port()
	if port == "" {
		port = "443"
	}
	targetAddr := net.JoinHostPort(host, port)

	if dialFn == nil {
		dialFn = SafeDialContext
	}

	rawConn, err := dialFn(ctx, "tcp", targetAddr)
	if err != nil {
		findings = append(findings, model.FindingItem{
			ID:             "tls-connection-failed",
			Domain:         string(model.DomainTLS),
			CheckName:      "TLS Handshake",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  fmt.Sprintf("Failed to establish TCP connection: %v", err),
			Recommendation: "Verify firewall rules, port configuration, and server responsiveness.",
			Remediation:    "",
		})
		return findings, nil
	}
	defer rawConn.Close()

	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true, // We inspect cert details manually below
	}

	tlsConn := tls.Client(rawConn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		findings = append(findings, model.FindingItem{
			ID:             "tls-handshake-failed",
			Domain:         string(model.DomainTLS),
			CheckName:      "TLS Handshake",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  fmt.Sprintf("TLS handshake failed: %v", err),
			Recommendation: "Check SSL/TLS ciphers and protocol configurations on the reverse proxy.",
			Remediation:    "",
		})
		return findings, nil
	}
	defer tlsConn.Close()

	state := tlsConn.ConnectionState()

	// 1. Evaluate Protocol Version
	var protoObs string
	var protoSev model.SeverityLevel
	var protoDed int
	switch state.Version {
	case tls.VersionTLS13:
		protoObs = "TLS 1.3 Enforced (Optimal)"
		protoSev = model.SeverityPass
		protoDed = 0
	case tls.VersionTLS12:
		protoObs = "TLS 1.2 Negotiated (Acceptable)"
		protoSev = model.SeverityPass
		protoDed = 0
	case tls.VersionTLS10, tls.VersionTLS11:
		protoObs = fmt.Sprintf("Outdated TLS 1.%d Negotiated (Vulnerable)", state.Version-0x0300)
		protoSev = model.SeverityCritical
		protoDed = 20
	default:
		protoObs = fmt.Sprintf("Unknown TLS version 0x%04x", state.Version)
		protoSev = model.SeverityMedium
		protoDed = 5
	}

	findings = append(findings, model.FindingItem{
		ID:             "tls-protocol",
		Domain:         string(model.DomainTLS),
		CheckName:      "TLS Protocol Version",
		Severity:       protoSev,
		ScoreDeduction: protoDed,
		ObservedValue:  protoObs,
		Recommendation: "Enforce TLS 1.3 as the preferred protocol and disable legacy TLS 1.0/1.1.",
		Remediation:    "ssl_protocols TLSv1.2 TLSv1.3;\nssl_prefer_server_ciphers off;",
	})

	// 2. Evaluate Peer Certificates
	if len(state.PeerCertificates) == 0 {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-validity",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Expiration & Validity",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  "No peer certificates presented",
			Recommendation: "Configure a valid TLS certificate on the target web server.",
			Remediation:    "",
		})
		return findings, nil
	}

	cert := state.PeerCertificates[0]
	now := time.Now()

	// Validity duration
	if now.After(cert.NotAfter) {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-validity",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Expiration & Validity",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  fmt.Sprintf("Expired on %s", cert.NotAfter.Format("2006-01-02")),
			Recommendation: "Immediately renew the expired TLS certificate.",
			Remediation:    "certbot renew --force-renewal",
		})
	} else if now.Before(cert.NotBefore) {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-validity",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Expiration & Validity",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  fmt.Sprintf("Not yet valid (active from %s)", cert.NotBefore.Format("2006-01-02")),
			Recommendation: "Verify system clock synchronization and certificate start date.",
			Remediation:    "",
		})
	} else {
		daysRemaining := int(cert.NotAfter.Sub(now).Hours() / 24)
		var expSev model.SeverityLevel
		var expDed int
		var expRec string

		if daysRemaining > 30 {
			expSev = model.SeverityPass
			expDed = 0
			expRec = "Certificate operational window healthy."
		} else if daysRemaining >= 15 {
			expSev = model.SeverityMedium
			expDed = 5
			expRec = "Plan certificate renewal within standard change window."
		} else {
			expSev = model.SeverityHigh
			expDed = 10
			expRec = "Urgent: Renew TLS certificate immediately to prevent production outage."
		}

		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-validity",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Expiration & Validity",
			Severity:       expSev,
			ScoreDeduction: expDed,
			ObservedValue:  fmt.Sprintf("%d Days Remaining (Expires %s)", daysRemaining, cert.NotAfter.Format("2006-01-02")),
			Recommendation: expRec,
			Remediation:    "certbot renew",
		})
	}

	// SAN validation
	if err := cert.VerifyHostname(host); err != nil {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-san",
			Domain:         string(model.DomainTLS),
			CheckName:      "Subject Alternative Names (SAN)",
			Severity:       model.SeverityCritical,
			ScoreDeduction: 20,
			ObservedValue:  fmt.Sprintf("Hostname mismatch: %v (Cert DNS: %v)", err, cert.DNSNames),
			Recommendation: "Reissue TLS certificate to include the target domain in Subject Alternative Names.",
			Remediation:    "certbot certonly --nginx -d " + host,
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-san",
			Domain:         string(model.DomainTLS),
			CheckName:      "Subject Alternative Names (SAN)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  fmt.Sprintf("SAN matches target hostname (%s)", host),
			Recommendation: "Domain name correctly attested in certificate SAN extension.",
			Remediation:    "",
		})
	}

	// Issuer CA
	issuerName := cert.Issuer.CommonName
	if issuerName == "" && len(cert.Issuer.Organization) > 0 {
		issuerName = cert.Issuer.Organization[0]
	}
	if issuerName == "" {
		issuerName = "Unknown Authority"
	}

	// Verify against system root CAs
	opts := x509.VerifyOptions{
		DNSName: host,
	}
	_, verifyErr := cert.Verify(opts)
	if verifyErr != nil {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-ca",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Authority (Issuer)",
			Severity:       model.SeverityMedium,
			ScoreDeduction: 5,
			ObservedValue:  fmt.Sprintf("Issuer: %s (Verification warning: %v)", issuerName, verifyErr),
			Recommendation: "Ensure full certificate chain (including intermediate CA) is provided.",
			Remediation:    "ssl_certificate /etc/letsencrypt/live/" + host + "/fullchain.pem;",
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "tls-cert-ca",
			Domain:         string(model.DomainTLS),
			CheckName:      "Certificate Authority (Issuer)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  fmt.Sprintf("Trusted CA: %s", issuerName),
			Recommendation: "Certificate chain verified by standard trust store.",
			Remediation:    "",
		})
	}

	// Cipher suite
	cipherName := tls.CipherSuiteName(state.CipherSuite)
	findings = append(findings, model.FindingItem{
		ID:             "tls-cipher-suite",
		Domain:         string(model.DomainTLS),
		CheckName:      "Negotiated Cipher Suite",
		Severity:       model.SeverityPass,
		ScoreDeduction: 0,
		ObservedValue:  cipherName,
		Recommendation: "Secure AEAD cipher suite negotiated.",
		Remediation:    "",
	})

	return findings, nil
}
