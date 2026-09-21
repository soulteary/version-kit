# Version Kit

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/version-kit/v4.svg)](https://pkg.go.dev/github.com/soulteary/version-kit/v4)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/version-kit/graph/badge.svg)](https://codecov.io/gh/soulteary/version-kit)

[中文文档](README_CN.md)

A version information management toolkit for Go applications. Provides structured
version info, HTTP endpoints, and middleware for both net/http and Fiber. The root
package depends on nothing outside the standard library and does not import
`net/http` — the handlers live in the `httpadapter` and `fiberadapter`
subpackages, so a CLI that only prints `--version` links neither.


> **Breaking in v4.0.0 — new module path, and the net/http handlers moved to a
> subpackage.**
>
> **Step 1 — everyone, including services that serve no HTTP.** The module path
> is now `github.com/soulteary/version-kit/v4`, **and so is the `-ldflags -X`
> path**, which fails silently if you miss it — the build succeeds, exits 0 and
> reports `dev`:
>
> ```bash
> go get github.com/soulteary/version-kit/v4
> go mod edit -droprequire github.com/soulteary/version-kit/v3
> ```
>
> **Step 2 — net/http users only.** The six net/http entry points moved to
> `github.com/soulteary/version-kit/v4/httpadapter`, keeping their names, so
> importing the root package no longer links a web server into a binary that
> never starts one. For a program that imports only the root package — a CLI
> printing `--version`, which is most of what this kit is used for — that is
> **124 fewer linked packages and a 53% smaller binary** (200 → 76 packages,
> 3,723,527 → 1,745,056 bytes, `-trimpath -ldflags="-s -w"` on linux/amd64).
>
> | Before | After |
> |---|---|
> | `version.Handler(c)` | `httpadapter.Handler(c)` |
> | `version.TextHandler(c)` | `httpadapter.TextHandler(c)` |
> | `version.SimpleHandler()` | `httpadapter.SimpleHandler()` |
> | `version.Middleware(info, prefix)` | `httpadapter.Middleware(info, prefix)` |
> | `version.MiddlewareWithConfig(c)` | `httpadapter.MiddlewareWithConfig(c)` |
> | `version.RegisterEndpoint(mux, path, c)` | `httpadapter.RegisterEndpoint(mux, path, c)` |
>
> **`HandlerConfig` stayed in the root package**, and so did `ResolveConfig`,
> `DefaultHandlerConfig` and the `Payload` / `TextPayload` / `JSONResponse` /
> `Headers` methods. They decide *what* is served and contain no `net/http`, so
> a service on Echo, Gin or chi keeps building its own two-line adapter against
> the root package with no change at all. Nothing else was removed or resigned:
> every other symbol, and every behaviour, is as it was in v3.0.0.
>
> → **[Migrating from v3](#migrating-from-v3)**

## Features

- **Version Information**: Structured version info with version, commit, build date, branch, and runtime details
- **HTTP Endpoints**: JSON and text format endpoints for version APIs
- **Middleware**: Add version headers to all responses
- **Dual Framework Support**: Works with both net/http and Fiber
- **Pay For What You Import**: the root package depends on nothing outside the
  standard library, `net/http` included — net/http and Fiber each live in their
  own subpackage, so a binary links only the server it actually runs
- **Builder Pattern**: Fluent interface for constructing version info
- **Build-time Injection**: Support for ldflags version injection

## Requirements

- **Go 1.27+** for building and running (`go.mod` declares `go 1.27.0`).
- Fiber APIs (`fiberadapter.Handler`, `fiberadapter.Middleware`, etc.) require Fiber v3.4.0 or later.
- net/http APIs (`httpadapter.Handler`, `httpadapter.Middleware`, etc.) need only the standard library.

This v4 module line targets Fiber v3. Applications that still use Fiber v2 should remain on `github.com/soulteary/version-kit` v1.

## Installation

```bash
go get github.com/soulteary/version-kit/v4
```

The root package depends on nothing outside the standard library — not even
`net/http`. Everything that needs a server lives in its own subpackage, so a
binary links only what it actually serves:

```bash
# net/http handlers and middleware — standard library only
go get github.com/soulteary/version-kit/v4/httpadapter

# Fiber v3 handlers — links Fiber, and with it fasthttp
go get github.com/soulteary/version-kit/v4/fiberadapter
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v4"
)

func main() {
    // Create version info
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    // Print version string
    fmt.Println(info.String()) // Output: 1.0.0 (abc123)
    
    // Print full version info
    fmt.Println(info.Full())
    
    // Get as JSON
    fmt.Println(info.JSON())
}
```

### Using Package Variables with ldflags

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v4"
)

func main() {
    // Use default package variables
    // Set during build with ldflags
    info := version.Default()
    fmt.Println(info.String())
}
```

Build with version info:

```bash
go build -ldflags "\
  -X github.com/soulteary/version-kit/v4.Version=1.0.0 \
  -X github.com/soulteary/version-kit/v4.Commit=$(git rev-parse HEAD) \
  -X github.com/soulteary/version-kit/v4.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/soulteary/version-kit/v4.Branch=$(git rev-parse --abbrev-ref HEAD)" \
  -o myapp
```

For a short commit hash only, use `git rev-parse --short HEAD` instead of `git rev-parse HEAD` for the `Commit` variable.

### HTTP Endpoint (net/http)

```go
package main

import (
    "net/http"
    
    version "github.com/soulteary/version-kit/v4"
    "github.com/soulteary/version-kit/v4/httpadapter"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    mux := http.NewServeMux()
    
    // Register JSON endpoint
    httpadapter.RegisterEndpoint(mux, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // Or use handler directly
    mux.HandleFunc("/v", httpadapter.Handler(version.HandlerConfig{Info: info}))
    
    // Text format endpoint
    mux.HandleFunc("/version.txt", httpadapter.TextHandler(version.HandlerConfig{Info: info}))
    
    // Simple version string
    mux.HandleFunc("/v/simple", httpadapter.SimpleHandler())
    
    http.ListenAndServe(":8080", mux)
}
```

### HTTP Endpoint (Fiber)

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    version "github.com/soulteary/version-kit/v4"
    "github.com/soulteary/version-kit/v4/fiberadapter"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    app := fiber.New()
    
    // Register JSON endpoint
    fiberadapter.RegisterEndpoint(app, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // Or use handler directly
    app.Get("/v", fiberadapter.Handler(version.HandlerConfig{Info: info}))
    
    // Text format endpoint
    app.Get("/version.txt", fiberadapter.TextHandler(version.HandlerConfig{Info: info}))
    
    // Simple version string
    app.Get("/v/simple", fiberadapter.SimpleHandler())
    
    app.Listen(":3000")
}
```

### Version Headers Middleware

Add version information to all response headers:

```go
package main

import (
    "net/http"
    
    version "github.com/soulteary/version-kit/v4"
    "github.com/soulteary/version-kit/v4/httpadapter"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello"))
    })
    
    // Wrap with version middleware
    wrapped := httpadapter.Middleware(info, "X-")(handler)
    
    // All responses will have:
    // X-Version: 1.0.0
    // X-Branch:  main     (when set)
    
    http.ListenAndServe(":8080", wrapped)
}
```

These headers ride on **every** response, so `Middleware` emits only the
public fields. `X-Commit` and `X-Build-Date` are an opt-in, exactly as they
are for the handlers:

```go
wrapped := httpadapter.MiddlewareWithConfig(version.HandlerConfig{
    Info:                info,
    HeaderPrefix:        "X-",
    IncludeBuildDetails: true, // adds X-Commit and X-Build-Date
})(handler)
```

Use it on internal routes, or behind authentication: the commit and build
date let anyone match a published CVE to the exact build serving them.

For Fiber:

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    version "github.com/soulteary/version-kit/v4"
    "github.com/soulteary/version-kit/v4/fiberadapter"
)

func main() {
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    app := fiber.New()
    
    // Add public version headers to all responses.
    // For X-Commit / X-Build-Date use:
    //   fiberadapter.MiddlewareWithConfig(version.HandlerConfig{
    //       Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true})
    app.Use(fiberadapter.Middleware(info, "X-"))
    
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendString("Hello")
    })
    
    app.Listen(":3000")
}
```

### Builder Pattern

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v4"
)

func main() {
    info := version.NewBuilder().
        WithVersion("1.0.0").
        WithCommit("abc123").
        WithBuildDate("2025-01-01T00:00:00Z").
        WithBranch("main").
        Build()
    
    fmt.Println(info.String())
}
```

### Include Headers in Endpoint Response

```go
httpadapter.RegisterEndpoint(mux, "/version", version.HandlerConfig{
    Info:           info,
    IncludeHeaders: true,  // Add version headers to response
    HeaderPrefix:   "X-App-", // Custom prefix
})
```

## API Reference

### Constructing Info

| Function | Description |
|----------|-------------|
| `New(version, commit, buildDate string) *Info` | Creates version info with the given version, commit, and build date. Runtime fields (Go version, platform, compiler) are set automatically. |
| `NewWithBranch(version, commit, buildDate, branch string) *Info` | Like `New` but also sets the branch name. |
| `Default() *Info` | Returns info from package variables (Version, Commit, BuildDate, Branch), typically set via ldflags. |
| `NewBuilder() *Builder` | Returns a builder for constructing `Info` with a fluent API. |

### Serving it over net/http

`github.com/soulteary/version-kit/v4/httpadapter` — standard library only.
Importing **this** package is what puts `net/http` in your binary; the root
package does not.

| Function | Description |
|----------|-------------|
| `httpadapter.Handler(config ...version.HandlerConfig) http.HandlerFunc` | Serves the version info as JSON. |
| `httpadapter.TextHandler(config ...version.HandlerConfig) http.HandlerFunc` | Serves it as plain text. |
| `httpadapter.SimpleHandler() http.HandlerFunc` | Serves just the version string, read from the package variables per request. |
| `httpadapter.RegisterEndpoint(mux *http.ServeMux, path string, config ...version.HandlerConfig)` | Registers `Handler` on a `ServeMux`. |
| `httpadapter.Middleware(info *version.Info, prefix string) func(http.Handler) http.Handler` | Adds the **public** version headers (`X-Version`, `X-Branch`) to every response. |
| `httpadapter.MiddlewareWithConfig(config version.HandlerConfig) func(http.Handler) http.Handler` | Same, with the full `HandlerConfig`; set `IncludeBuildDetails` for `X-Commit` and `X-Build-Date`. |

`DefaultHandlerConfig() HandlerConfig` stays in the root package with the rest
of the configuration API — it is what every adapter reads, not something
net/http owns.

### Serving it over Fiber

`github.com/soulteary/version-kit/v4/fiberadapter` — the same set, serving the
same bytes, returning `fiber.Handler`. Importing **this** package is what links
Fiber, and with it fasthttp, into your binary; the root package does not.

| Function | httpadapter counterpart |
|----------|----------------------|
| `fiberadapter.Handler(config ...version.HandlerConfig) fiber.Handler` | `Handler` |
| `fiberadapter.TextHandler(config ...version.HandlerConfig) fiber.Handler` | `TextHandler` |
| `fiberadapter.SimpleHandler() fiber.Handler` | `SimpleHandler` |
| `fiberadapter.RegisterEndpoint(app *fiber.App, path string, config ...version.HandlerConfig)` | `RegisterEndpoint` |
| `fiberadapter.Middleware(info *version.Info, prefix string) fiber.Handler` | `Middleware` |
| `fiberadapter.MiddlewareWithConfig(config version.HandlerConfig) fiber.Handler` | `MiddlewareWithConfig` |

### Serving it over anything else

Echo, Gin, chi — the rules that decide *what* gets served are methods on
`HandlerConfig`, so an adapter reads them instead of restating them and cannot
drift from the handlers above. `fiberadapter` is written against exactly these
and nothing more.

| Method | Returns |
|--------|---------|
| `ResolveConfig(config ...HandlerConfig) HandlerConfig` | The effective config for a variadic handler argument: the first element, or `DefaultHandlerConfig()`, `Normalized` either way. |
| `(c HandlerConfig) Normalized() HandlerConfig` | The config with a nil `Info` replaced by `Default()` and a `HeaderPrefix` that is not a valid HTTP token replaced by `"X-"`. |
| `(c HandlerConfig) Payload() *Info` | The `Info` to serve, reduced to the public fields unless `IncludeBuildDetails` is set. |
| `(c HandlerConfig) TextPayload() string` | `Payload` rendered for a plain-text endpoint. |
| `(c HandlerConfig) JSONResponse() (body []byte, status int)` | `Payload` marshalled — indented when `Pretty` is set — and the status to send it with. |
| `(c HandlerConfig) Headers() map[string]string` | The version headers to emit, already sanitized and keyed by full header name. Respects `IncludeBuildDetails`. |

A whole adapter is this:

```go
func Handler(config ...version.HandlerConfig) echo.HandlerFunc {
    // Neither of these changes per request, so compute them once here.
    cfg := version.ResolveConfig(config...)
    headers := cfg.Headers()
    body, status := cfg.JSONResponse()

    return func(c echo.Context) error {
        if cfg.IncludeHeaders {
            for name, value := range headers {
                c.Response().Header().Set(name, value)
            }
        }
        return c.Blob(status, "application/json", body)
    }
}
```

### Info Methods

| Method | Description |
|--------|-------------|
| `String()` | Returns version with short commit (e.g., "1.0.0 (abc1234)") |
| `Full()` | Returns detailed multi-line version info |
| `JSON()` | Returns JSON representation |
| `JSONPretty()` | Returns pretty-printed JSON |
| `Map()` | Returns version info as map[string]string |
| `Validate()` | Validates required fields |
| `IsDev()` | Returns true if version is "dev" or empty |
| `BuildTimestamp()` | Parses build date as time.Time |
| `ShortCommit()` | Returns first 7 characters of commit |
| `Public()` | A copy carrying only the fields safe for an unauthenticated endpoint: the version, and the branch when set |

### HandlerConfig Options

Use `DefaultHandlerConfig()` to get a config with default values when you only want to override specific fields.

```go
type HandlerConfig struct {
    Info                *Info  // Version info (default: Default())
    Pretty              bool   // Pretty-print JSON (default: false)
    IncludeHeaders      bool   // Add version headers (default: false)
    HeaderPrefix        string // Header prefix (default: "X-")
    IncludeBuildDetails bool   // Serve the full Info (default: false)
}
```

### Package Variables

Set these via ldflags at build time:

```go
var (
    Version   = "dev"      // Application version
    Commit    = "unknown"  // Git commit hash
    BuildDate = "unknown"  // Build timestamp
    Branch    = ""         // Git branch name
)
```

## Example Response

These show the **full** response, which needs `IncludeBuildDetails: true`.
By default the endpoint serves only `version` and `branch` -- see
[Build Details](#build-details).

### JSON Endpoint

```json
{
  "version": "1.0.0",
  "commit": "abc123def456",
  "build_date": "2025-01-01T00:00:00Z",
  "branch": "main",
  "go_version": "go1.26",
  "platform": "linux/amd64",
  "compiler": "gc"
}
```

### Text Endpoint

```
Version:    1.0.0
Commit:     abc123def456
Branch:     main
Built:      2025-01-01T00:00:00Z
Go version: go1.26
Platform:   linux/amd64
Compiler:   gc
```

## Testing

What CI runs:

```bash
gofmt -s -l .
go vet ./...
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
golangci-lint run --timeout=5m
govulncheck ./...

go tool cover -func=coverage.out   # per-function summary
go tool cover -html=coverage.out   # annotated source
```

`-covermode=atomic` is required alongside `-race`; the default `set` mode is
not race-safe and Codecov reads the atomic counts.

One of those tests is not about behaviour. `TestRootPackageStaysDependencyFree`
reads `go list -deps .` and fails if the root package has picked up `net/http`,
Fiber or fasthttp again. That is the only thing standing between the root
package and a re-import: adding `net/http` back — for an `http.StatusOK`
constant, say — compiles and passes everything else, while quietly putting 124
packages back into every importer's binary. Test files are exempt by
construction, since `go list -deps .` reports the package's own import graph
and not its tests'.

## Build Details

`IncludeBuildDetails` is **false by default**, so the endpoint serves only
`version` and `branch`:

```json
{"version":"1.2.3","branch":"main"}
```

This endpoint is usually unauthenticated, and `go_version` lets anyone match a
published Go runtime CVE to the exact build serving them, while the commit and
build date fingerprint your deployment. Turn it on for an internal endpoint, or
behind authentication, to get the full response back:

```go
httpadapter.Handler(version.HandlerConfig{
    Info:                version.Default(),
    IncludeBuildDetails: true,
})
```

```json
{"version":"1.2.3","branch":"main","commit":"abc1234","build_date":"2026-01-02T03:04:05Z","go_version":"go1.27.0",...}
```

The same flag gates the `X-*-Commit` and `X-*-Build-Date` response headers, so
`IncludeHeaders` alone does not expose them.

Building your own response? `Info.Public()` returns the reduced copy:

```go
json.NewEncoder(w).Encode(version.Default().Public())
```

`Full()` skips the `Go version:`, `Platform:` and `Compiler:` labels when those
fields are unset, so a reduced `Info` renders without a run of empty lines.

### Header prefix

`HeaderPrefix` is concatenated into a header **name**, so a prefix containing a
space, colon or newline would produce a malformed header. An invalid prefix falls
back to `"X-"`.

## Migrating from v3

### 1. The module path

```bash
go get github.com/soulteary/version-kit/v4
go mod edit -droprequire github.com/soulteary/version-kit/v3
```

Then rewrite every import:

```diff
-version "github.com/soulteary/version-kit/v3"
+version "github.com/soulteary/version-kit/v4"
```

This applies to **everyone**, including binaries that serve no HTTP at all.
`go get -u` will not do it for you — that is what a new major version means.
v3 stays where it is on `v3.0.0`.

### 2. The ldflags path — this one fails silently, again

The `-X` paths carry the module path too, and a wrong one is not an error: the
linker has nothing to write to, so the build succeeds, exits 0 and the binary
reports `dev`. Grep your Makefile, Dockerfile and release workflow for
`version-kit/v3` — not just your `.go` files.

```bash
# WRONG after upgrading: builds fine, exits 0, reports "dev"
go build -ldflags "-X github.com/soulteary/version-kit/v3.Version=1.0.0" .

# Right
go build -ldflags "-X github.com/soulteary/version-kit/v4.Version=1.0.0" .
```

### 3. The net/http functions moved

Six symbols left the root package for `httpadapter`, keeping their names:

```diff
 import (
     version "github.com/soulteary/version-kit/v4"
+    "github.com/soulteary/version-kit/v4/httpadapter"
 )

-mux.HandleFunc("/version", version.Handler(version.HandlerConfig{Info: info}))
+mux.HandleFunc("/version", httpadapter.Handler(version.HandlerConfig{Info: info}))
```

| v3 | v4 |
|----|----|
| `version.Handler(...)` | `httpadapter.Handler(...)` |
| `version.TextHandler(...)` | `httpadapter.TextHandler(...)` |
| `version.SimpleHandler()` | `httpadapter.SimpleHandler()` |
| `version.RegisterEndpoint(...)` | `httpadapter.RegisterEndpoint(...)` |
| `version.Middleware(...)` | `httpadapter.Middleware(...)` |
| `version.MiddlewareWithConfig(...)` | `httpadapter.MiddlewareWithConfig(...)` |

Keeping them in place as shims was not an option: a shim imports `net/http`,
which is the entire thing being moved out.

### 4. If you are not on net/http, there is no step 3

`HandlerConfig`, `ResolveConfig`, `DefaultHandlerConfig`, `Info`, `New`,
`Default`, `NewBuilder` and the `Payload` / `TextPayload` / `JSONResponse` /
`Headers` / `Normalized` methods all stayed in the root package, unchanged. A
CLI, or a service on Echo, Gin or chi that builds its own adapter the way
[Serving it over anything else](#serving-it-over-anything-else) describes,
needs step 1 and step 2 and nothing more.

`fiberadapter` users likewise: it reads the same root-package configuration it
always did, so only its import path changes.

### 5. Nothing else changed

No behaviour, no signature, no response byte. v4 is the module path and the
package boundary; that is all of it.

## Migrating from v2

### 1. The module path

```bash
go get github.com/soulteary/version-kit/v4
```

Then rewrite every import:

```diff
-version "github.com/soulteary/version-kit/v2"
+version "github.com/soulteary/version-kit/v4"
```

This applies to **everyone**, including net/http-only services that never touch
a Fiber handler. `go get -u` will not do it for you — that is what a new major
version means. v2 stays where it is on `v2.2.0`.

### 2. The ldflags path — this one fails silently

`-X` names a package-level variable by its full import path. Point it at a
package that no longer exists and the linker says nothing:

```bash
# WRONG after upgrading: builds fine, exits 0, reports "dev"
go build -ldflags "-X github.com/soulteary/version-kit/v2.Version=1.0.0" .

# right
go build -ldflags "-X github.com/soulteary/version-kit/v4.Version=1.0.0" .
```

There is no error and no warning. The binary builds, passes CI, ships, and
serves `{"version":"dev"}`. Grep your build scripts, Makefiles, Dockerfiles and
CI workflows for the old path before you tag a release.

### 3. The Fiber functions moved

```go
import (
    version "github.com/soulteary/version-kit/v4"
    "github.com/soulteary/version-kit/v4/fiberadapter"
)
```

| Before | After |
|---|---|
| `version.FiberHandler(...)` | `fiberadapter.Handler(...)` |
| `version.FiberTextHandler(...)` | `fiberadapter.TextHandler(...)` |
| `version.FiberSimpleHandler()` | `fiberadapter.SimpleHandler()` |
| `version.FiberMiddleware(...)` | `fiberadapter.Middleware(...)` |
| `version.FiberMiddlewareWithConfig(...)` | `fiberadapter.MiddlewareWithConfig(...)` |
| `version.RegisterEndpointFiber(...)` | `fiberadapter.RegisterEndpoint(...)` |

### 4. Behaviour changes

**Both frameworks — `Info` is read once, when the handler is built.** The
response body and the version headers are computed at construction and reused
on every response, so mutating an `Info` after handing it to a handler no
longer changes what is served:

```go
info := version.New("1.0.0", "abc1234", "")
h := httpadapter.Handler(version.HandlerConfig{Info: info})
info.Version = "2.0.0"   // v2 served 2.0.0 here; v3 serves 1.0.0
```

Finish building the `Info` before you build the handler. `SimpleHandler` is the
exception — it takes no config and still reads the package variables per
request.

**Fiber only — `Content-Type` is now `application/json`,** matching net/http.
Fiber used to answer `application/json; charset=utf-8`; RFC 8259 defines no
charset parameter for `application/json`.

**Fiber only — `Pretty` now works.** `fiber.Ctx.JSON` always writes compact
JSON, so a Fiber endpoint configured with `Pretty: true` was silently served
compact and now comes back indented.

**Fiber only — the body no longer goes through the app's `JSONEncoder`.**
version-kit encodes it, which is what makes the two frameworks byte-for-byte
identical; a custom Fiber encoder no longer applies to this endpoint.

### 5. Go 1.27+

`go.mod` declares `go 1.27.0`. v3 will not build on an older toolchain.

## Earlier: build details became opt-in (v2.2.0)

These notes describe a change that landed in v2.2.0 and still holds in v3. If
you are coming from v2.2.0 or later, you have already dealt with it.

**The default endpoint response is smaller.** That is the fix, and it is the one
thing to check before upgrading.

- **Build details are withheld by default.** `Handler`, `fiberadapter.Handler`,
  `TextHandler` and the version-header middleware served the full `Info` — Go
  runtime version, commit, build date, platform and compiler. The endpoint is
  usually unauthenticated, and `Middleware` put the same data on **every
  response**, so `go_version` let anyone match a published Go runtime CVE to the
  exact build serving them, with the commit and build date narrowing it further.
  The default response is now `{"version":…,"branch":…}`. **If your tooling parses
  `commit`, `build_date` or `go_version` from `/version`, set
  `IncludeBuildDetails: true`** and put that endpoint behind authentication or on
  an internal route.
- **`MiddlewareWithConfig` and `fiberadapter.MiddlewareWithConfig` are new.**
  `Middleware(info, prefix)` keeps its signature and now emits only the public
  headers; use the `WithConfig` forms with `IncludeBuildDetails` to get
  `X-Commit` and `X-Build-Date` back.
- **`Info.Public()` is new**, for callers building their own response.
- **`Full()` skips unset fields.** It emitted `Go version:`, `Platform:` and
  `Compiler:` labels unconditionally, so a reduced `Info` rendered with a run of
  empty lines. `Commit`, `Branch` and `BuildDate` were already handled this way.
- **An invalid `HeaderPrefix` falls back to `"X-"`.** It was concatenated into a
  header name with only an empty-string check, so a prefix containing a space,
  colon or newline produced a malformed header.
- **Requirements said Go 1.26**; `go.mod` requires `1.27.0`.

## License

Apache License 2.0
