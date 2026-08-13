# Windows 单文件 exe 打包 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 产出单个 `ticktask.exe`（前端嵌入、数据落 `%APPDATA%\TickTask\`），并让构建彻底摆脱 CGO/gcc 依赖。

**Architecture:** 三处代码改动 + 一个新包：① SQLite 驱动换成纯 Go 的 `github.com/glebarez/sqlite`（API 兼容，仅改 import）；② 新包 `backend/web` 用 `go:embed` 嵌入前端 dist（仓库内提交占位页）；③ router 静态服务改为"磁盘优先、嵌入兜底"；④ main.go 路径解析落 APPDATA + 打包模式自动开浏览器。构建脚本新增 `exe` 命令。

**Tech Stack:** Go 1.21 / Gin 1.10 / GORM 1.25 / glebarez/sqlite（纯 Go SQLite）/ go:embed / Vite 构建产物

**Spec:** `docs/superpowers/specs/2026-08-13-windows-single-exe-packaging-design.md`

## Global Constraints

- Go 1.21，module path 为 `ticktask`；新依赖只用 `github.com/glebarez/sqlite@v1.11.0`
- 所有 Go 构建命令一律 `CGO_ENABLED=0`（驱动替换后无任何 cgo import）
- 不改动 `frontend/src/` 下任何源码；前端只作为构建产物被拷贝嵌入
- 开发机为 Windows + Git Bash：`make` 不可用，验证脚本一律 `bash scripts/build.sh ...`
- 数据不变式：`ticktask.db` 与 `.keyvault` 必须位于同一目录（vault 加密 API key，成对迁移）
- 提交遵循 Conventional Commits（`feat:` / `docs:` / `chore:`）
- 仓库默认状态下 `backend/web/dist/` 只含占位 `index.html`（构建脚本用后恢复），保证 `go test ./...` 的 stub 语义确定

---

### Task 1: SQLite 驱动替换为纯 Go 实现

**Files:**
- Modify: `backend/go.mod` / `backend/go.sum`（go get 自动）
- Modify: `backend/pkg/database/db.go:8`
- Modify: 7 个测试文件的 import 行：
  - `backend/pkg/database/migrate_work_items_title_test.go:6`
  - `backend/internal/repository/work_log_repo_test.go:8`
  - `backend/internal/repository/setting_repo_test.go:12`
  - `backend/internal/repository/data_repo_test.go:7`
  - `backend/internal/repository/agent_repo_test.go:8`
  - `backend/internal/agent/service_test.go:17`
  - `backend/internal/service/work_log_service_quick_test.go:8`

**Interfaces:**
- Consumes: 现有 `gorm.io/gorm`；各测试中已有的 `sqlite.Open(...)` 调用（包名不变，调用点零改动）
- Produces: `database.Init(path string) (*gorm.DB, error)` 签名不变；全仓 CGO-free

- [ ] **Step 1: 添加依赖**

```bash
cd backend && GOPROXY=https://goproxy.cn,direct go get github.com/glebarez/sqlite@v1.11.0
```

Expected: go.mod 出现 `github.com/glebarez/sqlite v1.11.0`。若该版本因 go 指令版本不兼容报错，降级试 `@v1.10.0`。

- [ ] **Step 2: 替换 import（8 个文件，逐个把 `"gorm.io/driver/sqlite"` 改为 `"github.com/glebarez/sqlite"`）**

`backend/pkg/database/db.go` 的 import 块变为：

```go
import (
	"encoding/json"
	"ticktask/internal/model"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)
```

7 个测试文件同理：只把 `"gorm.io/driver/sqlite"` 这一行替换为 `"github.com/glebarez/sqlite"`，其余不动（两个驱动的包名都是 `sqlite`，`sqlite.Open(":memory:")` 等调用零改动）。

- [ ] **Step 3: tidy 并确认旧依赖移除**

```bash
cd backend && go mod tidy && grep -r "gorm.io/driver/sqlite" --include="*.go" . ; echo "exit=$?"
```

Expected: grep 无输出（exit=1），go.mod 中不再有 `gorm.io/driver/sqlite`。

- [ ] **Step 4: 全量测试（此时起不再需要 gcc）**

```bash
cd backend && CGO_ENABLED=0 go test ./...
```

Expected: 全部 PASS（repository/agent/database 测试真实跑在纯 Go 驱动上，即驱动兼容性验证）。

- [ ] **Step 5: Commit**

```bash
git add backend/go.mod backend/go.sum backend/pkg/database/db.go backend/pkg/database/migrate_work_items_title_test.go backend/internal/repository/work_log_repo_test.go backend/internal/repository/setting_repo_test.go backend/internal/repository/data_repo_test.go backend/internal/repository/agent_repo_test.go backend/internal/agent/service_test.go backend/internal/service/work_log_service_quick_test.go
git commit -m "feat: switch SQLite driver to pure-Go glebarez/sqlite (CGO-free builds)"
```

---

### Task 2: config.Resolve — 配置搜索与 APPDATA 数据目录解析

**Files:**
- Modify: `backend/pkg/config/config.go`
- Test: `backend/pkg/config/config_test.go`（新建）

**Interfaces:**
- Consumes: 现有 `Load(path)` / `LoadDefault()`
- Produces:
  - `func AppDir() (string, bool)` — 用户级 TickTask 根目录（Windows = `%APPDATA%\TickTask`），不可用时 ok=false
  - `func Resolve() (*Config, string)` — 生效配置 + 来源路径（默认值时为 `""`）。后续 Task 5 的 main.go 依赖这两个函数

- [ ] **Step 1: 写失败测试**

`backend/pkg/config/config_test.go`：

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

// 同时设置 Windows(APPDATA) 与 Linux(XDG_CONFIG_HOME) 的用户配置目录
func setUserConfigDir(t *testing.T, dir string) {
	t.Helper()
	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME"} {
		old, had := os.LookupEnv(k)
		os.Setenv(k, dir)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		})
	}
}

func unsetUserConfigDir(t *testing.T) {
	t.Helper()
	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME"} {
		old, had := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		})
	}
}

func writeConfig(t *testing.T, path, portYAML string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "server:\n  port: " + portYAML + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePrefersCwdConfig(t *testing.T) {
	cwd := t.TempDir()
	appdata := t.TempDir()
	writeConfig(t, filepath.Join(cwd, "configs", "config.yaml"), "9999")
	writeConfig(t, filepath.Join(appdata, "TickTask", "config.yaml"), "7777")
	chdir(t, cwd)
	setUserConfigDir(t, appdata)

	cfg, path := Resolve()
	if cfg.Server.Port != 9999 {
		t.Fatalf("port = %d, want 9999 (CWD config wins)", cfg.Server.Port)
	}
	if path != filepath.Join("configs", "config.yaml") {
		t.Fatalf("path = %q, want configs/config.yaml", path)
	}
}

func TestResolveFallsBackToAppDirConfig(t *testing.T) {
	cwd := t.TempDir()
	appdata := t.TempDir()
	writeConfig(t, filepath.Join(appdata, "TickTask", "config.yaml"), "7777")
	chdir(t, cwd)
	setUserConfigDir(t, appdata)

	cfg, path := Resolve()
	if cfg.Server.Port != 7777 {
		t.Fatalf("port = %d, want 7777 (APPDATA config)", cfg.Server.Port)
	}
	want := filepath.Join(appdata, "TickTask", "config.yaml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestResolveDefaultsPutDataUnderAppDir(t *testing.T) {
	cwd := t.TempDir()
	appdata := t.TempDir()
	chdir(t, cwd)
	setUserConfigDir(t, appdata)

	cfg, path := Resolve()
	if path != "" {
		t.Fatalf("path = %q, want empty (defaults)", path)
	}
	want := filepath.Join(appdata, "TickTask", "data", "ticktask.db")
	if cfg.Database.Path != want {
		t.Fatalf("db path = %q, want %q", cfg.Database.Path, want)
	}
}

func TestResolveDefaultsFallbackWhenNoUserDir(t *testing.T) {
	cwd := t.TempDir()
	chdir(t, cwd)
	unsetUserConfigDir(t)

	cfg, path := Resolve()
	if path != "" {
		t.Fatalf("path = %q, want empty", path)
	}
	if cfg.Database.Path != "./data/ticktask.db" {
		t.Fatalf("db path = %q, want ./data/ticktask.db", cfg.Database.Path)
	}
}

func TestAppDirJoinsTickTask(t *testing.T) {
	base := t.TempDir()
	setUserConfigDir(t, base)
	dir, ok := AppDir()
	if !ok {
		t.Fatal("ok = false, want true")
	}
	want := filepath.Join(base, "TickTask")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd backend && go test ./pkg/config/ -v
```

Expected: 编译失败，`undefined: Resolve` / `undefined: AppDir`。

- [ ] **Step 3: 最小实现**

在 `backend/pkg/config/config.go` 追加（import 增加 `"path/filepath"`）：

```go
// AppDir 返回用户级 TickTask 根目录（Windows 为 %APPDATA%\TickTask）。
// 操作系统用户配置目录不可用时 ok=false。
func AppDir() (string, bool) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return "", false
	}
	return filepath.Join(base, "TickTask"), true
}

// Resolve 加载生效配置：CWD 的 configs/config.yaml 优先（仓库开发布局），
// 其次 <AppDir>/config.yaml（打包 exe），都没有则用默认值，并把数据库
// 落到 <AppDir>/data/ticktask.db。第二个返回值为配置来源路径（默认值时为 ""）。
func Resolve() (*Config, string) {
	candidates := []string{filepath.Join("configs", "config.yaml")}
	if appDir, ok := AppDir(); ok {
		candidates = append(candidates, filepath.Join(appDir, "config.yaml"))
	}
	for _, p := range candidates {
		if cfg, err := Load(p); err == nil {
			return cfg, p
		}
	}
	cfg := LoadDefault()
	if appDir, ok := AppDir(); ok {
		cfg.Database.Path = filepath.Join(appDir, "data", "ticktask.db")
	}
	return cfg, ""
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
cd backend && go test ./pkg/config/ -v
```

Expected: 5 个测试全 PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/config/config.go backend/pkg/config/config_test.go
git commit -m "feat: add config.Resolve with APPDATA fallback for packaged exe"
```

---

### Task 3: backend/web — 前端嵌入包 + 占位页

**Files:**
- Create: `backend/web/embed.go`
- Create: `backend/web/dist/index.html`（占位页，进 git）
- Modify: `.gitignore`
- Test: `backend/web/embed_test.go`

**Interfaces:**
- Produces（Task 4 router 与 Task 5 main.go 依赖）:
  - `func DistFS() fs.FS` — 嵌入的前端文件系统（根即 dist 内容，含 `index.html`）
  - `func IsStub() bool` — 嵌入内容是否为占位页（marker: `ticktask-frontend-stub`）
  - `func FindDiskDist() string` — 工作目录下首个存在的磁盘 dist 路径（`dist` / `../frontend/dist` / `../../frontend/dist`），无则 `""`

- [ ] **Step 1: 写失败测试**

`backend/web/embed_test.go`：

```go
package web

import (
	"os"
	"testing"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

func TestDistFSContainsIndexHTML(t *testing.T) {
	f, err := DistFS().Open("index.html")
	if err != nil {
		t.Fatalf("open index.html: %v", err)
	}
	f.Close()
}

func TestIsStubTrueInRepoState(t *testing.T) {
	// 仓库默认只提交占位页；构建脚本打完 exe 会恢复占位页，此断言保持确定。
	if !IsStub() {
		t.Fatal("IsStub() = false, want true (repo embeds the placeholder)")
	}
}

func TestFindDiskDistEmptyOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if got := FindDiskDist(); got != "" {
		t.Fatalf("FindDiskDist() = %q, want empty", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd backend && go test ./web/ -v
```

Expected: 编译失败，`undefined: DistFS` 等。

- [ ] **Step 3: 创建占位页 + embed.go**

`backend/web/dist/index.html`：

```html
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="generator" content="ticktask-frontend-stub">
  <title>TickTask — 前端未构建</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 40rem; margin: 4rem auto; padding: 0 1rem; color: #333; background: #fafafa; }
    code { background: #eee; padding: .1rem .4rem; border-radius: 4px; }
  </style>
</head>
<body>
  <h1>TickTask 前端未构建</h1>
  <p>当前二进制只嵌入了占位页。请在仓库根目录运行 <code>bash scripts/build.sh exe</code> 重新打包，前端会被嵌入到 exe 中。</p>
</body>
</html>
```

`backend/web/embed.go`：

```go
// Package web 嵌入前端构建产物。仓库默认只提交占位页；打包脚本在编译前
// 把 frontend/dist 拷贝到本目录，编译后再恢复占位页。
package web

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
)

//go:embed all:dist
var distFS embed.FS

const stubMarker = "ticktask-frontend-stub"

// DistFS 返回嵌入的前端文件系统（内容即 dist/ 目录）。
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err) // dist 始终随二进制嵌入，不会失败
	}
	return sub
}

// IsStub 报告嵌入的前端是否为占位页（即构建前未拷入真实产物）。
func IsStub() bool {
	b, err := fs.ReadFile(distFS, "dist/index.html")
	if err != nil {
		return true
	}
	return bytes.Contains(b, []byte(stubMarker))
}

// FindDiskDist 返回工作目录下首个存在的前端磁盘 dist（仓库开发布局），
// 都不存在时返回 ""（打包 exe 场景）。
func FindDiskDist() string {
	for _, p := range []string{"dist", "../frontend/dist", "../../frontend/dist"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
```

`.gitignore` 在 `# ===== Build output =====` 小节追加（真实 dist 不入库，占位页保留）：

```
backend/web/dist/*
!backend/web/dist/index.html
```

- [ ] **Step 4: 跑测试确认通过**

```bash
cd backend && go test ./web/ -v
```

Expected: 3 个测试 PASS。再跑 `cd backend && go test ./...` 确认无回归。

- [ ] **Step 5: Commit**

```bash
git add backend/web/embed.go backend/web/dist/index.html backend/web/embed_test.go .gitignore
git commit -m "feat: add embedded frontend package with placeholder page"
```

---

### Task 4: router 静态服务 — 磁盘优先、嵌入兜底

**Files:**
- Modify: `backend/internal/api/router.go:147-173`（静态服务块整体替换）
- Test: `backend/internal/api/static_test.go`（新建）

**Interfaces:**
- Consumes: Task 3 的 `web.FindDiskDist() string` / `web.IsStub() bool` / `web.DistFS() fs.FS`
- Produces: 包内函数 `serveFrontend(r *gin.Engine)`（SetupRouter 末尾调用；main.go 不直接用）

- [ ] **Step 1: 写失败测试**

`backend/internal/api/static_test.go`：

```go
package api

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

// 仓库默认嵌入占位页：无磁盘 dist 时兜底返回占位页，API 未匹配返回 404 JSON。
// 注意：依赖"仓库默认只含占位页"这一不变式（构建脚本打完 exe 会恢复占位页）。
func TestServeFrontendFallbackServesStub(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	chdir(t, dir)

	r := gin.New()
	serveFrontend(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatalf("GET / code = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ticktask-frontend-stub") {
		t.Fatalf("GET / body missing stub marker, got: %.200s", w.Body.String())
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/nope", nil))
	if w2.Code != 404 {
		t.Fatalf("GET /api/nope code = %d, want 404", w2.Code)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd backend && go test ./internal/api/ -v -run TestServeFrontend
```

Expected: 编译失败，`undefined: serveFrontend`。

- [ ] **Step 3: 实现**

`router.go` 中替换原 147-173 行静态服务块为：

```go
	// 静态文件服务：磁盘 dist 优先（仓库开发布局），嵌入式前端兜底（打包 exe）
	serveFrontend(r)
```

同文件新增函数与 import（import 增加 `"io/fs"`、`"strings"`、`"ticktask/web"`；`"os"` 若不再被其他代码引用则移除，`"path/filepath"` 保留）：

```go
// serveFrontend 注册前端静态资源服务。
// 顺序：磁盘 dist（dist / ../frontend/dist / ../../frontend/dist）→
// 嵌入的真实前端 → 嵌入占位页。API 前缀的未匹配路由一律 404 JSON。
func serveFrontend(r *gin.Engine) {
	if diskPath := web.FindDiskDist(); diskPath != "" {
		r.Static("/assets", filepath.Join(diskPath, "assets"))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.File(filepath.Join(diskPath, "index.html"))
		})
		return
	}

	dist := web.DistFS()
	if !web.IsStub() {
		if assets, err := fs.Sub(dist, "assets"); err == nil {
			r.StaticFS("/assets", http.FS(assets))
		}
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.FileFromFS("index.html", http.FS(dist))
	})
}
```

- [ ] **Step 4: 跑测试确认通过 + 全量回归**

```bash
cd backend && go test ./internal/api/ -v -run TestServeFrontend && go test ./...
```

Expected: 目标测试 PASS（占位页含 marker、/api/nope 404），全量 PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/router.go backend/internal/api/static_test.go
git commit -m "feat: serve embedded frontend as fallback in router"
```

---

### Task 5: main.go — Resolve 接线、真实建目录、监听与自动开浏览器

**Files:**
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: Task 2 的 `config.Resolve() (*Config, string)`；Task 3 的 `web.FindDiskDist() string` / `web.IsStub() bool`
- Produces: 无（进程入口；行为由 Task 8 人工验收）

- [ ] **Step 1: 替换配置加载（main.go:22-27）**

```go
	// 加载配置：CWD configs/config.yaml → <APPDATA>/TickTask/config.yaml → 默认值
	// 注意：cfg 直接使用 Resolve 的返回值 —— 其默认值分支已把 database.path
	// 指向 <APPDATA>/TickTask/data，不要再用 LoadDefault() 覆盖（会退回 ./data）。
	cfg, cfgPath := config.Resolve()
	if cfgPath == "" {
		logger.Logger.Warn("using default config")
	} else {
		logger.Logger.Info("loaded config", "path", cfgPath)
	}
```

- [ ] **Step 2: ensureDataDir 真实创建目录（main.go:162-165）**

```go
func ensureDataDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
```

（打包模式首启时 APPDATA 下目录不存在，必须先建。）

- [ ] **Step 3: 监听改造 + 打包模式自动开浏览器（main.go:154-159 替换）**

import 增加 `"net"`、`"net/http"`、`"os/exec"`、`"runtime"`、`"ticktask/web"`：

```go
	// 先绑定端口再服务，确保自动开浏览器时页面已可访问
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	logger.Logger.Info("server listening", "addr", addr)

	// 打包运行（无磁盘 dist 且嵌入了真实前端）时自动打开默认浏览器
	if web.FindDiskDist() == "" && !web.IsStub() {
		go openBrowser(fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
	}

	if err := (&http.Server{Handler: router}).Serve(ln); err != nil {
		log.Fatal(err)
	}
```

文件末尾追加：

```go
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		logger.Logger.Warn("failed to open browser", "err", err)
	}
}
```

- [ ] **Step 4: 编译 + 全量测试**

```bash
cd backend && go vet ./... && CGO_ENABLED=0 go build ./... && go test ./...
```

Expected: vet/build/test 全绿。

- [ ] **Step 5: 仓库开发模式冒烟（确认行为不变）**

```bash
cd backend && go run ./cmd/server &
sleep 3 && curl -s http://localhost:8080/api/tasks | head -c 200; kill %1
```

Expected: API 正常返回 JSON（前端由 vite 提供，不依赖本改动）。

- [ ] **Step 6: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: packaged-mode data dir resolution and auto-open browser"
```

---

### Task 6: 构建脚本 — exe 命令 + CGO_ENABLED=0

**Files:**
- Modify: `scripts/build.sh`
- Modify: `Makefile`

**Interfaces:**
- Consumes: Task 3 的占位页恢复约定（`git checkout` 恢复 tracked index.html）
- Produces: `bash scripts/build.sh exe` → `backend/bin/ticktask.exe`；`make exe` 等价入口

- [ ] **Step 1: build_backend 改 CGO-free（scripts/build.sh:40）**

```bash
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/ticktask-server ./cmd/server
```

- [ ] **Step 2: 新增 build_exe 函数与命令分支**

在 `scripts/build.sh` 的 `build_frontend()` 之后追加：

```bash
# 构建单文件 exe：前端产物嵌入二进制，目标机器无需 Go/Node/gcc
build_exe() {
    echo -e "${YELLOW}📦 构建单文件 exe...${NC}"

    # 1) 构建前端
    "$SCRIPT_DIR/build.sh" frontend

    # 2) 拷贝前端产物到嵌入目录（覆盖占位页）
    echo "拷贝前端产物到 backend/web/dist ..."
    rm -rf "$PROJECT_DIR/backend/web/dist"
    cp -r "$PROJECT_DIR/frontend/dist" "$PROJECT_DIR/backend/web/dist"

    # 3) 纯 Go 构建（无 CGO）
    cd "$PROJECT_DIR/backend"
    export GOPROXY=https://goproxy.cn,direct
    go mod download
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/ticktask.exe ./cmd/server
    cd "$PROJECT_DIR"

    # 4) 恢复占位页：保证 go test 的 stub 语义与 git 状态干净
    rm -rf "$PROJECT_DIR/backend/web/dist"
    git -C "$PROJECT_DIR" checkout -- backend/web/dist

    echo -e "${GREEN}✅ 单文件 exe 构建完成: backend/bin/ticktask.exe${NC}"
}
```

`main()` 的 case 增加：

```bash
        exe)
            build_exe
            ;;
```

用法提示（`用法: $0 [backend|frontend|all|exe]`）同步更新。

- [ ] **Step 3: Makefile 增加 exe 目标**

`.PHONY` 行加 `exe`；新增：

```make
# 构建单文件 Windows exe（前端嵌入，无 CGO）
exe:
	@./scripts/build.sh exe
```

- [ ] **Step 4: 验证脚本链路**

```bash
bash scripts/build.sh exe && ls -lh backend/bin/ticktask.exe && git status --short backend/web/
```

Expected: exe 存在（约 30-50MB）；`backend/web/` 无未提交变更（占位页已恢复）。

```bash
cd backend && go test ./web/ -v
```

Expected: `TestIsStubTrueInRepoState` 仍 PASS（恢复占位页生效）。

- [ ] **Step 5: Commit**

```bash
git add scripts/build.sh Makefile
git commit -m "feat: add single-file exe build (frontend embedded, CGO-free)"
```

---

### Task 7: 文档同步

**Files:**
- Modify: `AGENTS.md`（Build & Test Commands、Repository Structure、领域能力/打包说明）
- Modify: `backend/pkg/database/AGENTS.md:21`（依赖表述）

**Interfaces:**
- Consumes: Task 1/6 的实际命令与产物路径
- Produces: 与代码一致的构建文档

- [ ] **Step 1: AGENTS.md 构建命令更新**

- `cd backend && CGO_ENABLED=1 go build -o bin/ticktask-server cmd/server/main.go  # Build binary` 替换为：
  `cd backend && CGO_ENABLED=0 go build -o bin/ticktask-server ./cmd/server  # Build binary (pure-Go SQLite, no gcc needed)`
- 命令清单追加一行：
  `bash scripts/build.sh exe       # Build single-file ticktask.exe (frontend embedded)`
- Repository Structure 的 `backend/` 树中 `pkg/` 行后追加：
  `│   ├── web/                  # Embedded frontend (go:embed dist + placeholder page)`

- [ ] **Step 2: AGENTS.md 追加"打包分发"小节（放在 Testing Guidelines 之后）**

```markdown
## 打包分发（Windows 单文件 exe）

- 构建：`bash scripts/build.sh exe`（或 `make exe`）→ `backend/bin/ticktask.exe`，前端已嵌入，目标机器零依赖。
- 数据位置：`%APPDATA%\TickTask\data\`（`ticktask.db` + `.keyvault` 必须成对迁移）。
- 可选配置：`%APPDATA%\TickTask\config.yaml`（缺省用默认值；AI 设置存在数据库里）。
- 存量数据迁移：把旧 `backend/data/ticktask.db` 与 `.keyvault` 成对拷到上述目录即可。
- 打包模式启动会自动打开浏览器；开发模式（仓库内有磁盘 dist）不会。
```

- [ ] **Step 3: backend/pkg/database/AGENTS.md 依赖行更新**

`- Depends on: \`internal/model\` (all 5 GORM entities), \`gorm.io/gorm\`, \`gorm.io/driver/sqlite\`` 中的驱动改为 `github.com/glebarez/sqlite`（纯 Go，CGO-free）。

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md backend/pkg/database/AGENTS.md
git commit -m "docs: update build and packaging docs for CGO-free single exe"
```

---

### Task 8: 端到端人工验收

**Files:** 无代码改动（验证清单）

**Interfaces:**
- Consumes: Task 1-7 全部产物
- Produces: 验收结论（对照 spec 验收标准）

- [ ] **Step 1: 全量自动化检查**

```bash
cd backend && CGO_ENABLED=0 go test ./... && go vet ./...
cd ../frontend && npm run build
```

Expected: 全绿（spec 验收 #1：新机器仅装 Go 即可通过，无需 gcc）。

- [ ] **Step 2: 模拟干净机器运行 exe**

```bash
bash scripts/build.sh exe
mkdir -p /tmp/tt-clean /tmp/tt-appdata && rm -rf /tmp/tt-clean/* /tmp/tt-appdata/*
cp backend/bin/ticktask.exe /tmp/tt-clean/
cd /tmp/tt-clean && APPDATA=/tmp/tt-appdata ./ticktask.exe &
sleep 3 && curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/
```

注意：用 `APPDATA=/tmp/tt-appdata` 覆盖环境变量来模拟干净用户目录，**绝不删真实 `%APPDATA%\TickTask`**。

Expected: 浏览器自动打开；HTTP 200；`/tmp/tt-appdata/TickTask/data/ticktask.db` 与 `.keyvault` 已生成。

- [ ] **Step 3: 数据持久化 + 迁移验证**

在 UI 中创建一个任务 → `kill %1` → 重新 `APPDATA=/tmp/tt-appdata ./ticktask.exe &` → 任务仍在（spec 验收 #2/#4）。
迁移路径：把仓库旧 `backend/data/ticktask.db` + `.keyvault` 成对拷入 `/tmp/tt-appdata/TickTask/data/`，重启 exe（带同样的 APPDATA 覆盖），确认历史任务与 Settings 页 API key 可用。真实迁移到本机 `%APPDATA%\TickTask\data\` 由用户自行决定时机。

- [ ] **Step 4: 开发模式回归**

```bash
bash scripts/start.sh dev
```

Expected: 后端 `go run` 无 gcc 也能起（本机 MinGW 可保留但不再必需）；前端 :5173 正常，行为与改造前一致（spec 验收 #3：磁盘 dist 优先、CWD 配置优先）。

- [ ] **Step 5: 清理**

确认 `backend/server.exe`（历史遗留的游离产物）可删除后删除；`git status` 干净。

---

## Self-Review 记录

- **Spec 覆盖**：驱动替换(T1)、嵌入(T3/T4)、路径解析(T2/T5)、自动开浏览器(T5)、构建交付(T6)、迁移文档(T7)、错误处理(T5 Step2 建目录 + 占位页文案 T3)、测试(T2/T3/T4 单测 + T8 人工)、文档同步(T7) —— 全部对应。
- **占位符扫描**：无 TBD/TODO；所有代码步骤含完整代码。
- **类型一致性**：`Resolve() (*Config, string)`、`AppDir() (string, bool)`、`DistFS() fs.FS`、`IsStub() bool`、`FindDiskDist() string`、`serveFrontend(r *gin.Engine)` 在各任务间引用一致。
