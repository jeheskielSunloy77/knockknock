package fetch

import "testing"

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
		got, err := resolveLocation(tc.base, tc.loc)
		if err != nil {
			t.Fatalf("resolveLocation(%q, %q): %v", tc.base, tc.loc, err)
		}
		if got != tc.want {
			t.Fatalf("resolveLocation(%q, %q) = %q, want %q", tc.base, tc.loc, got, tc.want)
		}
	}
}

func TestHeaderGetCaseInsensitive(t *testing.T) {
	got := headerGet(map[string][]string{"location": {"https://example.com"}}, "Location")
	if got != "https://example.com" {
		t.Fatalf("headerGet() = %q", got)
	}
}

func TestIsRedirect(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		if !isRedirect(code) {
			t.Fatalf("isRedirect(%d) = false", code)
		}
	}
	if isRedirect(403) {
		t.Fatal("isRedirect(403) = true")
	}
}
