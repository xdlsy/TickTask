# Windows 单文件 exe 打包设计

日期：2026-08-13
状态：已与用户确认设计，待实现

## 背景与问题

换到一台新 Windows 电脑后项目无法启动。根因：后端使用 `gorm.io/driver/sqlite`（底层
`mattn/go-sqlite3`），编译强制要求 CGO + gcc（MinGW），新机器没有 MinGW，`go run` 第一步即失败。

现状也没有可分发的打包产物：生产模式需要二进制 + `frontend/dist` 按相对路径摆放、在正确
的工作目录下启动（router.go 以 CWD 相对路径查找 dist），`configs/config.yaml` 不进 git。

## 目标

- 产出**单个 `ticktask.exe`**：双击即可运行，目标机器不需要安装 Go / Node / gcc。
- 前端静态资源嵌入二进制。
- 数据（SQLite + `.keyvault`）落 `%APPDATA%\TickTask\`。
- 构建不再依赖 CGO：任何装有 Go 的机器（含新电脑）都能构建与开发，支持交叉编译。

## 非目标

- 不做安装器（installer）、开机自启、系统托盘、隐藏控制台窗口。
- 不改动前端代码。
- 不做多平台产物发布流水线（Windows 优先；纯 Go 驱动使将来加 linux/mac 产物零成本）。

## 方案选型

选定方案 A（纯 Go SQLite + go:embed 前端），否决的备选：

- 方案 B（保留 mattn 驱动、在有 MinGW 的机器构建）：exe 可跑但构建链未改善，保留事故根因。
- 方案 C（zip 包带 exe + dist 文件夹）：非单文件，结构散了就跑不起来，也不解决开发环境。

## 架构变更

### 1. SQLite 驱动替换

`backend/pkg/database/db.go`：import 从 `gorm.io/driver/sqlite` 改为
`github.com/glebarez/sqlite`（基于 modernc.org/sqlite 的纯 Go GORM 驱动，API 兼容）。
`sqlite.Open(path)` 调用不变。构建改为 `CGO_ENABLED=0`。

### 2. 前端嵌入（新包 `backend/web/`）

- 新增 `backend/web/embed.go`：`//go:embed all:dist`，导出 `DistFS()`（`fs.FS`）与
  `IsStub()`（当前嵌入内容是否为占位页）。
- 仓库提交占位文件 `backend/web/dist/index.html`（"前端未构建"提示页），保证未构建前端时
  `go build` / `go test ./...` 不失败。
- `.gitignore` 忽略 `backend/web/dist/`，并用否定规则保留占位 `index.html`。

### 3. 静态资源服务（router.go 改造）

磁盘优先、嵌入兜底：

1. 依次探测磁盘 `dist`、`../frontend/dist`、`../../frontend/dist`（现状不变，覆盖仓库内开发）；
2. 否则若嵌入内容非占位，则用嵌入 FS 服务 `/assets` 与 SPA `NoRoute → index.html`；
3. 否则返回占位页（提示重新执行打包脚本）。

### 4. 路径解析（main.go）

- 配置查找顺序：CWD `configs/config.yaml`（现状，开发场景）→ `%APPDATA%\TickTask\config.yaml`
  → `config.LoadDefault()`。
- 数据目录：找到配置文件时沿用其 `database.path`（现状不变）；未找到时 =
  `os.UserConfigDir()` + `\TickTask\data\`，数据库 `ticktask.db` 与 `.keyvault` 均在此
  （两者配套加密存储 API key，必须成对迁移）。
  仓库开发布局（磁盘 dist 存在）下无配置时仍用 `./data/ticktask.db`，开发行为不变；
  仅打包运行（无磁盘 dist）落到 `<AppDir>/data`（2026-08-14 用户裁决）。

### 5. 打包模式自动开浏览器

启动监听成功后，若"未探测到磁盘 dist"（即打包运行场景），自动用系统默认浏览器打开
`http://localhost:<port>`。磁盘 dist 存在（开发场景）不打开。

## 构建与交付

- `scripts/build.sh` 新增 `exe` 命令：构建前端 → 拷贝 `frontend/dist` → `backend/web/dist`
  → `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/ticktask.exe`。
- `Makefile` 新增 `make exe`；现有 `build backend` 路径同步改为 `CGO_ENABLED=0`。
- 产物：`backend/bin/ticktask.exe`（约 40MB，单文件）。

## 存量数据迁移（一次性，纯文档）

把旧库 `ticktask.db` 与 `.keyvault` **成对**拷贝到 `%APPDATA%\TickTask\data\` 即完成迁移。
写入 README 打包使用小节，不写迁移代码。

## 错误处理

- `os.UserConfigDir()` 失败 → 沿用默认 `./data`；数据目录创建失败（MkdirAll 报错）→
  `log.Fatal` 直接终止，原因在控制台可见（2026-08-14 用户裁决：不实现回落，exe 旁目录
  往往同样不可写）。
- 嵌入为占位且磁盘无 dist → 服务占位页，文案指引用户重新打包。
- 端口占用等启动失败：维持现状 `log.Fatal`，控制台窗口可见原因。

## 测试策略

- 存量回归：service/handler 测试走 mock 不受影响；`pkg/database` 迁移测试将真实运行在
  纯 Go 驱动上，顺带验证驱动兼容性。
- 新增单测：配置/数据目录解析（搜索顺序、APPDATA 回落）；router 静态服务的
  嵌入兜底与占位页路径。
- 人工验收：干净目录运行 exe → 浏览器自动打开 → 创建任务、跑一个番茄 → 重启 exe → 数据
  仍在 `%APPDATA%\TickTask\data\`；换一台无 Go/Node/gcc 的机器重复一次。
- 文档同步：AGENTS.md 技术栈与构建说明去掉 CGO/MinGW 表述（实现合入后走资产刷新流程）。

## 验收标准

1. 新 Windows 电脑仅装 Go：`go test ./...` 绿、`go run` 可启动（无需 gcc）。
2. `make exe` 产出单文件 exe；在无任何工具链的机器上双击可用，数据落 APPDATA。
3. 仓库内开发行为不变：磁盘 dist 优先、CWD 配置优先。
4. 旧数据按文档成对拷贝后，exe 启动即可识别（含加密 API key）。
