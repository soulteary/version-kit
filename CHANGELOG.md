# Changelog

Notable changes per release. This file starts at v3.0.0; for the v1 and v2
history before that, see the [tags](https://github.com/soulteary/version-kit/tags).

## v3.0.0

A new major version, which in Go means a new module path. **Every import
changes, and so does the `-ldflags -X` path.** See
[Migrating from v2](README.md#migrating-from-v2) — the ldflags step fails
silently if you miss it.

### Breaking

- **Module path is now `github.com/soulteary/version-kit/v3`.** `go get -u`
  will not move a v2 project here; v2 stays on `v2.2.0`.
- **Fiber support moved to `fiberadapter`.** `FiberHandler`, `FiberTextHandler`,
  `FiberSimpleHandler`, `FiberMiddleware`, `FiberMiddlewareWithConfig` and
  `RegisterEndpointFiber` are gone from the root package; the subpackage has
  the same six without the prefix. Nothing else was removed or resigned.
- **`Info` is read once, when the handler is built.** The response body and the
  version headers are computed at construction and reused, so mutating an
  `Info` after handing it to a handler no longer changes what is served. Build
  the `Info` first. `SimpleHandler` still reads the package variables per
  request.
- **Fiber: `Content-Type` is now `application/json`**, matching net/http, which
  never sent the `charset=utf-8` Fiber did. RFC 8259 defines no charset
  parameter for `application/json`.
- **Fiber: `HandlerConfig.Pretty` now takes effect.** It was silently ignored —
  `fiber.Ctx.JSON` always writes compact JSON — so an endpoint configured with
  `Pretty: true` starts returning indented JSON.
- **Fiber: the JSON body no longer goes through the app's `JSONEncoder`.** It
  is encoded by version-kit, which is what makes the two frameworks
  byte-for-byte identical.

### Why the split

`handler.go` imported `github.com/gofiber/fiber/v3`, so every importer of the
root package linked Fiber — and with it fasthttp — whether or not it ever
touched a Fiber handler. Measured on a net/http program that only calls
`RegisterEndpoint` and `Default()`:

| | v2.2.0 | v3.0.0 |
|---|---|---|
| linked fasthttp/gofiber packages | 25 | **0** |
| binary size | 8020 KB | **6860 KB** (−14%) |
| modules in the build list | 30 | **19** |

### Added

- `HandlerConfig.Payload`, `TextPayload`, `JSONResponse`, `Headers`,
  `Normalized` and `ResolveConfig` are exported, so an out-of-tree adapter for
  Echo, Gin or chi reads the rules that decide what gets served instead of
  restating them. `fiberadapter` is written against exactly these.
- `fiberadapter.Handler`, `TextHandler`, `SimpleHandler`, `RegisterEndpoint`,
  `Middleware` and `MiddlewareWithConfig`.

### Performance

Headers and body are computed when the handler is built rather than per
request. net/http, handler only, ns/op and allocs/op:

| | v2.2.0 | v3.0.0 |
|---|---|---|
| `Handler` | 730.7 / 7 | **199.3 / 3** |
| `TextHandler` | 562.0 / 11 | **201.4 / 3** |
| `Middleware` | 187.1 / 4 | **153.1 / 2** |

### Other

- Requires Go 1.27+ (`go.mod` declares `go 1.27.0`).
- CI actions moved to `checkout@v7`, `setup-go@v7`, `upload-artifact@v7`,
  `codecov-action@v7`, `goreportcard-action@v1.1.2`.
- Coverage: 98.3% root, 100% `fiberadapter`, 98.7% total.

## v2.2.0

- **Build details are withheld by default.** `Handler`, `TextHandler` and the
  version-header middleware served the full `Info` — Go runtime version,
  commit, build date, platform and compiler — on an endpoint that is usually
  unauthenticated. The default response is now `{"version":…,"branch":…}`; set
  `IncludeBuildDetails: true` to get the rest back.
- `MiddlewareWithConfig` and `Info.Public()` added.
- `Full()` skips unset fields instead of emitting empty labels.
- An invalid `HeaderPrefix` falls back to `"X-"` rather than producing a
  malformed header name.

## v2.1.0

- Go 1.27.0.

## v2.0.0

- The v2 module line targets Fiber v3. Applications still on Fiber v2 should
  remain on v1.
