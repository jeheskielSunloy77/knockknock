package signature_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"knockknock/internal/signature"
)

func TestCompileFromRepositorySignatures(t *testing.T) {
	pattern, err := signature.Compile("../../signatures", "")
	if err != nil {
		t.Fatalf("Compile(): %v", err)
	}
	if _, err := regexp.Compile(pattern); err != nil {
		t.Fatalf("compiled pattern invalid: %v", err)
	}
}

func TestChallengeMustBeSignalSubset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vendorx.json")
	raw := `{"vendor":"vendorx","signals":["B:present"],"challenge":["B:missing"]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := signature.CompileChallenge(dir, ""); err == nil {
		t.Fatal("CompileChallenge() error = nil, want subset validation error")
	}
}
