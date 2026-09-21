package version

import (
	"testing"
)

// TestPublicKeepsOnlySafeFields guards Public(), the reduction every handler
// and middleware applies before serving. The endpoint policy it implements --
// and why the build details are withheld -- is documented with the handler
// tests in httpadapter; here the concern is narrower: Public() must drop every
// build detail, keep the two safe fields, and not mutate its receiver.
func TestPublicKeepsOnlySafeFields(t *testing.T) {
	full := NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")
	pub := full.Public()

	if pub.Version != "1.0.0" || pub.Branch != "main" {
		t.Errorf("Public() dropped a safe field: %+v", pub)
	}
	if pub.Commit != "" || pub.GoVersion != "" || pub.Platform != "" || pub.Compiler != "" || pub.BuildDate != "" {
		t.Errorf("Public() kept a build detail: %+v", pub)
	}
	// The original is untouched.
	if full.Commit != "abc123" {
		t.Error("Public() mutated its receiver")
	}
	if (*Info)(nil).Public() != nil {
		t.Error("Public() on nil should return nil")
	}
}

// --- Codex review follow-ups (PR #4) ---

// TestValidHeaderPrefixAcceptsTokenChars pins the validator to RFC 9110 5.6.2:
// a field name is a token, so every tchar is allowed -- not just the
// alphanumeric/dash subset. Prefixes such as "X.App-" were valid before this
// validator existed and must keep working. The end-to-end half of this lives
// in httpadapter, where the header is actually written.
func TestValidHeaderPrefixAcceptsTokenChars(t *testing.T) {
	valid := []string{"X-", "X.App-", "App_", "a~b+", "x^y-", "My!App-", "v1.2-"}
	for _, p := range valid {
		if !validHeaderPrefix(p) {
			t.Errorf("validHeaderPrefix(%q) = false, want true (all tchars)", p)
		}
	}

	invalid := []string{"", "X ", "X:", "X-\n", "X(", "X\"", "X@", "X/"}
	for _, p := range invalid {
		if validHeaderPrefix(p) {
			t.Errorf("validHeaderPrefix(%q) = true, want false (not a token)", p)
		}
	}
}
