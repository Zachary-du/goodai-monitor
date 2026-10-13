package alert

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

// EmailSender defines the interface for dispatching notification emails.
type EmailSender interface {
	SendOutageAlert(ctx context.Context, recipient string, m *model.Monitor, inc *model.Incident) error
	SendRecoveryAlert(ctx context.Context, recipient string, m *model.Monitor, inc *model.Incident) error
}

// SMTPConfig holds configuration for the SMTP relay.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPEmailSender dispatches notification emails via an external SMTP server.
type SMTPEmailSender struct {
	cfg SMTPConfig
}

// NewSMTPEmailSender creates an SMTPEmailSender with the provided configuration.
func NewSMTPEmailSender(cfg SMTPConfig) *SMTPEmailSender {
	return &SMTPEmailSender{cfg: cfg}
}

// IsConfigured returns true if minimum SMTP parameters (Host and From) are set.
func (s *SMTPEmailSender) IsConfigured() bool {
	return s.cfg.Host != "" && s.cfg.From != ""
}

// SendOutageAlert dispatches an outage alert email when a monitor downtime is confirmed.
func (s *SMTPEmailSender) SendOutageAlert(ctx context.Context, recipient string, m *model.Monitor, inc *model.Incident) error {
	if !s.IsConfigured() {
		log.Printf("[INFO] SMTP alerting unconfigured; skipping outage alert for monitor '%s' to %s", m.Name, maskEmail(recipient))
		return nil
	}

	subject := fmt.Sprintf("[ALERT] Monitor Down: %s", m.Name)
	timeFormatted := formatDualTime(inc.StartedAt)

	body := fmt.Sprintf(
		"GoodAI Monitor Outage Alert\r\n"+
			"--------------------------------------------------\r\n"+
			"Monitor Name : %s\r\n"+
			"Target URL   : %s\r\n"+
			"Status       : DOWN\r\n"+
			"Incident ID  : #%d\r\n"+
			"Trigger Time : %s\r\n"+
			"Root Cause   : %s\r\n"+
			"--------------------------------------------------\r\n"+
			"Automated probe confirmed downtime after 2 consecutive failed checks.\r\n"+
			"Our system will alert you as soon as service is restored.\r\n",
		m.Name,
		m.URL,
		inc.ID,
		timeFormatted,
		inc.Cause,
	)

	return s.sendMail(recipient, subject, body)
}

// SendRecoveryAlert dispatches a recovery notification email when a monitor successfully recovers.
func (s *SMTPEmailSender) SendRecoveryAlert(ctx context.Context, recipient string, m *model.Monitor, inc *model.Incident) error {
	if !s.IsConfigured() {
		log.Printf("[INFO] SMTP alerting unconfigured; skipping recovery alert for monitor '%s' to %s", m.Name, maskEmail(recipient))
		return nil
	}

	subject := fmt.Sprintf("[RESOLVED] Monitor Recovered: %s", m.Name)
	resolvedAt := time.Now().UTC()
	if inc.ResolvedAt != nil {
		resolvedAt = *inc.ResolvedAt
	}
	timeFormatted := formatDualTime(resolvedAt)
	durationFormatted := formatDuration(inc.DurationSeconds)

	body := fmt.Sprintf(
		"GoodAI Monitor Recovery Notification\r\n"+
			"--------------------------------------------------\r\n"+
			"Monitor Name   : %s\r\n"+
			"Target URL     : %s\r\n"+
			"Status         : RESOLVED (UP)\r\n"+
			"Incident ID    : #%d\r\n"+
			"Recovered Time : %s\r\n"+
			"Total Downtime : %s\r\n"+
			"Initial Cause  : %s\r\n"+
			"--------------------------------------------------\r\n"+
			"Automated probe confirmed recovery after 2 consecutive successful checks.\r\n",
		m.Name,
		m.URL,
		inc.ID,
		timeFormatted,
		durationFormatted,
		inc.Cause,
	)

	return s.sendMail(recipient, subject, body)
}

// sendMail connects to the configured SMTP relay and sends an email.
func (s *SMTPEmailSender) sendMail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	// Build RFC 5322 compliant message headers
	header := make(map[string]string)
	header["From"] = s.cfg.From
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=UTF-8"
	header["Date"] = time.Now().Format(time.RFC1123Z)

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	var auth smtp.Auth
	if s.cfg.Username != "" && s.cfg.Password != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	maskedRecipient := maskEmail(to)

	// Direct SSL/TLS on port 465
	if s.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: s.cfg.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("[WARN] SMTP TLS dial failed to %s: %v", s.cfg.Host, err)
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			log.Printf("[WARN] SMTP client initialization failed: %v", err)
			return err
		}
		defer client.Quit()

		if auth != nil {
			if ok, _ := client.Extension("AUTH"); ok {
				if err := client.Auth(auth); err != nil {
					log.Printf("[WARN] SMTP AUTH failed on %s: %v", s.cfg.Host, err)
					return err
				}
			}
		}

		if err := client.Mail(s.cfg.From); err != nil {
			return err
		}
		if err := client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(msg.String())); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}

		log.Printf("[INFO] Dispatched alert email to %s via SMTP TLS :465", maskedRecipient)
		return nil
	}

	// Standard STARTTLS connection (ports 587, 25, or custom)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		log.Printf("[WARN] SMTP TCP dial failed to %s: %v", addr, err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		log.Printf("[WARN] SMTP client creation failed: %v", err)
		return err
	}
	defer client.Quit()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: s.cfg.Host}
		if err := client.StartTLS(tlsConfig); err != nil {
			log.Printf("[WARN] SMTP STARTTLS handshake failed: %v", err)
			return err
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				log.Printf("[WARN] SMTP AUTH failed on %s: %v", s.cfg.Host, err)
				return err
			}
		}
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	log.Printf("[INFO] Dispatched alert email to %s via SMTP STARTTLS", maskedRecipient)
	return nil
}

// formatDualTime formats a timestamp in both Pacific/Auckland (NZDT) and UTC.
func formatDualTime(t time.Time) string {
	loc, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		loc = time.FixedZone("NZDT", 13*3600)
	}
	nzTime := t.In(loc).Format("2006-01-02 15:04:05 MST")
	utcTime := t.UTC().Format("2006-01-02 15:04:05 MST")
	return fmt.Sprintf("%s (%s)", nzTime, utcTime)
}

// formatDuration converts seconds into a human-readable duration string.
func formatDuration(sec int64) string {
	if sec <= 0 {
		return "0s"
	}
	d := time.Duration(sec) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// maskEmail masks recipient emails in log statements to preserve user privacy.
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	user, domain := parts[0], parts[1]
	if len(user) <= 2 {
		return user[:1] + "***@" + domain
	}
	return user[:2] + "***@" + domain
}
