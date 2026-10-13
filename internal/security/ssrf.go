package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// ErrRestrictedTargetURL is returned when a target URL fails SSRF security validation.
	ErrRestrictedTargetURL = errors.New("Invalid or restricted target URL")
)

// AllowedPorts defines standard web ports permitted for monitoring targets.
var AllowedPorts = map[int]bool{
	80:   true,
	443:  true,
	8080: true,
	8443: true,
}

// restrictedIPv4Nets and restrictedIPv6Nets contain pre-parsed CIDR network ranges that must never be targeted.
var (
	restrictedIPv4Nets []*net.IPNet
	restrictedIPv6Nets []*net.IPNet
)

func init() {
	ipv4CIDRs := []string{
		// IPv4 Private and Reserved
		"0.0.0.0/8",          // "This network" (RFC 1122)
		"10.0.0.0/8",         // Private network (RFC 1918)
		"100.64.0.0/10",      // Shared Address Space (RFC 6598)
		"127.0.0.0/8",        // Loopback (RFC 1122)
		"169.254.0.0/16",     // Link-Local / Cloud IMDS (RFC 3927)
		"172.16.0.0/12",      // Private network (RFC 1918)
		"192.0.0.0/24",       // IETF Protocol Assignments (RFC 6890)
		"192.0.2.0/24",       // TEST-NET-1 (RFC 5737)
		"192.88.99.0/24",     // 6to4 Relay Anycast (RFC 7526)
		"192.168.0.0/16",     // Private network (RFC 1918)
		"198.18.0.0/15",      // Benchmarking (RFC 2544)
		"198.51.100.0/24",    // TEST-NET-2 (RFC 5737)
		"203.0.113.0/24",     // TEST-NET-3 (RFC 5737)
		"224.0.0.0/4",        // Multicast (RFC 5771)
		"240.0.0.0/4",        // Reserved for Future Use (RFC 1112)
		"255.255.255.255/32", // Limited Broadcast (RFC 919)
	}

	ipv6CIDRs := []string{
		// IPv6 Private and Reserved
		"::/128",        // Unspecified
		"::1/128",       // Loopback
		"100::/64",      // Discard prefix (RFC 6666)
		"2001:db8::/32", // Documentation (RFC 3849)
		"fc00::/7",      // Unique Local Address (RFC 4193)
		"fe80::/10",     // Link-Local Unicast (RFC 4291)
		"ff00::/8",      // Multicast (RFC 4291)
	}

	for _, cidr := range ipv4CIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet != nil {
			restrictedIPv4Nets = append(restrictedIPv4Nets, ipNet)
		}
	}

	for _, cidr := range ipv6CIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet != nil {
			restrictedIPv6Nets = append(restrictedIPv6Nets, ipNet)
		}
	}
}

// IsRestrictedIP checks whether an IP address belongs to loopback, private, link-local,
// cloud metadata (Oracle/AWS/GCP IMDS), multicast, or reserved ranges.
func IsRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Standard Go net.IP methods
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	// If it's an IPv4 address or IPv4-mapped IPv6, evaluate as IPv4
	if ip4 := ip.To4(); ip4 != nil {
		// Specific Oracle Cloud / Cloud Metadata IP check
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// Additional IPv4 checks
		if ip4.IsLoopback() || ip4.IsPrivate() || ip4.IsLinkLocalUnicast() {
			return true
		}
		// Test against IPv4 CIDRs
		for _, ipNet := range restrictedIPv4Nets {
			if ipNet.Contains(ip4) {
				return true
			}
		}
		return false
	}

	// Test against IPv6 CIDRs
	for _, ipNet := range restrictedIPv6Nets {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateTargetURL parses and strictly validates a target URL against SSRF vulnerabilities.
// It verifies:
// 1. Allowed schemes (http, https).
// 2. Allowed ports (80, 443, 8080, 8443).
// 3. Rejects localhost and internal hostnames.
// 4. Resolves hostname and validates all resolved IP addresses against restricted ranges.
func ValidateTargetURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ErrRestrictedTargetURL
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return ErrRestrictedTargetURL
	}

	// Enforce HTTP / HTTPS scheme
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrRestrictedTargetURL
	}

	// Extract and validate hostname
	host := parsed.Hostname()
	if host == "" {
		return ErrRestrictedTargetURL
	}

	cleanHost := strings.ToLower(strings.TrimSuffix(host, "."))
	if cleanHost == "localhost" || strings.HasSuffix(cleanHost, ".localhost") {
		return ErrRestrictedTargetURL
	}

	// Port validation
	portStr := parsed.Port()
	var port int
	if portStr == "" {
		if scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	} else {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return ErrRestrictedTargetURL
		}
	}

	if !AllowedPorts[port] {
		return ErrRestrictedTargetURL
	}

	// IP resolution and validation
	if ip := net.ParseIP(cleanHost); ip != nil {
		if IsRestrictedIP(ip) {
			return ErrRestrictedTargetURL
		}
		return nil
	}

	// Domain name DNS resolution check
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", cleanHost)
	if err != nil {
		return ErrRestrictedTargetURL
	}

	if len(ips) == 0 {
		return ErrRestrictedTargetURL
	}

	for _, resolvedIP := range ips {
		if IsRestrictedIP(resolvedIP) {
			return ErrRestrictedTargetURL
		}
	}

	return nil
}

// NewSafeDialer creates a standard net.Dialer configured with a socket Control callback
// that blocks TCP connections to restricted IP addresses and forbidden ports at connect time.
func NewSafeDialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, portStr, err := net.SplitHostPort(address)
			if err != nil {
				return ErrRestrictedTargetURL
			}

			port, err := strconv.Atoi(portStr)
			if err != nil || !AllowedPorts[port] {
				return fmt.Errorf("%w: port %s is restricted", ErrRestrictedTargetURL, portStr)
			}

			ip := net.ParseIP(host)
			if ip != nil && IsRestrictedIP(ip) {
				return fmt.Errorf("%w: target ip %s is restricted", ErrRestrictedTargetURL, host)
			}

			return nil
		},
	}
}

// SafeDialContext resolves the target hostname, validates resolved IPs against restricted ranges,
// and connects to the verified IP directly to eliminate DNS Rebinding (TOCTOU) risks.
func SafeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, ErrRestrictedTargetURL
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || !AllowedPorts[port] {
		return nil, fmt.Errorf("%w: port %d is not allowed", ErrRestrictedTargetURL, port)
	}

	dialer := NewSafeDialer(10 * time.Second)

	// If already an IP address
	if ip := net.ParseIP(host); ip != nil {
		if IsRestrictedIP(ip) {
			return nil, ErrRestrictedTargetURL
		}
		return dialer.DialContext(ctx, network, address)
	}

	// Resolve hostname
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("%w: lookup failed: %v", ErrRestrictedTargetURL, err)
	}

	if len(ips) == 0 {
		return nil, ErrRestrictedTargetURL
	}

	for _, resolvedIP := range ips {
		if IsRestrictedIP(resolvedIP) {
			return nil, fmt.Errorf("%w: resolved to restricted IP %s", ErrRestrictedTargetURL, resolvedIP)
		}
	}

	// Dial the first verified IP directly
	targetAddr := net.JoinHostPort(ips[0].String(), portStr)
	return dialer.DialContext(ctx, network, targetAddr)
}

// NewSafeHTTPClient creates a secure HTTP client configured with SSRF defense:
// 1. SafeDialContext socket dialer preventing connections to internal IPs and restricted ports.
// 2. Redirect blocking (http.ErrUseLastResponse) preventing open-redirect SSRF bypasses.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		DialContext:           SafeDialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Disallow redirects to prevent target servers redirecting to internal infrastructure
			return http.ErrUseLastResponse
		},
	}
}
