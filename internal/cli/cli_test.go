package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDetectsFromStdinReadable(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run(Config{
		Args:   nil,
		Stdin:  strings.NewReader("HTTP/1.1 403\r\nSet-Cookie: __cf_bm=abc; path=/\r\n\r\nblocked"),
		Stdout: &out,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "protected: yes") || !strings.Contains(out.String(), "cloudflare") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestRunNoDetectionExitOne(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run(Config{
		Stdin:  strings.NewReader("HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\nhello"),
		Stdout: &out,
		Stderr: &stderr,
	})
	if code != 1 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(out.String()) != "protected: no" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunJSON(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run(Config{
		Args:   []string{"--json"},
		Stdin:  strings.NewReader("HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n"),
		Stdout: &out,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(out.String(), `"protected": true`) || !strings.Contains(out.String(), `"cloudflare"`) {
		t.Fatalf("unexpected JSON:\n%s", out.String())
	}
}

func TestRunUnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(Config{Args: []string{"--bad"}, Stderr: &stderr})
	if code != 2 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown option") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
