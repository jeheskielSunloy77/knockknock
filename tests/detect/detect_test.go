package detect_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"knockknock/internal/detect"
	"knockknock/internal/signature"
)

func TestNormalizeRedirectChainPreservesHeaders(t *testing.T) {
	raw := "HTTP/2 302 Found\r\nSet-Cookie: _abck=1\r\nLocation: /home\r\n\r\n" +
		"redirecting\r\nHTTP/2 200 OK\r\nContent-Type: text/html\r\n\r\n<html>home</html>"

	got := detect.Normalize([]byte(raw), detect.DefaultBodyCap)
	for _, want := range []string{
		"S:302",
		"S:200",
		"H:set-cookie:_abck=1",
		"H:location:/home",
		"B:<html>home</html>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Normalize() missing %q in:\n%s", want, got)
		}
	}
}

func TestMatchReturnsSortedNamedGroups(t *testing.T) {
	re := regexp.MustCompile(`(?m)(?P<z_vendor>B:z)|(?P<a_vendor>H:x-a:1)`)
	raw := []byte("HTTP/1.1 200 OK\r\nX-A: 1\r\n\r\nz")
	got := detect.Match(raw, re, detect.DefaultBodyCap)
	want := []string{"a_vendor", "z_vendor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %v, want %v", got, want)
	}
}

func TestDetectInvalidRegex(t *testing.T) {
	if _, err := detect.Detect([]byte("body"), "(", detect.DefaultBodyCap); err == nil {
		t.Fatal("Detect() error = nil, want invalid regex error")
	}
}

func TestEmbeddedArtifactsInSync(t *testing.T) {
	pattern, err := signature.Compile("../../signatures", "")
	if err != nil {
		t.Fatalf("Compile(): %v", err)
	}
	if strings.TrimSpace(detect.PresenceRegexText) != pattern {
		t.Fatal("regex_presence.re2.txt is stale; run: go run ./cmd/knockknock compile")
	}

	challengePattern, err := signature.CompileChallenge("../../signatures", "")
	if err != nil {
		t.Fatalf("CompileChallenge(): %v", err)
	}
	if strings.TrimSpace(detect.ChallengeRegexText) != challengePattern {
		t.Fatal("regex_challenge.re2.txt is stale; run: go run ./cmd/knockknock compile")
	}
}
