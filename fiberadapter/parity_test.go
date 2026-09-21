// Parity and policy tests for the Fiber adapter.
//
// The moved tests in fiberadapter_test.go check that each handler works. These
// check the two things the split itself rests on: that the adapter serves the
// same bytes net/http does, and that it inherits the build-detail policy rather
// than quietly widening it. Both are invisible to statement coverage -- the
// package reaches 100% without either -- because opting build details in runs
// exactly the same lines as leaving them out.
//
// TestPrettyIsNotHonoured is the exception: it pins a known gap rather than a
// wanted behaviour, so that closing it shows up as a failing test.
package fiberadapter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	version "github.com/soulteary/version-kit/v2"
	"github.com/soulteary/version-kit/v2/fiberadapter"
)

// serveFiber runs one request through a Fiber app carrying h.
func serveFiber(t *testing.T, path string, h fiber.Handler) *http.Response {
	t.Helper()

	app := fiber.New()
	app.Get(path, h)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(b)
}

// The endpoint is usually unauthenticated, and go_version lets anyone match a
// published Go runtime CVE to the exact build serving them. The net/http side
// has covered this since the policy landed; the Fiber side never has.
func TestHandlerOmitsBuildDetailsByDefault(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	resp := serveFiber(t, "/version", fiberadapter.Handler(version.HandlerConfig{Info: info}))

	var served map[string]any
	if err := json.Unmarshal([]byte(body(t, resp)), &served); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	for _, field := range []string{"commit", "build_date", "go_version", "platform", "compiler"} {
		if value, ok := served[field]; ok {
			t.Errorf("%s = %v on the default endpoint, want it withheld", field, value)
		}
	}
	if served["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", served["version"])
	}
	if served["branch"] != "main" {
		t.Errorf("branch = %v, want main", served["branch"])
	}
}

func TestTextHandlerOmitsBuildDetailsByDefault(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	resp := serveFiber(t, "/version", fiberadapter.TextHandler(version.HandlerConfig{Info: info}))
	text := body(t, resp)

	for _, label := range []string{"Commit:", "Built:", "Go version:", "Platform:", "Compiler:"} {
		if strings.Contains(text, label) {
			t.Errorf("plain-text endpoint leaked %q by default:\n%s", label, text)
		}
	}
	if !strings.Contains(text, "Version:    1.0.0") {
		t.Errorf("plain-text endpoint = %q, want the version", text)
	}
}

func TestHandlerOmitsBuildDetailHeadersByDefault(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	resp := serveFiber(t, "/version", fiberadapter.Handler(version.HandlerConfig{
		Info:           info,
		IncludeHeaders: true,
		HeaderPrefix:   "X-",
	}))

	if got := resp.Header.Get("X-Commit"); got != "" {
		t.Errorf("X-Commit = %q, want it withheld without IncludeBuildDetails", got)
	}
	if got := resp.Header.Get("X-Build-Date"); got != "" {
		t.Errorf("X-Build-Date = %q, want it withheld without IncludeBuildDetails", got)
	}
	if got := resp.Header.Get("X-Version"); got != "1.0.0" {
		t.Errorf("X-Version = %q, want 1.0.0", got)
	}
}

// TestVersionHeadersMatchNetHTTP is the claim the split is built on: the
// adapter translates, it does not decide. Every header name and value comes
// from version.HandlerConfig.Headers, so the two frameworks cannot drift --
// and if someone reintroduces a Fiber-side rule, this is what catches it.
func TestVersionHeadersMatchNetHTTP(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")
	noCommit := &version.Info{Version: "1.0.0", Commit: "unknown", BuildDate: "unknown"}

	for _, tc := range []struct {
		name   string
		config version.HandlerConfig
	}{
		{"public only", version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-"}},
		{"build details", version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-", IncludeBuildDetails: true}},
		{"custom prefix", version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X-App-", IncludeBuildDetails: true}},
		{"token prefix", version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "X.App-", IncludeBuildDetails: true}},
		{"invalid prefix", version.HandlerConfig{Info: info, IncludeHeaders: true, HeaderPrefix: "Bad Prefix", IncludeBuildDetails: true}},
		{"nil info", version.HandlerConfig{IncludeHeaders: true, HeaderPrefix: "X-", IncludeBuildDetails: true}},
		{"unknown commit", version.HandlerConfig{Info: noCommit, IncludeHeaders: true, HeaderPrefix: "X-", IncludeBuildDetails: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			version.Handler(tc.config)(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
			std := rec.Result()
			defer func() { _ = std.Body.Close() }()

			fiberResp := serveFiber(t, "/version", fiberadapter.Handler(tc.config))

			want := tc.config.Headers()
			if len(want) == 0 {
				t.Fatal("Headers() returned nothing; the case is not exercising anything")
			}

			for name, value := range want {
				if got := std.Header.Get(name); got != value {
					t.Errorf("net/http %s = %q, want %q", name, got, value)
				}
				if got := fiberResp.Header.Get(name); got != value {
					t.Errorf("fiber %s = %q, want %q", name, got, value)
				}
			}

			// Neither side may emit a header the other does not.
			for _, suffix := range []string{"Version", "Commit", "Branch", "Build-Date"} {
				for _, prefix := range []string{"X-", "X-App-", "X.App-"} {
					name := prefix + suffix
					if a, b := std.Header.Get(name), fiberResp.Header.Get(name); a != b {
						t.Errorf("%s: net/http = %q, fiber = %q", name, a, b)
					}
				}
			}
		})
	}
}

// Header values are sanitized once, in the root package. This checks that the
// sanitizing survives the trip through fasthttp, which stores headers in its
// own buffers rather than net/http's map.
func TestVersionHeadersAreSanitized(t *testing.T) {
	info := version.NewWithBranch(
		"1.0.0\r\nX-Injected: yes",
		"abcdef1234567890\r\n",
		"2026-01-02T03:04:05Z\r\n",
		"main\r\n",
	)

	resp := serveFiber(t, "/version", fiberadapter.Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-",
		IncludeBuildDetails: true,
	}))

	if got := resp.Header.Get("X-Injected"); got != "" {
		t.Errorf("X-Injected = %q; a CR/LF in a version value split the response", got)
	}
	if got := resp.Header.Get("X-Version"); got != "1.0.0X-Injected: yes" {
		t.Errorf("X-Version = %q, want the control characters stripped", got)
	}
	if got := resp.Header.Get("X-Branch"); got != "main" {
		t.Errorf("X-Branch = %q, want main", got)
	}
	if got := resp.Header.Get("X-Build-Date"); got != "2026-01-02T03:04:05Z" {
		t.Errorf("X-Build-Date = %q, want the date without the trailing CR/LF", got)
	}
}

func TestMiddlewareNormalizesInvalidPrefix(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	app := fiber.New()
	app.Use(fiberadapter.Middleware(info, "Bad Prefix"))
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("X-Version"); got != "1.0.0" {
		t.Errorf("X-Version = %q, want the prefix to have fallen back to X-", got)
	}
}

// TestPrettyIsNotHonoured pins a known gap rather than a wanted behaviour:
// fiber.Ctx.JSON always writes compact JSON, so HandlerConfig.Pretty reaches
// the Fiber side and does nothing, while net/http honours it. Whoever closes
// that gap should see this fail and delete it.
func TestPrettyIsNotHonoured(t *testing.T) {
	info := version.New("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z")
	config := version.HandlerConfig{Info: info, Pretty: true, IncludeBuildDetails: true}

	resp := serveFiber(t, "/version", fiberadapter.Handler(config))
	got := body(t, resp)

	rec := httptest.NewRecorder()
	version.Handler(config)(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	std := rec.Body.String()

	if strings.Contains(got, "\n") {
		t.Errorf("Fiber honoured Pretty after all; drop this test and the comment in Handler:\n%s", got)
	}
	if !strings.Contains(std, "\n") {
		t.Errorf("net/http stopped honouring Pretty, which is a regression on its own:\n%s", std)
	}
}

// fiber.Ctx.JSON defaults to "application/json; charset=utf-8" and overwrites
// any Content-Type set before it, so the adapter has to hand it the type
// through JSON's ctype argument. Drop that argument and the two frameworks
// answer differently -- which is what this catches.
func TestJSONContentTypeMatchesNetHTTP(t *testing.T) {
	info := version.New("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z")
	config := version.HandlerConfig{Info: info}

	resp := serveFiber(t, "/version", fiberadapter.Handler(config))

	rec := httptest.NewRecorder()
	version.Handler(config)(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("net/http Content-Type = %q, want %q", got, want)
	}
	if got, want := resp.Header.Get("Content-Type"), rec.Header().Get("Content-Type"); got != want {
		t.Errorf("fiber Content-Type = %q, net/http = %q", got, want)
	}
}

// The text handlers do agree on Content-Type, and should keep doing so.
func TestTextContentTypeMatchesNetHTTP(t *testing.T) {
	info := version.New("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z")
	config := version.HandlerConfig{Info: info}

	resp := serveFiber(t, "/version", fiberadapter.TextHandler(config))

	rec := httptest.NewRecorder()
	version.TextHandler(config)(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if got, want := resp.Header.Get("Content-Type"), rec.Header().Get("Content-Type"); got != want {
		t.Errorf("fiber Content-Type = %q, net/http = %q", got, want)
	}
}

// The plain-text and JSON bodies must be the byte-for-byte same as net/http's
// wherever Pretty is not involved.
func TestBodiesMatchNetHTTP(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abcdef1234567890", "2026-01-02T03:04:05Z", "main")

	for _, tc := range []struct {
		name   string
		config version.HandlerConfig
	}{
		{"public", version.HandlerConfig{Info: info}},
		{"build details", version.HandlerConfig{Info: info, IncludeBuildDetails: true}},
		{"nil info", version.HandlerConfig{IncludeBuildDetails: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			jsonResp := serveFiber(t, "/version", fiberadapter.Handler(tc.config))
			rec := httptest.NewRecorder()
			version.Handler(tc.config)(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
			if got, want := body(t, jsonResp), rec.Body.String(); got != want {
				t.Errorf("JSON body:\n fiber    = %s\n net/http = %s", got, want)
			}

			textResp := serveFiber(t, "/version.txt", fiberadapter.TextHandler(tc.config))
			rec = httptest.NewRecorder()
			version.TextHandler(tc.config)(rec, httptest.NewRequest(http.MethodGet, "/version.txt", nil))
			if got, want := body(t, textResp), rec.Body.String(); got != want {
				t.Errorf("text body:\n fiber    = %q\n net/http = %q", got, want)
			}
		})
	}
}

// SimpleHandler is the one handler that reads the package variables at request
// time rather than from a config, on both sides.
func TestSimpleHandlerMatchesNetHTTP(t *testing.T) {
	origVersion, origCommit := version.Version, version.Commit
	t.Cleanup(func() {
		version.Version, version.Commit = origVersion, origCommit
	})
	version.Version, version.Commit = "2.0.0", "xyz789abcdef"

	resp := serveFiber(t, "/version", fiberadapter.SimpleHandler())

	rec := httptest.NewRecorder()
	version.SimpleHandler()(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	got := body(t, resp)
	if want := rec.Body.String(); got != want {
		t.Errorf("fiber = %q, net/http = %q", got, want)
	}
	if got != "2.0.0 (xyz789a)" {
		t.Errorf("body = %q, want the short-commit form", got)
	}
}
