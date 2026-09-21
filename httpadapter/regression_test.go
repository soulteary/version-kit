package httpadapter

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	version "github.com/soulteary/version-kit/v4"
)

// TestHandlerOmitsBuildDetailsByDefault: the version endpoint is usually
// unauthenticated, and go_version lets anyone match a published Go runtime CVE
// to the exact build serving them. Commit, build date, platform and compiler
// narrow it further. None of it is a vulnerability on its own; it is
// reconnaissance that costs nothing to withhold.
func TestHandlerOmitsBuildDetailsByDefault(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")

	rec := httptest.NewRecorder()
	Handler(version.HandlerConfig{Info: info})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	body, _ := io.ReadAll(rec.Result().Body)

	var parsed version.Info
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, body)
	}

	if parsed.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", parsed.Version)
	}
	if parsed.Branch != "main" {
		t.Errorf("branch = %q, want main", parsed.Branch)
	}
	for name, value := range map[string]string{
		"commit":     parsed.Commit,
		"build_date": parsed.BuildDate,
		"go_version": parsed.GoVersion,
		"platform":   parsed.Platform,
		"compiler":   parsed.Compiler,
	} {
		if value != "" {
			t.Errorf("%s = %q, want it withheld by default", name, value)
		}
	}
	if strings.Contains(string(body), runtime.Version()) {
		t.Errorf("the Go runtime version reached an unauthenticated response: %s", body)
	}
}

func TestHandlerIncludesBuildDetailsWhenAsked(t *testing.T) {
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	rec := httptest.NewRecorder()
	Handler(version.HandlerConfig{Info: info, IncludeBuildDetails: true})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	body, _ := io.ReadAll(rec.Result().Body)

	var parsed version.Info
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Commit != "abc123" || parsed.GoVersion == "" {
		t.Errorf("IncludeBuildDetails did not take effect: %s", body)
	}
}

func TestTextHandlerOmitsBuildDetailsByDefault(t *testing.T) {
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	rec := httptest.NewRecorder()
	TextHandler(version.HandlerConfig{Info: info})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	body, _ := io.ReadAll(rec.Result().Body)
	text := string(body)

	if !strings.Contains(text, "Version:") {
		t.Errorf("the version line is missing: %q", text)
	}
	for _, leaked := range []string{"abc123", runtime.Version(), "Go version:", "Compiler:"} {
		if strings.Contains(text, leaked) {
			t.Errorf("text response leaked %q: %q", leaked, text)
		}
	}
}

// TestInvalidHeaderPrefixFallsBack: the prefix is concatenated into a header
// NAME, so characters that cannot appear there produce a malformed header.

func TestInvalidHeaderPrefixFallsBack(t *testing.T) {
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	for _, prefix := range []string{"", "bad prefix ", "x:y", "a\nb"} {
		rec := httptest.NewRecorder()
		Handler(version.HandlerConfig{
			Info:           info,
			IncludeHeaders: true,
			HeaderPrefix:   prefix,
		})(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

		if got := rec.Header().Get("X-Version"); got != "1.0.0" {
			t.Errorf("prefix %q: X-Version = %q, want the fallback prefix to be used", prefix, got)
		}
	}

	// A valid prefix is honoured.
	rec := httptest.NewRecorder()
	Handler(version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-App-"})(
		rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if got := rec.Header().Get("X-App-Version"); got != "1.0.0" {
		t.Errorf("X-App-Version = %q, want 1.0.0", got)
	}
}

// TestHeadersHonourIncludeBuildDetails is the regression test for headers being
// built from cfg.Info instead of the payload actually served: an endpoint with
// IncludeHeaders on but IncludeBuildDetails off still emitted the commit and
// build date, so the build fingerprint leaked through the header path.
func TestHeadersHonourIncludeBuildDetails(t *testing.T) {
	info := &version.Info{
		Version:   "1.2.3",
		Branch:    "main",
		Commit:    "abcdef1234567890",
		BuildDate: "2026-01-02T03:04:05Z",
		GoVersion: "go1.27.0",
	}

	for _, tc := range []struct {
		name    string
		handler func(version.HandlerConfig) http.HandlerFunc
	}{
		{"json", func(c version.HandlerConfig) http.HandlerFunc { return Handler(c) }},
		{"text", func(c version.HandlerConfig) http.HandlerFunc { return TextHandler(c) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.handler(version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-"})
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

			if got := rec.Header().Get("X-Commit"); got != "" {
				t.Errorf("X-Commit leaked without IncludeBuildDetails: %q", got)
			}
			if got := rec.Header().Get("X-Build-Date"); got != "" {
				t.Errorf("X-Build-Date leaked without IncludeBuildDetails: %q", got)
			}
			if got := rec.Header().Get("X-Version"); got != "1.2.3" {
				t.Errorf("X-Version = %q, want 1.2.3", got)
			}
			if got := rec.Header().Get("X-Branch"); got != "main" {
				t.Errorf("X-Branch = %q, want main", got)
			}
		})
	}

	t.Run("opted in", func(t *testing.T) {
		h := Handler(version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-", IncludeBuildDetails: true})
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

		if got := rec.Header().Get("X-Commit"); got != "abcdef1" {
			t.Errorf("X-Commit = %q, want abcdef1", got)
		}
		if got := rec.Header().Get("X-Build-Date"); got == "" {
			t.Error("X-Build-Date missing with IncludeBuildDetails")
		}
	})
}

// TestUnusualHeaderPrefixSurvivesIntoTheHeaderName pairs with
// TestValidHeaderPrefixAcceptsTokenChars in the root package: that one checks
// which prefixes the validator accepts, this one that an accepted prefix is
// what actually reaches the wire rather than being rewritten to "X-".
func TestUnusualHeaderPrefixSurvivesIntoTheHeaderName(t *testing.T) {
	h := Handler(version.HandlerConfig{Info: &version.Info{Version: "9.9.9"}, IncludeHeaders: true, HeaderPrefix: "X.App-"})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if got := rec.Header().Get("X.App-Version"); got != "9.9.9" {
		t.Errorf("X.App-Version = %q, want 9.9.9 (prefix was rewritten)", got)
	}
}

// TestTextHandlersNormalizeHeaderPrefix is the regression test for the text
// handlers skipping validHeaderPrefix: an invalid prefix reached the header
// setter instead of falling back to "X-" the way Handler documents.

func TestTextHandlersNormalizeHeaderPrefix(t *testing.T) {
	h := TextHandler(version.HandlerConfig{
		Info:           &version.Info{Version: "4.5.6"},
		IncludeHeaders: true,
		HeaderPrefix:   "bad prefix:",
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if got := rec.Header().Get("X-Version"); got != "4.5.6" {
		t.Errorf("X-Version = %q, want 4.5.6 (invalid prefix was not normalized)", got)
	}
	for name := range rec.Header() {
		if strings.Contains(name, " ") || strings.Contains(name, ":") {
			t.Errorf("malformed header name reached the response: %q", name)
		}
	}
}

// --- Codex review round 3 (PR #4) ---

// TestDocumentedBuildDetailsExampleCompiles pins the READMEs' build-details
// example to the actual API. Both guides called version.Get(), which does not
// exist -- the package-level constructor for ldflag-backed information is
// version.Default() -- so the one example a reader would copy to turn the full
// response back on did not compile.

func TestDocumentedBuildDetailsExampleCompiles(t *testing.T) {
	h := Handler(version.HandlerConfig{
		Info:                version.Default(),
		IncludeBuildDetails: true,
	})

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	// The documented point of the flag: the build fingerprint comes back.
	for _, field := range []string{"version", "commit", "build_date", "go_version"} {
		if _, ok := payload[field]; !ok {
			t.Errorf("field %q missing; IncludeBuildDetails must serve the full Info", field)
		}
	}

	// And the documented default keeps them out.
	def := Handler(version.HandlerConfig{Info: version.Default()})
	rec2 := httptest.NewRecorder()
	def(rec2, httptest.NewRequest(http.MethodGet, "/version", nil))

	var bare map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &bare); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"commit", "build_date", "go_version"} {
		if _, ok := bare[field]; ok {
			t.Errorf("field %q served by default; the READMEs promise only version and branch", field)
		}
	}
}

// TestMiddlewareOmitsBuildDetailsByDefault: the handlers withhold build
// details, but these headers ride on EVERY response, so a middleware mounted
// on public routes hands out the commit and build date far more widely than
// the endpoint ever did -- and keeping that endpoint private buys nothing
// while this one is open.

func TestMiddlewareOmitsBuildDetailsByDefault(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abc1234567", "2025-01-01T00:00:00Z", "main")

	rec := httptest.NewRecorder()
	Middleware(info, "X-")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))

	h := rec.Result().Header
	if h.Get("X-Version") != "1.0.0" {
		t.Errorf("X-Version = %q, want 1.0.0", h.Get("X-Version"))
	}
	if h.Get("X-Branch") != "main" {
		t.Errorf("X-Branch = %q, want main", h.Get("X-Branch"))
	}
	if got := h.Get("X-Commit"); got != "" {
		t.Errorf("X-Commit = %q on an ordinary response, want it withheld by default", got)
	}
	if got := h.Get("X-Build-Date"); got != "" {
		t.Errorf("X-Build-Date = %q on an ordinary response, want it withheld by default", got)
	}
}

func TestMiddlewareWithConfigIncludesBuildDetailsWhenAsked(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abc1234567", "2025-01-01T00:00:00Z", "main")

	rec := httptest.NewRecorder()
	MiddlewareWithConfig(version.HandlerConfig{Info: info, IncludeBuildDetails: true})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))

	h := rec.Result().Header
	if h.Get("X-Commit") != "abc1234" {
		t.Errorf("X-Commit = %q, want the short commit", h.Get("X-Commit"))
	}
	if h.Get("X-Build-Date") == "" {
		t.Error("X-Build-Date is empty, want the opt-in to restore it")
	}
}

// TestMiddlewareNormalizesInvalidPrefix: the documented fallback applies here
// too, so a malformed prefix cannot produce an invalid header name.

func TestMiddlewareNormalizesInvalidPrefix(t *testing.T) {
	info := version.New("1.0.0", "abc1234567", "2025-01-01T00:00:00Z")

	rec := httptest.NewRecorder()
	Middleware(info, "Bad Prefix ")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))

	if rec.Result().Header.Get("X-Version") != "1.0.0" {
		t.Errorf("an invalid prefix did not fall back to X-: %v", rec.Result().Header)
	}
}
