# Version Kit

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/version-kit/v4.svg)](https://pkg.go.dev/github.com/soulteary/version-kit/v4)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/version-kit/graph/badge.svg)](https://codecov.io/gh/soulteary/version-kit)

[English](README.md)

一个用于 Go 应用程序的版本信息管理工具包。提供结构化的版本信息、HTTP 端点和中间件，
同时支持 net/http 和 Fiber 框架。根包不依赖标准库之外的任何东西，连 `net/http` 都
不导入——处理器分别住在 `httpadapter` 与 `fiberadapter` 子包里，于是一个只用来打印
`--version` 的 CLI 两个都不会链接进去。


> **v4.0.0 的破坏性变更——模块路径变了，net/http 处理器也移进了子包。**
>
> **第一步——所有人，包括完全不提供 HTTP 服务的程序。** 模块路径现在是
> `github.com/soulteary/version-kit/v4`，**`-ldflags -X` 的路径也是**，漏掉
> 不会报错：构建成功、退出码 0，然后二进制报告 `dev`。
>
> ```bash
> go get github.com/soulteary/version-kit/v4
> go mod edit -droprequire github.com/soulteary/version-kit/v3
> ```
>
> **第二步——只有 net/http 用户需要。** 六个 net/http 入口移到了
> `github.com/soulteary/version-kit/v4/httpadapter`，名字原样保留，于是导入根包
> 不再把一个从未启动的 Web 服务器链接进二进制。对一个只导入根包的程序——比如
> 打印 `--version` 的 CLI，这也正是这个 kit 最常见的用法——意味着
> **少链接 124 个包、二进制小 53%**（200 → 76 个包，3,723,527 → 1,745,056 字节，
> linux/amd64 上 `-trimpath -ldflags="-s -w"` 实测）。
>
> | 改之前 | 改之后 |
> |---|---|
> | `version.Handler(c)` | `httpadapter.Handler(c)` |
> | `version.TextHandler(c)` | `httpadapter.TextHandler(c)` |
> | `version.SimpleHandler()` | `httpadapter.SimpleHandler()` |
> | `version.Middleware(info, prefix)` | `httpadapter.Middleware(info, prefix)` |
> | `version.MiddlewareWithConfig(c)` | `httpadapter.MiddlewareWithConfig(c)` |
> | `version.RegisterEndpoint(mux, path, c)` | `httpadapter.RegisterEndpoint(mux, path, c)` |
>
> **`HandlerConfig` 没有动**，`ResolveConfig`、`DefaultHandlerConfig` 以及
> `Payload` / `TextPayload` / `JSONResponse` / `Headers` 这几个方法也没有。它们
> 决定的是「提供什么内容」，里面没有 `net/http`，所以跑在 Echo、Gin、chi 上、
> 自己写两行适配器的服务一行都不用改。除此之外没有删改任何东西：其余符号与行为
> 与 v3.0.0 完全一致。
>
> → **[从 v3 迁移](#从-v3-迁移)**

## 功能特性

- **版本信息**: 结构化的版本信息，包含版本号、提交哈希、构建日期、分支和运行时详情
- **HTTP 端点**: JSON 和文本格式的版本 API 端点
- **中间件**: 为所有响应添加版本头信息
- **双框架支持**: 同时支持 net/http 和 Fiber
- **用多少付多少**: 根包不依赖标准库之外的任何东西，`net/http` 也不例外——net/http
  与 Fiber 各自住在自己的子包里，二进制只链接它真正跑起来的那个服务器
- **构建器模式**: 流式接口构建版本信息
- **构建时注入**: 支持通过 ldflags 注入版本信息

## 运行要求

- **Go 1.27+**，用于构建与运行（`go.mod` 声明 `go 1.27.0`）。
- Fiber API（`fiberadapter.Handler`、`fiberadapter.Middleware` 等）要求 Fiber v3.4.0 或更高版本。
- net/http API（`httpadapter.Handler`、`httpadapter.Middleware` 等）只需要标准库。

此 v4 模块版本面向 Fiber v3。仍使用 Fiber v2 的应用应继续使用 `github.com/soulteary/version-kit` v1。

## 安装

```bash
go get github.com/soulteary/version-kit/v4
```

根包不依赖标准库之外的任何东西，连 `net/http` 都不依赖。需要服务器的部分都在各自的
子包里，二进制只链接它真正对外提供的那一种：

```bash
# net/http 处理器与中间件——只用标准库
go get github.com/soulteary/version-kit/v4/httpadapter

# Fiber v3 处理器——会链接 Fiber，连带 fasthttp
go get github.com/soulteary/version-kit/v4/fiberadapter
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v4"
)

func main() {
    // 创建版本信息
    info := version.New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
    
    // 打印版本字符串
    fmt.Println(info.String()) // 输出: 1.0.0 (abc123)
    
    // 打印完整版本信息
    fmt.Println(info.Full())
    
    // 获取 JSON 格式
    fmt.Println(info.JSON())
}
```

### 使用 ldflags 设置包变量

```go
package main

import (
    "fmt"
    
    version "github.com/soulteary/version-kit/v4"
)

func main() {
    // 使用默认包变量
    // 在构建时通过 ldflags 设置
    info := version.Default()
    fmt.Println(info.String())
}
```

构建时注入版本信息:

```bash
go build -ldflags "\
  -X github.com/soulteary/version-kit/v4.Version=1.0.0 \
  -X github.com/soulteary/version-kit/v4.Commit=$(git rev-parse HEAD) \
  -X github.com/soulteary/version-kit/v4.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/soulteary/version-kit/v4.Branch=$(git rev-parse --abbrev-ref HEAD)" \
  -o myapp
```

若只需短提交哈希，可将 `Commit` 变量中的 `git rev-parse HEAD` 改为 `git rev-parse --short HEAD`。

### HTTP 端点 (net/http)

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
    
    // 注册 JSON 端点
    httpadapter.RegisterEndpoint(mux, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // 或直接使用处理器
    mux.HandleFunc("/v", httpadapter.Handler(version.HandlerConfig{Info: info}))
    
    // 文本格式端点
    mux.HandleFunc("/version.txt", httpadapter.TextHandler(version.HandlerConfig{Info: info}))
    
    // 简单版本字符串
    mux.HandleFunc("/v/simple", httpadapter.SimpleHandler())
    
    http.ListenAndServe(":8080", mux)
}
```

### HTTP 端点 (Fiber)

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
    
    // 注册 JSON 端点
    fiberadapter.RegisterEndpoint(app, "/version", version.HandlerConfig{
        Info:   info,
        Pretty: true,
    })
    
    // 或直接使用处理器
    app.Get("/v", fiberadapter.Handler(version.HandlerConfig{Info: info}))
    
    // 文本格式端点
    app.Get("/version.txt", fiberadapter.TextHandler(version.HandlerConfig{Info: info}))
    
    // 简单版本字符串
    app.Get("/v/simple", fiberadapter.SimpleHandler())
    
    app.Listen(":3000")
}
```

### 版本头信息中间件

为所有响应添加版本信息头:

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
    
    // 使用版本中间件包装
    wrapped := httpadapter.Middleware(info, "X-")(handler)
    
    // 所有响应将包含:
    // X-Version: 1.0.0
    // X-Branch:  main     (设置了分支时)
    
    http.ListenAndServe(":8080", wrapped)
}
```

这些响应头会出现在**每一个**响应上，因此 `Middleware` 只输出公开字段。
与处理器一致，`X-Commit` 与 `X-Build-Date` 需要显式开启：

```go
wrapped := httpadapter.MiddlewareWithConfig(version.HandlerConfig{
    Info:                info,
    HeaderPrefix:        "X-",
    IncludeBuildDetails: true, // 增加 X-Commit 与 X-Build-Date
})(handler)
```

请只在内部路由或鉴权之后开启：提交号与构建时间足以让任何人把已公开的
CVE 对应到正在为其服务的具体构建。

Fiber 版本:

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
    
    // 为所有响应添加公开版本响应头。
    // 需要 X-Commit / X-Build-Date 时改用：
    //   fiberadapter.MiddlewareWithConfig(version.HandlerConfig{
    //       Info: info, HeaderPrefix: "X-", IncludeBuildDetails: true})
    app.Use(fiberadapter.Middleware(info, "X-"))
    
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendString("Hello")
    })
    
    app.Listen(":3000")
}
```

### 构建器模式

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

### 在端点响应中包含头信息

```go
httpadapter.RegisterEndpoint(mux, "/version", version.HandlerConfig{
    Info:           info,
    IncludeHeaders: true,     // 在响应中添加版本头信息
    HeaderPrefix:   "X-App-", // 自定义前缀
})
```

## API 参考

### 构造 Info

| 函数 | 描述 |
|------|------|
| `New(version, commit, buildDate string) *Info` | 根据版本号、提交哈希和构建日期创建版本信息，运行时字段（Go 版本、平台、编译器）会自动填充。 |
| `NewWithBranch(version, commit, buildDate, branch string) *Info` | 与 `New` 类似，同时设置分支名。 |
| `Default() *Info` | 使用包变量（Version、Commit、BuildDate、Branch）构造信息，通常由 ldflags 在构建时注入。 |
| `NewBuilder() *Builder` | 返回用于以流式 API 构建 `Info` 的 Builder。 |

### 用 net/http 提供服务

`github.com/soulteary/version-kit/v4/httpadapter` —— 只用标准库。把 `net/http`
链进二进制的是**这个**包，根包不会。

| 函数 | 描述 |
|------|------|
| `httpadapter.Handler(config ...version.HandlerConfig) http.HandlerFunc` | 以 JSON 提供版本信息。 |
| `httpadapter.TextHandler(config ...version.HandlerConfig) http.HandlerFunc` | 以纯文本提供版本信息。 |
| `httpadapter.SimpleHandler() http.HandlerFunc` | 只返回版本字符串，每次请求都从包变量读取。 |
| `httpadapter.RegisterEndpoint(mux *http.ServeMux, path string, config ...version.HandlerConfig)` | 把 `Handler` 注册到 `ServeMux` 上。 |
| `httpadapter.Middleware(info *version.Info, prefix string) func(http.Handler) http.Handler` | 为每个响应添加**公开**版本响应头（`X-Version`、`X-Branch`）。 |
| `httpadapter.MiddlewareWithConfig(config version.HandlerConfig) func(http.Handler) http.Handler` | 同上，接受完整 `HandlerConfig`；设置 `IncludeBuildDetails` 可输出 `X-Commit` 与 `X-Build-Date`。 |

`DefaultHandlerConfig() HandlerConfig` 留在根包，和其余配置 API 在一起——它是每个
适配器都要读的东西，不归 net/http 所有。

### 用 Fiber 提供服务

`github.com/soulteary/version-kit/v4/fiberadapter` —— 同一套函数、返回同样的
字节，类型换成 `fiber.Handler`。把 Fiber 链接进二进制的是**这个**包，根包不会。

| 函数 | 对应的 net/http 版本 |
|------|----------------------|
| `fiberadapter.Handler(config ...version.HandlerConfig) fiber.Handler` | `Handler` |
| `fiberadapter.TextHandler(config ...version.HandlerConfig) fiber.Handler` | `TextHandler` |
| `fiberadapter.SimpleHandler() fiber.Handler` | `SimpleHandler` |
| `fiberadapter.RegisterEndpoint(app *fiber.App, path string, config ...version.HandlerConfig)` | `RegisterEndpoint` |
| `fiberadapter.Middleware(info *version.Info, prefix string) fiber.Handler` | `Middleware` |
| `fiberadapter.MiddlewareWithConfig(config version.HandlerConfig) fiber.Handler` | `MiddlewareWithConfig` |

### 用其他框架提供服务

Echo、Gin、chi —— 决定**服务什么**的规则都是 `HandlerConfig` 上的方法，适配器
读取它们而不是各写一遍，因此不会与上面的 handler 产生分歧。`fiberadapter` 本身
就只用了这些，没有别的。

| 方法 | 返回 |
|------|------|
| `ResolveConfig(config ...HandlerConfig) HandlerConfig` | 变长 config 参数的最终配置：有就取第一个，没有就用 `DefaultHandlerConfig()`，两种情况都会经过 `Normalized`。 |
| `(c HandlerConfig) Normalized() HandlerConfig` | 把 nil 的 `Info` 换成 `Default()`、把不是合法 HTTP token 的 `HeaderPrefix` 换成 `"X-"` 之后的配置。 |
| `(c HandlerConfig) Payload() *Info` | 要服务的 `Info`；未设置 `IncludeBuildDetails` 时精简为公开字段。 |
| `(c HandlerConfig) TextPayload() string` | `Payload` 渲染成纯文本端点的形式。 |
| `(c HandlerConfig) JSONResponse() (body []byte, status int)` | `Payload` 序列化后的结果（`Pretty` 时缩进），以及随之发送的状态码。 |
| `(c HandlerConfig) Headers() map[string]string` | 要输出的版本响应头，已消毒、以完整头名为键。遵循 `IncludeBuildDetails`。 |

一个完整的适配器就是这样：

```go
func Handler(config ...version.HandlerConfig) echo.HandlerFunc {
    // 这两样都不随请求变化，在这里算一次即可。
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

### Info 方法

| 方法 | 描述 |
|------|------|
| `String()` | 返回带短提交哈希的版本 (例如 "1.0.0 (abc1234)") |
| `Full()` | 返回详细的多行版本信息 |
| `JSON()` | 返回 JSON 表示 |
| `JSONPretty()` | 返回格式化的 JSON |
| `Map()` | 返回 map[string]string 格式的版本信息 |
| `Validate()` | 验证必填字段 |
| `IsDev()` | 如果版本是 "dev" 或空则返回 true |
| `BuildTimestamp()` | 解析构建日期为 time.Time |
| `ShortCommit()` | 返回提交哈希的前 7 个字符 |
| `Public()` | 返回一个副本，只保留适合在未认证端点暴露的字段：版本号，以及已设置时的分支名 |

### HandlerConfig 选项

仅需覆盖部分字段时，可使用 `DefaultHandlerConfig()` 获取默认配置后再修改。

```go
type HandlerConfig struct {
    Info                *Info  // 版本信息 (默认: Default())
    Pretty              bool   // 格式化 JSON 输出 (默认: false)
    IncludeHeaders      bool   // 添加版本头信息 (默认: false)
    HeaderPrefix        string // 头信息前缀 (默认: "X-")
    IncludeBuildDetails bool   // 返回完整 Info (默认: false)
}
```

### 包变量

在构建时通过 ldflags 设置这些变量:

```go
var (
    Version   = "dev"      // 应用版本
    Commit    = "unknown"  // Git 提交哈希
    BuildDate = "unknown"  // 构建时间戳
    Branch    = ""         // Git 分支名
)
```

## 响应示例

以下是**完整**响应，需要设置 `IncludeBuildDetails: true`。
默认情况下端点只返回 `version` 和 `branch`，详见[构建详情](#构建详情)。

### JSON 端点

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

### 文本端点

```
Version:    1.0.0
Commit:     abc123def456
Branch:     main
Built:      2025-01-01T00:00:00Z
Go version: go1.26
Platform:   linux/amd64
Compiler:   gc
```

## 构建详情

`IncludeBuildDetails` **默认为 false**，因此端点只返回 `version` 和 `branch`：

```json
{"version":"1.2.3","branch":"main"}
```

这个端点通常是不需要认证的，而 `go_version` 会让任何人把已公开的 Go 运行时 CVE
精确对应到正在服务的这个构建上，commit 与构建时间则等于给部署留下指纹。
仅在内部端点、或已有认证保护的端点上开启它，以取回完整响应：

```go
httpadapter.Handler(version.HandlerConfig{
    Info:                version.Default(),
    IncludeBuildDetails: true,
})
```

```json
{"version":"1.2.3","branch":"main","commit":"abc1234","build_date":"2026-01-02T03:04:05Z","go_version":"go1.27.0",...}
```

同一个开关也控制 `X-*-Commit` 与 `X-*-Build-Date` 响应头，因此单独开启
`IncludeHeaders` 并不会暴露它们。

要自己拼响应？`Info.Public()` 会返回这个精简副本：

```go
json.NewEncoder(w).Encode(version.Default().Public())
```

当 `Go version:`、`Platform:`、`Compiler:` 这些字段未设置时，`Full()` 会跳过对应的标签，
因此精简后的 `Info` 渲染出来不会带一串空行。

### 头前缀

`HeaderPrefix` 会被拼进 header **名称**里，因此含空格、冒号或换行的前缀会产出一个畸形的
header。非法前缀会回退为 `"X-"`。

## 测试

CI 实际跑的是：

```bash
gofmt -s -l .
go vet ./...
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

go tool cover -func=coverage.out   # 按函数汇总
go tool cover -html=coverage.out   # 带标注的源码
```

`-covermode=atomic` 在搭配 `-race` 时是必需的：默认的 `set` 模式不是 race-safe，
而且 Codecov 读的就是 atomic 计数。

## 从 v3 迁移

### 1. 模块路径

```bash
go get github.com/soulteary/version-kit/v4
go mod edit -droprequire github.com/soulteary/version-kit/v3
```

然后改掉所有 import：

```diff
-version "github.com/soulteary/version-kit/v3"
+version "github.com/soulteary/version-kit/v4"
```

这一条**所有人**都要做，包括完全不提供 HTTP 服务的二进制。`go get -u` 不会帮你
做——这正是新主版本号的含义。v3 停留在 `v3.0.0`。

### 2. ldflags 路径——这一条还是会静默失效

`-X` 的路径里同样带着模块路径，写错了不会报错：链接器找不到要写的目标，于是构建
成功、退出码 0，二进制报告 `dev`。除了 `.go` 文件，记得把 Makefile、Dockerfile
和发布流水线里的 `version-kit/v3` 一起 grep 出来。

```bash
# 升级后这样写是错的：构建正常、退出码 0、报告 "dev"
go build -ldflags "-X github.com/soulteary/version-kit/v3.Version=1.0.0" .

# 正确写法
go build -ldflags "-X github.com/soulteary/version-kit/v4.Version=1.0.0" .
```

### 3. net/http 函数移了位置

六个符号从根包搬到了 `httpadapter`，名字保持不变：

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

没有保留转发用的空壳函数：空壳自己就要 import `net/http`，而搬走的正是它。

### 4. 不用 net/http 的话，没有第 3 步

`HandlerConfig`、`ResolveConfig`、`DefaultHandlerConfig`、`Info`、`New`、
`Default`、`NewBuilder`，以及 `Payload` / `TextPayload` / `JSONResponse` /
`Headers` / `Normalized` 这些方法全都留在根包，原样不动。一个 CLI，或者按
[用其他框架提供服务](#用其他框架提供服务)那样自己写适配器的 Echo / Gin / chi
服务，做完第 1、2 步就结束了。

`fiberadapter` 的用户同理：它读的还是根包里那份配置，只有 import 路径要改。

### 5. 其余没有任何变化

行为、签名、响应字节都没有动。v4 改的只是模块路径和包边界，仅此而已。

## 从 v2 迁移

### 1. 模块路径

```bash
go get github.com/soulteary/version-kit/v4
```

然后改掉每一处 import：

```diff
-version "github.com/soulteary/version-kit/v2"
+version "github.com/soulteary/version-kit/v4"
```

这对**所有人**都适用，包括一行 Fiber 代码都没碰过的纯 net/http 服务。
`go get -u` 不会帮你做这件事——这正是新主版本的含义。v2 继续停在 `v2.2.0`。

### 2. ldflags 路径——这一条会静默失效

`-X` 是按完整导入路径指定包级变量的。指向一个已经不存在的包时，链接器不会吭声：

```bash
# 升级后这样写是错的：构建成功、退出码 0、版本是 "dev"
go build -ldflags "-X github.com/soulteary/version-kit/v2.Version=1.0.0" .

# 正确
go build -ldflags "-X github.com/soulteary/version-kit/v4.Version=1.0.0" .
```

没有报错，也没有警告。二进制照样构建、照样过 CI、照样上线，然后对外返回
`{"version":"dev"}`。发布打标签之前，先在构建脚本、Makefile、Dockerfile 和
CI 工作流里搜一遍旧路径。

### 3. Fiber 函数移了位置

```go
import (
    version "github.com/soulteary/version-kit/v4"
    "github.com/soulteary/version-kit/v4/fiberadapter"
)
```

| 原来 | 现在 |
|---|---|
| `version.FiberHandler(...)` | `fiberadapter.Handler(...)` |
| `version.FiberTextHandler(...)` | `fiberadapter.TextHandler(...)` |
| `version.FiberSimpleHandler()` | `fiberadapter.SimpleHandler()` |
| `version.FiberMiddleware(...)` | `fiberadapter.Middleware(...)` |
| `version.FiberMiddlewareWithConfig(...)` | `fiberadapter.MiddlewareWithConfig(...)` |
| `version.RegisterEndpointFiber(...)` | `fiberadapter.RegisterEndpoint(...)` |

### 4. 行为变化

**两个框架都受影响——`Info` 只在构建 handler 时读一次。** 响应体和版本响应头
都在构造期算好并在每个响应上复用，因此把 `Info` 交给 handler 之后再改它，
不会再影响实际服务的内容：

```go
info := version.New("1.0.0", "abc1234", "")
h := httpadapter.Handler(version.HandlerConfig{Info: info})
info.Version = "2.0.0"   // v2 这里会返回 2.0.0；v3 返回 1.0.0
```

请先把 `Info` 构造完整，再去构建 handler。`SimpleHandler` 是例外——它不接受
配置，仍然每次请求都读包变量。

**仅 Fiber——`Content-Type` 现在是 `application/json`**，与 net/http 一致。
Fiber 此前返回 `application/json; charset=utf-8`；RFC 8259 并没有为
`application/json` 定义 charset 参数。

**仅 Fiber——`Pretty` 开始生效。** `fiber.Ctx.JSON` 只写紧凑 JSON，所以此前配了
`Pretty: true` 的 Fiber 端点一直被静默地按紧凑返回，现在会缩进。

**仅 Fiber——响应体不再经过 app 配置的 `JSONEncoder`。** 改由 version-kit 自己
编码，这正是两个框架能做到逐字节一致的原因；自定义的 Fiber 编码器对这个端点
不再生效。

### 5. 需要 Go 1.27+

`go.mod` 声明的是 `go 1.27.0`，v3 在更低版本的工具链上构建不了。

## 更早的变更：构建详情改为 opt-in（v2.2.0）

下面这些是 v2.2.0 引入、在 v3 中依然成立的说明。如果你是从 v2.2.0 或更新的
版本升上来的，这部分你已经处理过了。

**端点的默认响应变小了。** 这正是本次修复，也是升级前唯一需要确认的一点。

- **构建详情默认不再暴露。** `Handler`、`fiberadapter.Handler`、`TextHandler` 以及版本头中间件
  此前会输出完整的 `Info`——Go 运行时版本、commit、构建时间、平台和编译器。这个端点通常
  没有认证，而 `Middleware` 会把同样的数据放在**每一个响应**上，于是 `go_version` 让任何
  人都能把一个已公开的 Go 运行时 CVE 对应到正在服务他们的那个具体构建，commit 和构建时间
  进一步收窄范围。现在默认响应是 `{"version":…,"branch":…}`。**如果你的工具链会从
  `/version` 解析 `commit`、`build_date` 或 `go_version`，请设置
  `IncludeBuildDetails: true`**，并把该端点放到认证之后或内部路由上。
- **新增 `MiddlewareWithConfig` 和 `fiberadapter.MiddlewareWithConfig`。**
  `Middleware(info, prefix)` 签名不变，现在只输出公开头；要把 `X-Commit` 和
  `X-Build-Date` 拿回来，请使用带 `IncludeBuildDetails` 的 `WithConfig` 形式。
- **新增 `Info.Public()`**，供自己拼响应的调用方使用。
- **`Full()` 会跳过未设置的字段。** 它此前无条件输出 `Go version:`、`Platform:` 和
  `Compiler:` 标签，于是精简后的 `Info` 渲染出来带着一串空行。`Commit`、`Branch` 和
  `BuildDate` 本来就是这样处理的。
- **非法的 `HeaderPrefix` 回退为 `"X-"`。** 它此前只做了空字符串检查就被拼进 header
  名称，于是含空格、冒号或换行的前缀会产出畸形 header。
- **运行要求里写的是 Go 1.26**；`go.mod` 需要 `1.27.0`。

## 许可证

Apache License 2.0
