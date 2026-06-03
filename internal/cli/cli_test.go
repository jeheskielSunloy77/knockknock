package cli

import (
	"strings"
	"testing"
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
