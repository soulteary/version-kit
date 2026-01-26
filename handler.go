package version

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// HandlerConfig configures the version endpoint handler.
type HandlerConfig struct {
	// Info is the version information to return.
	// If nil, Default() will be used.
	Info *Info

	// Pretty enables pretty-printed JSON output.
	// Default: false
	Pretty bool

	// IncludeHeaders adds version info to response headers.
	// Default: false
	IncludeHeaders bool

	// HeaderPrefix is the prefix for version headers.
	// Default: "X-"
	HeaderPrefix string
}

// DefaultHandlerConfig returns a HandlerConfig with default values.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		Info:           Default(),
		Pretty:         false,
		IncludeHeaders: false,
		HeaderPrefix:   "X-",
	}
}

// Handler returns an http.HandlerFunc that serves version information.
func Handler(config ...HandlerConfig) http.HandlerFunc {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	if cfg.HeaderPrefix == "" {
		cfg.HeaderPrefix = "X-"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if cfg.IncludeHeaders {
			setVersionHeaders(w.Header(), cfg.Info, cfg.HeaderPrefix)
		}

		var output []byte
		var err error

		if cfg.Pretty {
			output, err = json.MarshalIndent(cfg.Info, "", "  ")
		} else {
			output, err = json.Marshal(cfg.Info)
		}

		if err != nil {
			http.Error(w, `{"error": "failed to marshal version info"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}

// FiberHandler returns a Fiber handler that serves version information.
func FiberHandler(config ...HandlerConfig) fiber.Handler {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	if cfg.HeaderPrefix == "" {
		cfg.HeaderPrefix = "X-"
	}

	return func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")

		if cfg.IncludeHeaders {
			setVersionHeadersFiber(c, cfg.Info, cfg.HeaderPrefix)
		}

		if cfg.Pretty {
			return c.JSON(cfg.Info)
		}

		return c.JSON(cfg.Info)
	}
}

// RegisterEndpoint registers the version handler on an http.ServeMux.
func RegisterEndpoint(mux *http.ServeMux, path string, config ...HandlerConfig) {
	mux.HandleFunc(path, Handler(config...))
}

// RegisterEndpointFiber registers the version handler on a Fiber app.
func RegisterEndpointFiber(app *fiber.App, path string, config ...HandlerConfig) {
	app.Get(path, FiberHandler(config...))
}

// setVersionHeaders adds version information to HTTP headers.
func setVersionHeaders(h http.Header, info *Info, prefix string) {
	h.Set(prefix+"Version", info.Version)

	if info.Commit != "" && info.Commit != "unknown" {
		h.Set(prefix+"Commit", info.ShortCommit())
	}

	if info.Branch != "" {
		h.Set(prefix+"Branch", info.Branch)
	}

	if info.BuildDate != "" && info.BuildDate != "unknown" {
		h.Set(prefix+"Build-Date", info.BuildDate)
	}
}

// setVersionHeadersFiber adds version information to Fiber response headers.
func setVersionHeadersFiber(c *fiber.Ctx, info *Info, prefix string) {
	c.Set(prefix+"Version", info.Version)

	if info.Commit != "" && info.Commit != "unknown" {
		c.Set(prefix+"Commit", info.ShortCommit())
	}

	if info.Branch != "" {
		c.Set(prefix+"Branch", info.Branch)
	}

	if info.BuildDate != "" && info.BuildDate != "unknown" {
		c.Set(prefix+"Build-Date", info.BuildDate)
	}
}

// Middleware returns an http.Handler middleware that adds version headers to all responses.
func Middleware(info *Info, prefix string) func(http.Handler) http.Handler {
	if info == nil {
		info = Default()
	}
	if prefix == "" {
		prefix = "X-"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setVersionHeaders(w.Header(), info, prefix)
			next.ServeHTTP(w, r)
		})
	}
}

// FiberMiddleware returns a Fiber middleware that adds version headers to all responses.
func FiberMiddleware(info *Info, prefix string) fiber.Handler {
	if info == nil {
		info = Default()
	}
	if prefix == "" {
		prefix = "X-"
	}

	return func(c *fiber.Ctx) error {
		setVersionHeadersFiber(c, info, prefix)
		return c.Next()
	}
}

// TextHandler returns an http.HandlerFunc that serves version information as plain text.
func TextHandler(config ...HandlerConfig) http.HandlerFunc {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setVersionHeaders(w.Header(), cfg.Info, cfg.HeaderPrefix)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cfg.Info.Full()))
	}
}

// FiberTextHandler returns a Fiber handler that serves version information as plain text.
func FiberTextHandler(config ...HandlerConfig) fiber.Handler {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Info == nil {
		cfg.Info = Default()
	}

	return func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")

		if cfg.IncludeHeaders {
			setVersionHeadersFiber(c, cfg.Info, cfg.HeaderPrefix)
		}

		return c.SendString(cfg.Info.Full())
	}
}

// SimpleHandler returns a minimal handler that just returns the version string.
func SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(Default().String()))
	}
}

// FiberSimpleHandler returns a minimal Fiber handler that just returns the version string.
func FiberSimpleHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(Default().String())
	}
}
