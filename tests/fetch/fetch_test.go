package fetch_test

import (
	"testing"

	"knockknock/internal/fetch"
)

func TestResolveLocation(t *testing.T) {
	cases := []struct {
		base string
		loc  string
		want string
	}{
		{"https://a.com/x", "https://b.com/y", "https://b.com/y"},
		{"https://a.com/x/y", "/z", "https://a.com/z"},
		{"https://a.com/x/y", "z", "https://a.com/x/z"},
		{"https://a.com/x", "//cdn.com/p", "https://cdn.com/p"},
	}
	for _, tc := range cases {
		got, err := fetch.ResolveLocation(tc.base, tc.loc)
		if err != nil {
			t.Fatalf("ResolveLocation(%q, %q): %v", tc.base, tc.loc, err)
		}
		if got != tc.want {
			t.Fatalf("ResolveLocation(%q, %q) = %q, want %q", tc.base, tc.loc, got, tc.want)
		}
	}
}

func TestHeaderGetCaseInsensitive(t *testing.T) {
	got := fetch.HeaderGet(map[string][]string{"location": {"https://example.com"}}, "Location")
	if got != "https://example.com" {
		t.Fatalf("HeaderGet() = %q", got)
	}
}

func TestIsRedirect(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		if !fetch.IsRedirect(code) {
			t.Fatalf("IsRedirect(%d) = false", code)
		}
	}
	if fetch.IsRedirect(403) {
		t.Fatal("IsRedirect(403) = true")
	}
}
