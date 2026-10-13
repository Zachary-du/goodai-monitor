package security

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestIsRestrictedIP(t *testing.T) {
	tests := []struct {
		name       string
		ipStr      string
		restricted bool
	}{
		// IPv4 Loopback
		{"IPv4 Loopback 127.0.0.1", "127.0.0.1", true},
		{"IPv4 Loopback 127.0.0.2", "127.0.0.2", true},
		{"IPv4 Loopback 127.255.255.254", "127.255.255.254", true},

		// Cloud Metadata / Link-Local (Oracle IMDS / AWS IMDS)
		{"Oracle IMDS 169.254.169.254", "169.254.169.254", true},
		{"Link-Local 169.254.1.1", "169.254.1.1", true},

		// RFC 1918 Private
		{"Class A Private 10.0.0.1", "10.0.0.1", true},
		{"Class A Private 10.254.10.1", "10.254.10.1", true},
		{"Class B Private 172.16.0.1", "172.16.0.1", true},
		{"Class B Private 172.31.255.255", "172.31.255.255", true},
		{"Class C Private 192.168.1.1", "192.168.1.1", true},
		{"Class C Private 192.168.0.100", "192.168.0.100", true},

		// Carrier-Grade NAT
		{"CGNAT 100.64.0.1", "100.64.0.1", true},

		// Broadcast & Unspecified
		{"Unspecified 0.0.0.0", "0.0.0.0", true},
		{"Broadcast 255.255.255.255", "255.255.255.255", true},

		// IPv6 Loopback & Link-Local & Private
		{"IPv6 Loopback ::1", "::1", true},
		{"IPv6 Link-Local fe80::1", "fe80::1", true},
		{"IPv6 Unique Local fc00::1", "fc00::1", true},
		{"IPv6 Unique Local fd00::1", "fd00::1", true},
		{"IPv6 Unspecified ::", "::", true},

		// IPv4-mapped IPv6
		{"IPv4-mapped Loopback ::ffff:127.0.0.1", "::ffff:127.0.0.1", true},
		{"IPv4-mapped Oracle IMDS ::ffff:169.254.169.254", "::ffff:169.254.169.254", true},
		{"IPv4-mapped Private ::ffff:10.0.0.1", "::ffff:10.0.0.1", true},
		{"IPv4-mapped Private ::ffff:192.168.1.1", "::ffff:192.168.1.1", true},

		// Public routable IPs (Should NOT be restricted)
		{"Google DNS 8.8.8.8", "8.8.8.8", false},
		{"Cloudflare DNS 1.1.1.1", "1.1.1.1", false},
		{"Example.com Public IP 93.184.216.34", "93.184.216.34", false},
		{"Google Public IPv6 2001:4860:4860::8888", "2001:4860:4860::8888", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ipStr)
			if ip == nil {
				t.Fatalf("failed to parse test IP %s", tc.ipStr)
			}
			res := IsRestrictedIP(ip)
			if res != tc.restricted {
				t.Errorf("IsRestrictedIP(%s) = %v, expected %v", tc.ipStr, res, tc.restricted)
			}
		})
	}
}

func TestValidateTargetURL_RestrictedDestinations(t *testing.T) {
	restrictedURLs := []string{
		// AC-2 Acceptance Criteria explicit test targets
		"http://127.0.0.1",
		"http://127.0.0.1:80",
		"http://localhost",
		"http://localhost:8080",
		"http://169.254.169.254/opc/v2/instance/",
		"http://169.254.169.254:80",
		"http://10.0.0.1",
		"http://192.168.1.1",
		"http://172.16.0.1",

		// Additional loopbacks and link-locals
		"http://127.0.0.2:80",
		"https://127.0.0.1:8443",
		"http://[::1]",
		"http://[::1]:80",
		"http://[fe80::1]",
		"http://[fe80::1]:80",
		"http://[::ffff:127.0.0.1]",
		"http://[::ffff:169.254.169.254]",
		"http://0.0.0.0",
		"http://255.255.255.255",

		// AC-3 Forbidden port numbers
		"http://example.com:60324",
		"http://example.com:5244",
		"http://example.com:22",
		"http://example.com:8088",
		"http://example.com:2019",
		"http://example.com:3000",
		"http://example.com:65535",

		// Disallowed schemes
		"ftp://example.com",
		"file:///etc/passwd",
		"gopher://127.0.0.1:70",
		"ssh://127.0.0.1:22",
		"javascript:alert(1)",
		"",
		"http://",
	}

	for _, u := range restrictedURLs {
		t.Run("Block_"+u, func(t *testing.T) {
			err := ValidateTargetURL(u)
			if err == nil {
				t.Errorf("expected ValidateTargetURL(%q) to be rejected, but passed", u)
			}
			if !errors.Is(err, ErrRestrictedTargetURL) {
				t.Errorf("expected ErrRestrictedTargetURL, got: %v", err)
			}
		})
	}
}

func TestValidateTargetURL_AllowedURLs(t *testing.T) {
	allowedURLs := []string{
		"http://example.com",
		"https://example.com",
		"http://example.com:80",
		"https://example.com:443",
		"http://example.com:8080",
		"https://example.com:8443",
		"https://google.com",
		"https://cloudflare.com",
		"http://93.184.216.34",      // Public IP literal for example.com
		"http://93.184.216.34:80",   // Port 80
		"http://93.184.216.34:8080", // Port 8080
	}

	for _, u := range allowedURLs {
		t.Run("Allow_"+u, func(t *testing.T) {
			err := ValidateTargetURL(u)
			if err != nil {
				t.Errorf("expected ValidateTargetURL(%q) to pass, but got error: %v", u, err)
			}
		})
	}
}

func TestSafeDialer_RestrictedAddressRejection(t *testing.T) {
	ctx := context.Background()

	// Direct socket dial to private loopback should fail
	_, err := SafeDialContext(ctx, "tcp", "127.0.0.1:80")
	if err == nil {
		t.Fatalf("expected SafeDialContext to block 127.0.0.1:80")
	}

	// Direct socket dial to Oracle IMDS should fail
	_, err = SafeDialContext(ctx, "tcp", "169.254.169.254:80")
	if err == nil {
		t.Fatalf("expected SafeDialContext to block 169.254.169.254:80")
	}

	// Dial to restricted port should fail
	_, err = SafeDialContext(ctx, "tcp", "93.184.216.34:22")
	if err == nil {
		t.Fatalf("expected SafeDialContext to block port 22")
	}

	_, err = SafeDialContext(ctx, "tcp", "93.184.216.34:60324")
	if err == nil {
		t.Fatalf("expected SafeDialContext to block port 60324")
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	client := NewSafeHTTPClient(2 * time.Second)
	if client == nil {
		t.Fatalf("expected non-nil safe HTTP client")
	}
	if client.CheckRedirect == nil {
		t.Fatalf("expected CheckRedirect to be configured")
	}
}
