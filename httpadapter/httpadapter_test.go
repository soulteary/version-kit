package httpadapter

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	version "github.com/soulteary/version-kit/v4"
)

func TestHandler(t *testing.T) {
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	handler := Handler(version.HandlerConfig{Info: info, IncludeBuildDetails: true})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var parsed version.Info
	err = json.Unmarshal(body, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
	// Build detail is opt-in now; see TestHandlerOmitsBuildDetailsByDefault.
	assert.Equal(t, "abc123", parsed.Commit)
}

func TestHandler_DefaultConfig(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var parsed version.Info
	err = json.Unmarshal(body, &parsed)
	require.NoError(t, err)

	// Should use default version
	assert.NotEmpty(t, parsed.Version)
}

func TestHandler_WithHeaders(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")
	handler := Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-App-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
	assert.Equal(t, "2025-01-01T00:00:00Z", resp.Header.Get("X-App-Build-Date"))
}

func TestHandler_WithHeaders_SanitizesValues(t *testing.T) {
	info := version.NewWithBranch("1.0.0\r\n", "abc1234567890", "2025-01-01T00:00:00Z\r\n", "main\r\n")
	handler := Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
	assert.Equal(t, "2025-01-01T00:00:00Z", resp.Header.Get("X-App-Build-Date"))
}

func TestHandler_Pretty(t *testing.T) {
	info := version.New("1.0.0", "abc123", "")
	handler := Handler(version.HandlerConfig{
		Info:                info,
		Pretty:              true,
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Pretty output should contain indentation
	assert.Contains(t, string(body), "\n")
	assert.Contains(t, string(body), "  ")
}

func TestTextHandler(t *testing.T) {
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	handler := TextHandler(version.HandlerConfig{Info: info, IncludeBuildDetails: true})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Version:    1.0.0")
	assert.Contains(t, string(body), "Commit:     abc123")
}

func TestSimpleHandler(t *testing.T) {
	// Save and restore original values
	origVersion := version.Version
	origCommit := version.Commit
	defer func() {
		version.Version = origVersion
		version.Commit = origCommit
	}()

	version.Version = "2.0.0"
	version.Commit = "xyz789"

	handler := SimpleHandler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "2.0.0")
}

func TestMiddleware(t *testing.T) {
	info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	// The commit header is the opt-in form now; Middleware alone is public-only.
	middleware := MiddlewareWithConfig(version.HandlerConfig{Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true})

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := middleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
	assert.Equal(t, "abc123", resp.Header.Get("X-Commit"))
}

func TestMiddleware_DefaultInfo(t *testing.T) {
	middleware := Middleware(nil, "")

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should have default X-Version header
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestRegisterEndpoint(t *testing.T) {
	info := version.New("1.0.0", "abc123", "")
	mux := http.NewServeMux()
	RegisterEndpoint(mux, "/version", version.HandlerConfig{Info: info, IncludeBuildDetails: true})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/version")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed version.Info
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestHandler_NoCommit(t *testing.T) {
	info := &version.Info{
		Version: "1.0.0",
	}
	handler := Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should not have commit header when commit is empty
	assert.Empty(t, resp.Header.Get("X-Commit"))
}

func TestHandler_NilInfoAndEmptyPrefix(t *testing.T) {
	// Test with nil version.Info (should use Default) and empty HeaderPrefix (should use "X-")
	handler := Handler(version.HandlerConfig{
		Info:                nil,
		IncludeHeaders:      true,
		HeaderPrefix:        "", // Empty prefix should default to "X-"
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Should have X- prefix headers
	assert.NotEmpty(t, resp.Header.Get("X-Version"))
}

func TestTextHandler_DefaultConfig(t *testing.T) {
	handler := TextHandler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info
	assert.Contains(t, string(body), "Version:")
}

func TestTextHandler_WithHeaders(t *testing.T) {
	info := version.NewWithBranch("1.0.0", "abc1234567890", "2025-01-01T00:00:00Z", "main")
	handler := TextHandler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		HeaderPrefix:        "X-App-",
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "1.0.0", resp.Header.Get("X-App-Version"))
	assert.Equal(t, "abc1234", resp.Header.Get("X-App-Commit"))
	assert.Equal(t, "main", resp.Header.Get("X-App-Branch"))
}

func TestTextHandler_NilInfo(t *testing.T) {
	handler := TextHandler(version.HandlerConfig{
		Info:                nil, // Should use version.Default()
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain version info from version.Default()
	assert.Contains(t, string(body), "Version:")
}

func TestSetVersionHeaders_UnknownCommit(t *testing.T) {
	info := &version.Info{
		Version:   "1.0.0",
		Commit:    "unknown",
		BuildDate: "unknown",
	}
	handler := Handler(version.HandlerConfig{
		Info:                info,
		IncludeHeaders:      true,
		IncludeBuildDetails: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should not have commit/build-date headers when they are "unknown"
	assert.Empty(t, resp.Header.Get("X-Commit"))
	assert.Empty(t, resp.Header.Get("X-Build-Date"))
}
