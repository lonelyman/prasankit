package smtp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/modules/email"
)

// mailpitSMTPPort is the host-side SMTP port for Mailpit (mapped from container port 1025).
// Reads MAIL_SMTP_EXTERNAL_PORT from env; falls back to 11025.
func mailpitSMTPPort() string {
	if v := os.Getenv("MAIL_SMTP_EXTERNAL_PORT"); v != "" {
		return v
	}
	return "11025"
}

// mailpitUIPort is the host-side HTTP API port for Mailpit (mapped from container port 8025).
// Reads MAILPIT_UI_EXTERNAL_PORT from env; falls back to 18025.
func mailpitUIPort() string {
	if v := os.Getenv("MAILPIT_UI_EXTERNAL_PORT"); v != "" {
		return v
	}
	return "18025"
}

// repoRoot returns the prasankit repository root (parent of app-api/).
func repoRoot() string {
	// This file is at app-api/internal/adapters/email/smtp/; root is 5 levels up.
	dir, _ := os.Getwd()
	parts := strings.Split(dir, string(os.PathSeparator))
	// Walk up until we find docker-compose.yml.
	for i := len(parts); i > 0; i-- {
		candidate := strings.Join(parts[:i], string(os.PathSeparator))
		if _, err := os.Stat(candidate + "/docker-compose.yml"); err == nil {
			return candidate
		}
	}
	return dir
}

// ensureMailpit checks that Mailpit SMTP port is reachable; if not, starts it.
func ensureMailpit(t *testing.T, smtpPort string) {
	t.Helper()

	addr := net.JoinHostPort("localhost", smtpPort)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err == nil {
		conn.Close()
		return // already up
	}

	t.Logf("Mailpit not reachable at %s; starting with docker compose...", addr)
	root := repoRoot()
	cmd := exec.Command("docker", "compose", "up", "-d", "prasankit-mailpit")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to start prasankit-mailpit: %v\n%s", err, out)
	}
	t.Logf("docker compose output: %s", out)

	// Wait up to 30s for the SMTP port.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("Mailpit SMTP port %s never became reachable after start", smtpPort)
}

// clearMailpit deletes all messages via the Mailpit API.
func clearMailpit(t *testing.T, uiPort string) {
	t.Helper()
	url := fmt.Sprintf("http://localhost:%s/api/v1/messages", uiPort)
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("clearMailpit: DELETE failed: %v (non-fatal)", err)
		return
	}
	resp.Body.Close()
}

// TestSMTPSender_SendsMessage is the integration test.
// It sends a real message through Mailpit and asserts arrival via the Mailpit API.
func TestSMTPSender_SendsMessage(t *testing.T) {
	smtpPort := mailpitSMTPPort()
	uiPort := mailpitUIPort()

	ensureMailpit(t, smtpPort)

	// Clean state before test.
	clearMailpit(t, uiPort)

	// Unique subject so we can verify the correct message arrived.
	subject := fmt.Sprintf("Prasankit test email %d", time.Now().UnixNano())

	sender := smtpadapter.New(smtpadapter.Config{
		Host:        "localhost",
		Port:        smtpPort,
		FromAddress: "test@prasankit.local",
		FromName:    "Prasankit Test",
		// No username/password — Mailpit dev has no auth.
	})

	ctx := context.Background()
	msg := email.Message{
		To:       "recipient@prasankit.local",
		Subject:  subject,
		TextBody: "Hello from the Prasankit SMTP integration test.",
	}

	if err := sender.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Allow brief propagation.
	time.Sleep(500 * time.Millisecond)

	// Assert via Mailpit API.
	apiURL := fmt.Sprintf("http://localhost:%s/api/v1/messages", uiPort)
	resp, err := http.Get(apiURL)
	if err != nil {
		t.Fatalf("Mailpit API GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Mailpit API returned status %d", resp.StatusCode)
	}

	// Decode Mailpit /api/v1/messages response.
	var result struct {
		Messages []struct {
			Subject string `json:"Subject"`
			To      []struct {
				Address string `json:"Address"`
			} `json:"To"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode Mailpit response: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatalf("expected at least 1 message in Mailpit, got 0")
	}

	found := false
	for _, m := range result.Messages {
		if m.Subject == subject {
			// Verify recipient.
			if len(m.To) == 0 {
				t.Errorf("message has no To recipients")
			} else if m.To[0].Address != msg.To {
				t.Errorf("expected To %q, got %q", msg.To, m.To[0].Address)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("message with subject %q not found in Mailpit (got %d messages)", subject, len(result.Messages))
	}

	t.Logf("Message arrived in Mailpit with subject %q — test passed", subject)

	// Teardown: clear Mailpit so it is empty after the test run.
	clearMailpit(t, uiPort)

	// Confirm empty.
	resp2, err := http.Get(apiURL)
	if err != nil {
		t.Logf("post-clear GET failed: %v (non-fatal)", err)
		return
	}
	defer resp2.Body.Close()
	var result2 struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err == nil && result2.Total != 0 {
		t.Logf("warning: Mailpit not fully empty after teardown (total=%d)", result2.Total)
	}
}
