// Fiber adapter tests. These moved out of the root package together with the
// handlers they cover; they deliberately use only version-kit's exported API,
// which is what an out-of-tree adapter would have to work with too.
package fiberadapter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	version "github.com/soulteary/version-kit/v4"
	"github.com/soulteary/version-kit/v4/fiberadapter"
)

func TestFiberHandler(t *testing.T) {
	app := fiber.New()
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	app.Get("/version", fiberadapter.Handler(version.HandlerConfig{Info: info, IncludeBuildDetails: true}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed version.Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
	// Build detail is opt-in now; see TestHandlerOmitsBuildDetailsByDefault.
	assert.Equal(t, "abc123", parsed.Commit)
}

func TestFiberHandler_WithHeaders(t *testing.T) {
	app := fiber.New()
	info := version.NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")

	app.Get("/version", fiberadapter.Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-",
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-Branch"))
}

func TestFiberTextHandler(t *testing.T) {
	app := fiber.New()
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	app.Get("/version", fiberadapter.TextHandler(version.HandlerConfig{Info: info, IncludeBuildDetails: true}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Version:    1.0.0")
}

func TestFiberSimpleHandler(t *testing.T) {
	// Save and restore original values
	origVersion := version.Version
	origCommit := version.Commit
	defer func() {
		version.Version = origVersion
		version.Commit = origCommit
	}()

	version.Version = "2.0.0"
	version.Commit = "xyz789"

	app := fiber.New()
	app.Get("/version", fiberadapter.SimpleHandler())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "2.0.0")
}

func TestFiberMiddleware(t *testing.T) {
	app := fiber.New()
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	app.Use(fiberadapter.MiddlewareWithConfig(version.HandlerConfig{Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true}))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
	assert.Equal(t, "abc123", resp.Header.Get("X-Commit"))
}

func TestFiberMiddleware_DefaultInfo(t *testing.T) {
	app := fiber.New()

	app.Use(fiberadapter.Middleware(nil, ""))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should have default X-Version header
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestRegisterEndpointFiber(t *testing.T) {
	app := fiber.New()
	info := version.New("1.0.0", "abc123", "")

	fiberadapter.RegisterEndpoint(app, "/version", version.HandlerConfig{Info: info, IncludeBuildDetails: true})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed version.Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestFiberHandler_NilInfoAndEmptyPrefix(t *testing.T) {
	app := fiber.New()

	// Test with nil version.Info (should use version.Default) and empty HeaderPrefix (should use "X-")
	app.Get("/version", fiberadapter.Handler(version.HandlerConfig{
		Info:                nil,
		IncludeHeaders:      true,
		HeaderPrefix:        "", // Empty prefix should default to "X-"
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Should have X- prefix headers
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestFiberHandler_DefaultConfig(t *testing.T) {
	app := fiber.New()
	app.Get("/version", fiberadapter.Handler())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed version.Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	// Should use default version
	assert.NotEmpty(t, parsed.Version)
}

func TestFiberHandler_Pretty(t *testing.T) {
	app := fiber.New()
	info := version.New("1.0.0", "abc123", "")

	app.Get("/version", fiberadapter.Handler(version.HandlerConfig{
		Info:                info,
		Pretty:              true,
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed version.Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestFiberTextHandler_DefaultConfig(t *testing.T) {
	app := fiber.New()
	app.Get("/version", fiberadapter.TextHandler())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info
	assert.Contains(t, string(body), "Version:")
}

func TestFiberTextHandler_WithHeaders(t *testing.T) {
	app := fiber.New()
	info := version.NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")

	app.Get("/version", fiberadapter.TextHandler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-App-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
}

func TestFiberTextHandler_NilInfo(t *testing.T) {
	app := fiber.New()
	app.Get("/version", fiberadapter.TextHandler(version.HandlerConfig{
		Info:                nil, // Should use version.Default()
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info from version.Default()
	assert.Contains(t, string(body), "Version:")
}

func TestSetVersionHeadersFiber_UnknownCommit(t *testing.T) {
	app := fiber.New()
	info := &version.Info{
		Version:   "1.0.0",
		Commit:    "unknown",
		BuildDate: "unknown",
	}

	app.Get("/version", fiberadapter.Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		IncludeBuildDetails: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should not have commit/build-date headers when they are "unknown"
	assert.Empty(t, resp.Header.Get("X-Commit"))
	assert.Empty(t, resp.Header.Get("X-Build-Date"))
}

func TestFiberMiddlewareOmitsBuildDetailsByDefault(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abc1234567", "2025-01-01T00:00:00Z", "main")

	app := fiber.New()
	app.Use(fiberadapter.Middleware(info, "X-"))
	app.Get("/anything", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/anything", nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Header.Get("X-Commit"); got != "" {
		t.Errorf("X-Commit = %q on an ordinary response, want it withheld by default", got)
	}
	if got := resp.Header.Get("X-Build-Date"); got != "" {
		t.Errorf("X-Build-Date = %q on an ordinary response, want it withheld by default", got)
	}
	if resp.Header.Get("X-Version") != "1.0.0" {
		t.Errorf("X-Version = %q, want 1.0.0", resp.Header.Get("X-Version"))
	}
}
