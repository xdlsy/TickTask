# 学习记录

开发过程中捕获的纠正、洞察和知识盲区。

**分类**：correction | insight | knowledge_gap | best_practice

---

### [LRN-20260607-001] Go nil slice 序列化为 JSON null 导致前端崩溃

- **分类**: correction
- **现象**: 用户点击"确定"应用日程修订时，前端崩溃并显示通用错误"应用修订失败，请重试"。
- **根因**: Go 中 `var x []T` 声明的 nil slice 经 `json.Marshal` 后变成 `null`，而前端 TypeScript 对 `null` 调用 `.map()` 抛出 TypeError，绕过了正常的错误处理流程。<!-- HUMAN_REVIEW -->
- **教训**: 在 Go 中所有将序列化为 JSON 的 slice 必须用 `make([]T, 0)` 初始化，而不是 `var x []T`。前端也应在 `.map()` 前加 `?? []` 空值守护。
- **参考**: commit `9eea638` — `schedule_service.go` 中 `GenerateSchedule` 和 `ApplyRevision` 两处修复。
- **适用范围**: 所有 Go 后端返回 JSON 数组的 API 端点。

<!-- HUMAN_REVIEW -->

---

### [LRN-20260607-002] 后端修改后必须手动重启才能生效

- **分类**: best_practice
- **现象**: 修改 Go 后端代码后，浏览器行为不变，误以为代码有误。
- **根因**: Go 是编译型语言，`go run` 不会监听文件变更自动重新编译。
- **教训**: Go 代码每次修改后必须重启后端进程。推荐工作流：`lsof -ti:8080 | xargs kill -9 && cd backend && go run cmd/server/main.go`。前端（Vite）则支持 HMR 热更新，修改组件/store/样式自动生效。

---

### [LRN-20260607-003] 后端重启后 WebSocket 连接断开需刷新前端状态

- **分类**: best_practice
- **现象**: 后端重启后，前端定时器状态异常（显示旧数据或操作无响应）。
- **根因**: 后端重启会断开 WebSocket 连接。虽然前端 `wsClient` 会自动重连，但重连前发送的本地状态可能与服务端不一致。<!-- HUMAN_REVIEW -->
- **教训**: 重启后端前应先刷新前端页面以清空 stale 状态。端到端测试 AI 功能的标准流程：杀旧进程 → 启动新后端 → 刷新前端页面。

---

### [LRN-20260607-004] SQLite 单写者限制在定时器场景下未经验证

- **分类**: knowledge_gap
- **现象**: 定时器运行期间，用户同时操作任务（CRUD）可能导致写入冲突。
- **根因**: SQLite 是单写者数据库，并发写入会触发 `SQLITE_BUSY` 错误。GORM 默认无重试逻辑。<!-- HUMAN_REVIEW -->
- **教训**: 当前未对定时器活跃期间的并发写入进行压力测试。若出现写入冲突，需考虑 WAL 模式、写入队列或连接池配置。
- **待验证**: 定时器 1 秒 ticker 更新 `PomodoroSession` 与用户同时编辑任务的并发安全性。

<!-- HUMAN_REVIEW -->

---

### [LRN-20260607-005] 前端类型必须添加到单一 barrel file src/types/index.ts

- **分类**: best_practice
- **现象**: 在组件旁新建 types 文件后，类型在其他组件中不可用或产生循环依赖。
- **根因**: 项目约定所有共享类型集中在 `src/types/index.ts`，组件旁的类型文件不在导入路径中。
- **教训**: 新增前端共享类型时，必须添加到 `src/types/index.ts`，不要在组件目录下创建独立类型文件。

---

### [LRN-20260607-006] Pinia store 测试必须在 beforeEach 中隔离

- **分类**: best_practice
- **现象**: Pinia store 测试之间状态泄漏，后一个测试用例看到前一个用例的残留数据。
- **根因**: Pinia store 是全局单例，不重置会跨测试共享状态。
- **教训**: 每个 `beforeEach` 中必须调用 `setActivePinia(createPinia())` 创建全新 Pinia 实例。

---

### [LRN-20260607-007] Repository 构造函数返回接口类型，不返回具体结构体

- **分类**: best_practice
- **现象**: 修改 repository 实现细节时，调用方代码编译失败。
- **根因**: 构造函数若返回具体类型，调用方会依赖实现细节而非接口契约。
- **教训**: Repository 构造函数必须返回接口类型：`NewTaskRepository(db *gorm.DB) TaskRepository`，而非 `*taskRepository`。所有消费方程序针对接口编程。Mock 测试中使用 `map[string]*Model` 内存存储实现接口。

---

### [LRN-20260607-008] WebSocket slow-client 会静默断开

- **分类**: insight
- **现象**: 前端 WebSocket 连接在长时间无交互后突然断开，无错误提示。
- **根因**: WebSocket Hub 对每个 Client 使用 buffered `send chan []byte`（容量 256）。非阻塞发送模式下，如果客户端消费速度跟不上，会触发 `select { case client.send <- msg: default: unregister }` 静默踢出。<!-- HUMAN_REVIEW -->
- **教训**: 前端 WebSocket 客户端必须及时消费消息。如果出现频繁断连，检查前端是否有阻塞主线程的操作。

<!-- HUMAN_REVIEW -->

---

### [LRN-20260607-009] GORM AutoMigrate 是非破坏性的——只加不删

- **分类**: insight
- **现象**: 删除 model 中的字段后，数据库表中对应列仍然存在。
- **根因**: GORM `AutoMigrate` 只添加新列和新表，从不删除或重命名已有列。
- **教训**: 需要删除列时必须手动执行 SQL migration。`AutoMigrate` 适合开发早期快速迭代，生产环境应有明确的 migration 策略。

---

### [LRN-20260607-010] AI API key 明文存储在 config.yaml，绝不能提交到 git

- **分类**: best_practice
- **现象**: API key 泄露风险。
- **根因**: `backend/configs/config.yaml` 以明文存储 AI API key。
- **教训**: 确保 `config.yaml` 在 `.gitignore` 中。生产环境优先使用环境变量 `TT_AI_API_KEY` 覆盖文件配置。Settings API 返回时需遮蔽 key（仅显示前4+后4字符）。

---

### [LRN-20260607-011] AI 响应解析需处理 markdown 代码围栏

- **分类**: best_practice
- **现象**: AI 功能调用返回 JSON 解析失败。
- **根因**: LLM 有时在 JSON 外包裹 markdown 代码围栏（\`\`\`json ... \`\`\`），导致 `json.Unmarshal` 失败。
- **教训**: 所有 AI 响应解析必须使用 `extractJSON()` 辅助函数剥离 markdown 代码围栏。Prompt 中已明确要求"只返回 JSON，不要包含任何其他文字"，但不可完全依赖 prompt 遵从度。

<!-- HUMAN_REVIEW -->

---

### [LRN-20260607-012] backend/internal/ 是编译器强制的私有包

- **分类**: best_practice
- **现象**: 尝试从 `backend/` 外部导入 `internal/` 包时编译失败。
- **根因**: Go 的 `internal` 包规则由编译器强制执行——`backend/internal/` 下的包只能被 `backend/` 内的代码导入。
- **教训**: 这是 Go 的语言级保护，不需要额外约定。但 `pkg/database` 导入 `internal/model` 是唯一允许的例外（在 DI 层级）。新包不要尝试绕过此限制。

---

### [LRN-20260607-013] Handler 不是单例——每个路由注册时新建实例

- **分类**: insight
- **现象**: 在 handler 中使用包级变量或假设单例时，状态不一致。
- **根因**: Handler 在 `router.go` 中每次路由注册时内联创建 `New*Handler(svc)`，不是单例。
- **教训**: 不要在 handler 中依赖单例模式或包级状态。所有状态通过注入的 service 获取。

<!-- HUMAN_REVIEW -->

---

### [LRN-20260607-014] Settings 存储 JSON 序列化——类型 API 背后是字符串存储

- **分类**: insight
- **现象**: 直接查看 Setting 表看到的是 JSON 字符串，而非结构化数据。
- **根因**: Settings 使用 key-value 存储，复杂配置（如 PomodoroSettings）在 repository 层做 JSON marshal/unmarshal 转换。
- **教训**: 新增设置项时需同时定义 typed struct（如 `PomodoroSettings`）和对应的 JSON 序列化逻辑。不要直接操作 Setting 表的字符串值。

<!-- HUMAN_REVIEW -->

---

### [LRN-20260802-015] time.Now().UnixNano() 作 ID 生成器会在高频调用下产生重复 ID

- **分类**: correction
- **现象**: `SaveWorkLog` 保存 2+ 个 items 时偶发主键冲突；单 item 总能成功。
- **根因**: `idGenerator: func() string { return fmt.Sprintf("id-%d", time.Now().UnixNano()) }` 在 `buildWorkLogFromInput` 内被快速连续调用 1+N 次（log ID + 每个 item ID）。现代 CPU 上纳秒级分辨率不足以区分这些相邻调用，多次调用返回相同值。单元测试用 stub `func() string { return "test-id" }` 一直返回相同值但 mock repo 不在乎唯一性，所以测试全 PASS 掩盖了问题。
- **教训**: 任何在循环或快速路径中调用的 ID 生成器必须用 UUID（`github.com/google/uuid`，已是依赖）。`time.Now().UnixNano()` 仅适合单次调用或低频场景。E2E 测试（含多 item POST）才能暴露此类 bug，单元测试不够。
- **参考**: commit `dff255d` — `work_log_service.go` 改用 `uuid.New().String()`。
- **适用范围**: 所有 ID 生成场景。

---

### [LRN-20260802-016] vue-tsc 阻塞 `npm run build`，但 `vite build` 可独立运行

- **分类**: insight
- **现象**: 修改前端代码后 `npm run build` 失败，但开发服务器（`npm run dev`）能正常运行。
- **根因**: `package.json` 的 `build` 脚本是 `vue-tsc && vite build`——先做类型检查再打包。任何类型错误（即使与本次改动无关的预存错误）都会阻塞整个 build。
- **教训**: 调试或验证打包时，可用 `npx vite build` 单独跑打包步骤，跳过 vue-tsc。但要意识到这不等于类型安全，生产 build 仍需 vue-tsc 通过。预存类型错误（如 `PomodoroSettings.scheduling_strategy`、`AISettings.cli_tool` 这类后端不存在的死字段）应及时清理，避免阻塞后续开发。
- **参考**: commit `96a8168` — 清理 3 个预存 TS 错误。
- **适用范围**: 整个前端项目。

---

### [LRN-20260802-017] 本机 git 全局身份未配置——commit 需要 inline `-c` override

- **分类**: best_practice
- **现象**: 在本仓库执行 `git commit` 时报错 `fatal: unable to auto-detect email address` / `Author identity unknown`，导致 commit 失败。
- **根因**: 本机（Windows VM `Administrator@P_V30315-GSAtW8`）的 `git config --global` 未设置 `user.name` / `user.email`，仓库本身也没 local config。CLAUDE.md 安全规则禁止 `git config --global` 写入。
- **教训**: 用 inline 一次性 override：`git -c user.name="lsy" -c user.email="lsy@local" commit -m "..."`。这不修改任何 config 文件，符合安全规则。最近的 commit 都是 `lsy <lsy@local>`，沿用此身份即可。
- **适用范围**: 本机所有 git commit 操作（直到用户手动配置 global identity）。
