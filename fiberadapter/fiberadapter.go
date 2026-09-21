// Package fiberadapter serves version-kit's information over Fiber v3.
//
// It lives in its own package so that importing the root package does not drag
// Fiber — and with it fasthttp — into binaries that never use it. A service on
// net/http, Echo, Gin or chi pays nothing for Fiber support existing; only
// importing this package links it in.
//
// Everything here is a translation layer. The rules that decide what gets
// served — which fields are public, which headers are emitted, how a nil Info
// or a malformed header prefix is resolved — live in the root package and are
// read from there, so Fiber and net/http can never drift apart.
package fiberadapter

import (
	"github.com/gofiber/fiber/v3"

	version "github.com/soulteary/version-kit/v2"
)

// Handler returns a Fiber handler that serves version information.
// It is the Fiber counterpart of version.Handler.
func Handler(config ...version.HandlerConfig) fiber.Handler {
	cfg := version.ResolveConfig(config...)
	headers := cfg.Headers()

	return func(c fiber.Ctx) error {
		if cfg.IncludeHeaders {
			setHeaders(c, headers)
		}

		// HandlerConfig.Pretty has never taken effect on the Fiber side:
		// fiber.Ctx.JSON always writes compact JSON. Kept as-is so this change
		// stays a pure move; see the PR for the follow-up.
		//
		// The Content-Type is handed to JSON rather than set beforehand,
		// because JSON overwrites it either way. Left to its default, Fiber
		// answers "application/json; charset=utf-8" where net/http answers
		// "application/json" -- and RFC 8259 defines no charset parameter.
		return c.JSON(cfg.Payload(), fiber.MIMEApplicationJSON)
	}
}

// RegisterEndpoint registers the version handler on a Fiber app.
// It is the Fiber counterpart of version.RegisterEndpoint.
func RegisterEndpoint(app *fiber.App, path string, config ...version.HandlerConfig) {
	app.Get(path, Handler(config...))
}

// Middleware returns a Fiber middleware that adds version headers to all
// responses. It follows the same build-detail policy as version.Middleware:
// only the public fields, because these headers ride on every response.
func Middleware(info *version.Info, prefix string) fiber.Handler {
	return MiddlewareWithConfig(version.HandlerConfig{Info: info, HeaderPrefix: prefix})
}

// MiddlewareWithConfig is Middleware with the handlers' full configuration,
// including IncludeBuildDetails.
func MiddlewareWithConfig(config version.HandlerConfig) fiber.Handler {
	headers := config.Normalized().Headers()

	return func(c fiber.Ctx) error {
		setHeaders(c, headers)
		return c.Next()
	}
}

// TextHandler returns a Fiber handler that serves version information as plain
// text. It is the Fiber counterpart of version.TextHandler.
func TextHandler(config ...version.HandlerConfig) fiber.Handler {
	cfg := version.ResolveConfig(config...)
	headers := cfg.Headers()

	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setHeaders(c, headers)
		}

		return c.SendString(cfg.TextPayload())
	}
}

// SimpleHandler returns a minimal Fiber handler that just returns the version
// string. It is the Fiber counterpart of version.SimpleHandler.
func SimpleHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(version.Default().String())
	}
}

// setHeaders writes the version headers the root package computed. The names,
// values and sanitizing all come from HandlerConfig.Headers, so Fiber emits
// byte-for-byte what net/http does. They are the same on every response, so
// callers compute them once when the handler is built.
func setHeaders(c fiber.Ctx, headers map[string]string) {
	for name, value := range headers {
		c.Set(name, value)
	}
}
