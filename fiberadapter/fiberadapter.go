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
	output, status := cfg.JSONResponse()

	return func(c fiber.Ctx) error {
		c.Set("Content-Type", fiber.MIMEApplicationJSON)

		if cfg.IncludeHeaders {
			setHeaders(c, headers)
		}

		// The body and status come from the root package rather than c.JSON,
		// which would re-encode the Info per request with whatever JSONEncoder
		// the app was configured with, and would ignore HandlerConfig.Pretty
		// because it always writes compact JSON. Sending the bytes the
		// net/http handler would send makes the two byte-identical.
		//
		// c.Send hands the slice to fasthttp without copying, so output is
		// written once here and only ever read afterwards.
		c.Status(status)

		return c.Send(output)
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
	output := cfg.TextPayload()

	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setHeaders(c, headers)
		}

		return c.SendString(output)
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
