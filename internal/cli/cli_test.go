package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeheskielSunloy77/knockknock/internal/detect"
)

func TestBannerContainsName(t *testing.T) {
	if !strings.Contains(banner, "KNOCK") && !strings.Contains(banner, "██╗") {
		t.Fatal("banner missing ASCII art")
	}
}

func TestBannerContainsDescription(t *testing.T) {
	if !strings.Contains(banner, "detect antibot") {
		t.Fatal("banner missing description")
	}
}

func TestBannerContainsExamples(t *testing.T) {
	if !strings.Contains(banner, "example") {
		t.Fatal("banner missing examples section")
	}
}

func TestWriteVerdictProtected(t *testing.T) {
	var buf bytes.Buffer
	verdict := detect.Verdict{Protected: true, Vendors: []string{"cloudflare"}}
	err := writeVerdict(&buf, verdict, false)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "who's there?") {
		t.Fatal("expected 'who's there?' in output, got:", out)
	}
	if !strings.Contains(out, "cloudflare") {
		t.Fatal("expected vendor name in output, got:", out)
	}
}

func TestWriteVerdictNotProtected(t *testing.T) {
	var buf bytes.Buffer
	verdict := detect.Verdict{Protected: false}
	err := writeVerdict(&buf, verdict, false)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "who's there?") {
		t.Fatal("expected 'who's there?' in output, got:", out)
	}
}

func TestWriteVerdictJSON(t *testing.T) {
	var buf bytes.Buffer
	verdict := detect.Verdict{Protected: true, Vendors: []string{"cloudflare"}, Mode: "presence", Source: "stdin"}
	err := writeVerdict(&buf, verdict, true)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"protected": true`) {
		t.Fatal("expected JSON output, got:", out)
	}
	if strings.Contains(out, "Knock knock!") {
		t.Fatal("JSON output should not contain knock-knock joke")
	}
}

func TestFormatVendors(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{}, ""},
		{[]string{"cloudflare"}, "cloudflare"},
		{[]string{"cloudflare", "akamai"}, "cloudflare and akamai"},
		{[]string{"cloudflare", "akamai", "datadome"}, "cloudflare, akamai, and datadome"},
	}
	for _, tt := range tests {
		got := formatVendors(tt.input)
		if got != tt.want {
			t.Errorf("formatVendors(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSkillEmbeddedMatchesSource(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "skills", "knockknock", "SKILL.md")
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", sourcePath, err)
	}
	if string(content) != SkillContent {
		t.Fatalf("embedded SkillContent differs from %s", sourcePath)
	}
}

func TestSkillUsage(t *testing.T) {
	var stdout bytes.Buffer
	code := Run(Config{
		Args:   []string{"skill"},
		Stdout: &stdout,
	})
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "knockknock skill") {
		t.Fatalf("expected skill usage in stdout, got: %s", stdout.String())
	}
}

func TestSkillShow(t *testing.T) {
	var stdout bytes.Buffer
	code := Run(Config{
		Args:   []string{"skill", "show"},
		Stdout: &stdout,
	})
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "name: knockknock") {
		t.Fatalf("expected skill frontmatter in stdout, got: %s", stdout.String())
	}
}

func TestSkillInstallCustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	code := Run(Config{
		Args:   []string{"skill", "install", "--dir", tmpDir},
		Stdout: &stdout,
	})
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	installedFile := filepath.Join(tmpDir, "SKILL.md")
	data, err := os.ReadFile(installedFile)
	if err != nil {
		t.Fatalf("failed to read installed skill file: %v", err)
	}
	if string(data) != SkillContent {
		t.Fatalf("installed content does not match SkillContent")
	}
}

func TestInstallSkillAlias(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	code := Run(Config{
		Args:   []string{"install-skill", "--dir", tmpDir},
		Stdout: &stdout,
	})
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	installedFile := filepath.Join(tmpDir, "SKILL.md")
	data, err := os.ReadFile(installedFile)
	if err != nil {
		t.Fatalf("failed to read installed skill file: %v", err)
	}
	if string(data) != SkillContent {
		t.Fatalf("installed content does not match SkillContent")
	}
}

