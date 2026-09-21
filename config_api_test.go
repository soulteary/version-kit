package version

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"testing"
)

// The config helpers below became exported in the fiberadapter split so that an
// out-of-tree adapter reads the serving rules instead of restating them. That
// makes them API: an adapter that calls them on a config it did not normalize
// must not panic, and the reduction rule must hold whichever door it comes in
// through. The handlers all normalize first, so nothing in-tree reaches those
// paths -- these tests do.

func TestResolveConfigDefaultsWithoutArguments(t *testing.T) {
	cfg := ResolveConfig()

	if cfg.Info == nil {
		t.Fatal("ResolveConfig() left Info nil, want Default()")
	}
	if cfg.Info.Version != Version {
		t.Errorf("Info.Version = %q, want %q from the package variables", cfg.Info.Version, Version)
	}
	if cfg.HeaderPrefix != "X-" {
		t.Errorf("HeaderPrefix = %q, want X-", cfg.HeaderPrefix)
	}
}

func TestResolveConfigNormalizesFirstArgument(t *testing.T) {
	cfg := ResolveConfig(HandlerConfig{HeaderPrefix: "bogus prefix"})

	if cfg.Info == nil {
		t.Fatal("ResolveConfig left a nil Info, want Default()")
	}
	if cfg.HeaderPrefix != "X-" {
		t.Errorf("HeaderPrefix = %q, want the X- fallback for a non-token prefix", cfg.HeaderPrefix)
	}
}

func TestResolveConfigIgnoresExtraArguments(t *testing.T) {
	cfg := ResolveConfig(
		HandlerConfig{HeaderPrefix: "First-"},
		HandlerConfig{HeaderPrefix: "Second-"},
	)

	if cfg.HeaderPrefix != "First-" {
		t.Errorf("HeaderPrefix = %q, want First- (only the first config is read)", cfg.HeaderPrefix)
	}
}

func TestNormalizedLeavesTheCallersConfigAlone(t *testing.T) {
	original := HandlerConfig{HeaderPrefix: "bogus prefix"}

	_ = original.Normalized()

	if original.Info != nil {
		t.Error("Normalized() filled in Info on the caller's config")
	}
	if original.HeaderPrefix != "bogus prefix" {
		t.Errorf("Normalized() rewrote the caller's HeaderPrefix to %q", original.HeaderPrefix)
	}
}

// A zero HandlerConfig is what an adapter gets from a caller who filled in
// nothing. Payload and TextPayload have to stand on their own there, because an
// adapter may reach for them without going through ResolveConfig.
func TestPayloadOnZeroConfig(t *testing.T) {
	info := HandlerConfig{}.Payload()

	if info == nil {
		t.Fatal("Payload() on a zero config returned nil")
	}
	if info.Version != Version {
		t.Errorf("Version = %q, want %q from Default()", info.Version, Version)
	}
	if info.GoVersion != "" {
		t.Errorf("GoVersion = %q, want it withheld without IncludeBuildDetails", info.GoVersion)
	}
}

func TestTextPayloadOnZeroConfig(t *testing.T) {
	text := HandlerConfig{}.TextPayload()

	if !strings.Contains(text, "Version:") {
		t.Errorf("TextPayload() on a zero config = %q, want it to render Default()", text)
	}
	if strings.Contains(text, "Go version:") {
		t.Errorf("TextPayload() leaked the Go runtime version without IncludeBuildDetails:\n%s", text)
	}
}

func TestPayloadReducesUnlessBuildDetailsAreAsked(t *testing.T) {
	info := NewWithBranch("1.2.3", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	reduced := HandlerConfig{Info: info}.Payload()
	if reduced.Commit != "" || reduced.BuildDate != "" || reduced.GoVersion != "" {
		t.Errorf("Payload() served build detail without IncludeBuildDetails: %+v", reduced)
	}
	if reduced.Version != "1.2.3" || reduced.Branch != "main" {
		t.Errorf("Payload() dropped a public field: %+v", reduced)
	}

	full := HandlerConfig{Info: info, IncludeBuildDetails: true}.Payload()
	if full.Commit != "abcdef1234567890" || full.GoVersion == "" {
		t.Errorf("Payload() withheld build detail that was asked for: %+v", full)
	}
}

func TestJSONResponseHonoursPretty(t *testing.T) {
	info := NewWithBranch("1.2.3", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	compact, status := HandlerConfig{Info: info}.JSONResponse()
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
	if bytes.Contains(compact, []byte("\n")) {
		t.Errorf("compact body = %s, want no newlines", compact)
	}

	pretty, status := HandlerConfig{Info: info, Pretty: true}.JSONResponse()
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
	if !bytes.Contains(pretty, []byte("\n  ")) {
		t.Errorf("pretty body = %s, want it indented", pretty)
	}

	// Same document either way -- Pretty is presentation, not policy.
	var a, b map[string]any
	if err := json.Unmarshal(compact, &a); err != nil {
		t.Fatalf("decode compact: %v", err)
	}
	if err := json.Unmarshal(pretty, &b); err != nil {
		t.Fatalf("decode pretty: %v", err)
	}
	if !maps.Equal(a, b) {
		t.Errorf("compact = %v, pretty = %v", a, b)
	}
}

func TestJSONResponseFollowsTheBuildDetailPolicy(t *testing.T) {
	info := NewWithBranch("1.2.3", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	body, _ := HandlerConfig{Info: info}.JSONResponse()

	var served map[string]any
	if err := json.Unmarshal(body, &served); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"commit", "build_date", "go_version", "platform", "compiler"} {
		if value, ok := served[field]; ok {
			t.Errorf("%s = %v, want it withheld without IncludeBuildDetails", field, value)
		}
	}
}

func TestJSONResponseOnZeroConfig(t *testing.T) {
	body, status := HandlerConfig{}.JSONResponse()

	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}

	var served map[string]any
	if err := json.Unmarshal(body, &served); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	if served["version"] != Version {
		t.Errorf("version = %v, want %q from Default()", served["version"], Version)
	}
}

func TestHeadersOnZeroConfig(t *testing.T) {
	headers := HandlerConfig{}.Headers()

	if _, ok := headers["X-Version"]; !ok {
		t.Errorf("Headers() on a zero config = %v, want an X-Version entry", headers)
	}
}

func TestHeadersFollowTheBuildDetailPolicy(t *testing.T) {
	info := NewWithBranch("1.2.3", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	reduced := HandlerConfig{Info: info, HeaderPrefix: "X-"}.Headers()
	if _, ok := reduced["X-Commit"]; ok {
		t.Errorf("Headers() emitted X-Commit without IncludeBuildDetails: %v", reduced)
	}
	if _, ok := reduced["X-Build-Date"]; ok {
		t.Errorf("Headers() emitted X-Build-Date without IncludeBuildDetails: %v", reduced)
	}
	if reduced["X-Version"] != "1.2.3" || reduced["X-Branch"] != "main" {
		t.Errorf("Headers() = %v, want the public headers", reduced)
	}

	full := HandlerConfig{Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true}.Headers()
	if full["X-Commit"] != "abcdef1" {
		t.Errorf("X-Commit = %q, want the short commit abcdef1", full["X-Commit"])
	}
	if full["X-Build-Date"] != "2026-01-02T03:04:05Z" {
		t.Errorf("X-Build-Date = %q, want the build date", full["X-Build-Date"])
	}
}

// Header values are sanitized in Headers, which is the single place every
// adapter now reads them from -- so this is the one test standing between a
// CR/LF in an ldflags-injected value and a response-splitting bug in whichever
// framework an adapter targets.
func TestHeadersStripControlCharacters(t *testing.T) {
	info := NewWithBranch(
		"1.0.0\r\nX-Injected: yes",
		"abc1234567890\r\n",
		"2026-01-02T03:04:05Z\r\n",
		"main\r\n",
	)

	headers := HandlerConfig{Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true}.Headers()

	for name, value := range headers {
		if strings.ContainsAny(value, "\r\n") {
			t.Errorf("%s = %q still carries a CR or LF", name, value)
		}
	}
	if headers["X-Version"] != "1.0.0X-Injected: yes" {
		t.Errorf("X-Version = %q, want the control characters removed and nothing else", headers["X-Version"])
	}
	if headers["X-Branch"] != "main" {
		t.Errorf("X-Branch = %q, want main", headers["X-Branch"])
	}
}

func TestHeadersNormalizeAnInvalidPrefix(t *testing.T) {
	info := New("1.0.0", "abc1234567890", "2026-01-02T03:04:05Z")

	headers := HandlerConfig{Info: info, HeaderPrefix: "Bad Prefix", IncludeBuildDetails: true}.Headers()

	for name := range headers {
		if strings.ContainsAny(name, " \t") {
			t.Errorf("header name %q contains whitespace; the prefix should have fallen back to X-", name)
		}
	}
	if _, ok := headers["X-Version"]; !ok {
		t.Errorf("Headers() = %v, want the X- fallback applied", headers)
	}
}

func TestHeadersOmitUnknownCommitAndBuildDate(t *testing.T) {
	info := &Info{Version: "1.0.0", Commit: "unknown", BuildDate: "unknown"}

	headers := HandlerConfig{Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true}.Headers()

	if _, ok := headers["X-Commit"]; ok {
		t.Errorf(`Headers() emitted X-Commit for a commit of "unknown": %v`, headers)
	}
	if _, ok := headers["X-Build-Date"]; ok {
		t.Errorf(`Headers() emitted X-Build-Date for a build date of "unknown": %v`, headers)
	}
}
