// Package httpadapter serves version-kit's information over net/http.
//
// It lives in its own package for the same reason fiberadapter does: importing
// the root package should not drag a web server into a binary that never
// serves one. version-kit's most common use is a --version flag in a CLI, and
// net/http costs such a binary 124 packages it has no use for.
//
// Everything here is a translation layer. What gets served -- which fields are
// public, which headers are emitted, how a nil Info or a malformed header
// prefix is resolved -- is decided by version.HandlerConfig in the root
// package and read from there, so net/http and Fiber can never drift apart.
package httpadapter

import (
	"net/http"

	version "github.com/soulteary/version-kit/v4"
)

// Handler returns an http.HandlerFunc that serves version information.
func Handler(config ...version.HandlerConfig) http.HandlerFunc {
	cfg := version.ResolveConfig(config...)
	headers := cfg.Headers()
	output, status := cfg.JSONResponse()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if cfg.IncludeHeaders {
			setVersionHeaders(w.Header(), headers)
		}

		w.WriteHeader(status)
		_, _ = w.Write(output)
	}
}

// RegisterEndpoint registers the version handler on an http.ServeMux.
func RegisterEndpoint(mux *http.ServeMux, path string, config ...version.HandlerConfig) {
	mux.HandleFunc(path, Handler(config...))
}

// setVersionHeaders writes headers that HandlerConfig.Headers already computed.
// They are the same on every response, so callers compute them once when the
// handler is built rather than rebuilding the map per request -- these run on
// every response a middleware touches.
func setVersionHeaders(h http.Header, headers map[string]string) {
	for name, value := range headers {
		h.Set(name, value)
	}
}

// Middleware returns an http.Handler middleware that adds version headers to
// all responses.
//
// It emits only the public fields. These headers ride on EVERY response, so a
// middleware mounted on public routes leaks the commit and build date far more
// widely than the /version endpoint does -- keeping the endpoint private buys
// nothing while this one is open. MiddlewareWithConfig opts back in.
func Middleware(info *version.Info, prefix string) func(http.Handler) http.Handler {
	return MiddlewareWithConfig(version.HandlerConfig{Info: info, HeaderPrefix: prefix})
}

// MiddlewareWithConfig is Middleware with the handlers' full configuration,
// including IncludeBuildDetails for the commit and build-date headers.
func MiddlewareWithConfig(config version.HandlerConfig) func(http.Handler) http.Handler {
	headers := config.Normalized().Headers()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setVersionHeaders(w.Header(), headers)
			next.ServeHTTP(w, r)
		})
	}
}

// TextHandler returns an http.HandlerFunc that serves version information as plain text.
func TextHandler(config ...version.HandlerConfig) http.HandlerFunc {
	cfg := version.ResolveConfig(config...)
	headers := cfg.Headers()
	output := []byte(cfg.TextPayload())

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setVersionHeaders(w.Header(), headers)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}

// SimpleHandler returns a minimal handler that just returns the version string.
func SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(version.Default().String()))
	}
}
