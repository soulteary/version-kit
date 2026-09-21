package version

import (
	"encoding/json"
	"strings"
)

// The two status codes JSONResponse can return.
//
// Written out rather than read from net/http, because referring to them as
// http.StatusOK and http.StatusInternalServerError would put net/http back
// into this package's import graph -- which is exactly what httpadapter
// exists to keep out of binaries that never serve HTTP. The values are fixed
// by RFC 9110 and cannot drift.
const (
	statusOK                  = 200
	statusInternalServerError = 500
)

// HandlerConfig configures the version endpoint handler.
type HandlerConfig struct {
	// Info is the version information to return.
	// If nil, Default() will be used.
	//
	// The response body and the version headers are both computed once, when
	// the handler or middleware is built, and reused on every response --
	// mutating an Info afterwards does not change what is served. That was
	// already a data race, so nothing which was safe before stopped working.
	// SimpleHandler is the exception: it takes no config and still reads the
	// package variables per request.
	Info *Info

	// Pretty enables pretty-printed JSON output.
	// Default: false
	Pretty bool

	// IncludeHeaders adds version info to response headers.
	// Default: false
	IncludeHeaders bool

	// HeaderPrefix is the prefix for version headers.
	// Default: "X-"
	//
	// An invalid prefix (one containing characters that cannot appear in an
	// HTTP header name) falls back to "X-" rather than emitting a malformed
	// header name.
	HeaderPrefix string

	// IncludeBuildDetails serves the full Info -- Go runtime version, commit,
	// build date, platform and compiler.
	//
	// Default: false. This endpoint is usually unauthenticated, and
	// go_version lets anyone match a published Go runtime CVE to the exact
	// build serving them. Turn it on for an internal endpoint, or behind
	// authentication.
	IncludeBuildDetails bool
}

// validHeaderPrefix reports whether prefix can appear in an HTTP header name.
//
// A field name is a token (RFC 9110 5.6.2), so every tchar is allowed here --
// not just the alphanumeric/dash subset. Prefixes such as "X.App-" were valid
// before this validator existed and must keep working.
func validHeaderPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	for _, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
		default:
			return false
		}
	}
	return true
}

// normalizeHeaderPrefix falls back to "X-" for a prefix that is not a token.
func normalizeHeaderPrefix(prefix string) string {
	if validHeaderPrefix(prefix) {
		return prefix
	}
	return "X-"
}

// Payload returns the Info to serve, reduced unless build details were asked for.
//
// Exported for framework adapters (httpadapter and fiberadapter in-tree, and
// whatever an Echo, Gin or chi service writes for itself): the reduction rule
// must be identical across frameworks, so every adapter reads it from here
// instead of restating it.
func (c HandlerConfig) Payload() *Info {
	if c.Info == nil {
		c = c.Normalized()
	}
	if c.IncludeBuildDetails {
		return c.Info
	}
	return c.Info.Public()
}

// TextPayload is Payload rendered for the plain-text handlers.
func (c HandlerConfig) TextPayload() string {
	if c.Info == nil {
		c = c.Normalized()
	}
	if c.IncludeBuildDetails {
		return c.Info.Full()
	}
	return c.Info.Public().Full()
}

// JSONResponse returns the JSON body the handlers serve and the status to
// serve it with: Payload marshalled, indented when Pretty is set.
//
// Exported for the same reason Payload is. An adapter that hands its framework
// the Info instead inherits whatever encoder that framework happens to be
// configured with, and gets Pretty only if that framework offers it -- which
// is how Fiber came to ignore Pretty while net/http honoured it.
//
// The status comes back with the body because the failure has to be decided
// somewhere, and one place is the point: marshalling an Info cannot actually
// fail -- every field is a string -- but an adapter should not have to guess
// what to serve if it ever did. Callers write the two values out and have
// nothing left to decide.
func (c HandlerConfig) JSONResponse() (body []byte, status int) {
	var (
		output []byte
		err    error
	)

	if c.Pretty {
		output, err = json.MarshalIndent(c.Payload(), "", "  ")
	} else {
		output, err = json.Marshal(c.Payload())
	}

	if err != nil {
		return []byte(`{"error": "failed to marshal version info"}`), statusInternalServerError
	}

	return output, statusOK
}

// Normalized applies the defaults every handler and middleware uses: a nil Info
// becomes Default(), and a header prefix that is not a token becomes "X-".
func (c HandlerConfig) Normalized() HandlerConfig {
	if c.Info == nil {
		c.Info = Default()
	}
	c.HeaderPrefix = normalizeHeaderPrefix(c.HeaderPrefix)
	return c
}

// ResolveConfig turns a handler's variadic config argument into the effective
// configuration: the first element if given, otherwise DefaultHandlerConfig,
// with Normalized applied either way.
func ResolveConfig(config ...HandlerConfig) HandlerConfig {
	cfg := DefaultHandlerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	return cfg.Normalized()
}

// Headers returns the version headers to emit, already sanitized and keyed by
// full header name. Adapters set these verbatim, so every framework emits the
// same headers under the same build-detail policy.
func (c HandlerConfig) Headers() map[string]string {
	info := c.Payload()
	prefix := normalizeHeaderPrefix(c.HeaderPrefix)

	h := map[string]string{prefix + "Version": sanitizeHeaderValue(info.Version)}
	if info.Commit != "" && info.Commit != "unknown" {
		h[prefix+"Commit"] = sanitizeHeaderValue(info.ShortCommit())
	}
	if info.Branch != "" {
		h[prefix+"Branch"] = sanitizeHeaderValue(info.Branch)
	}
	if info.BuildDate != "" && info.BuildDate != "unknown" {
		h[prefix+"Build-Date"] = sanitizeHeaderValue(info.BuildDate)
	}
	return h
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

func sanitizeHeaderValue(value string) string {
	return strings.Map(func(r rune) rune {
		if r <= 31 || r == 127 {
			return -1
		}
		return r
	}, value)
}
