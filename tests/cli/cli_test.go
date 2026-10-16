package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeheskielSunloy77/knockknock/internal/cli"
)

func TestRunDetectsFromStdinReadable(t *testing.T) {
	var out, stderr bytes.Buffer
	code := cli.Run(cli.Config{
		Args:   nil,
		Stdin:  strings.NewReader("HTTP/1.1 403\r\nSet-Cookie: __cf_bm=abc; path=/\r\n\r\nblocked"),
		Stdout: &out,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "who's there?") || !strings.Contains(out.String(), "cloudflare") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestRunNoDetectionExitOne(t *testing.T) {
	var out, stderr bytes.Buffer
	code := cli.Run(cli.Config{
		Stdin:  strings.NewReader("HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\nhello"),
		Stdout: &out,
		Stderr: &stderr,
	})
	if code != 1 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "who's there?") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunJSON(t *testing.T) {
	var out, stderr bytes.Buffer
	code := cli.Run(cli.Config{
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
	code := cli.Run(cli.Config{Args: []string{"--bad"}, Stderr: &stderr})
	if code != 2 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown option") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunSkillShow(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run(cli.Config{
		Args:   []string{"skill", "show"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "name: knockknock") {
		t.Fatalf("unexpected stdout:\n%s", stdout.String())
	}
}

func TestRunSkillInstall(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := cli.Run(cli.Config{
		Args:   []string{"skill", "install", "--dir", tmpDir},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("reading installed skill: %v", err)
	}
	if !strings.Contains(string(content), "name: knockknock") {
		t.Fatalf("invalid skill content:\n%s", string(content))
	}
}

func TestRunSkillUnknown(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run(cli.Config{
		Args:   []string{"skill", "nonexistent"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

