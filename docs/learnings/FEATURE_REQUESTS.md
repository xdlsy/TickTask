# 功能请求

用户请求的功能。

---

### [FEAT-20260607-001] SQLite 并发写入压力测试

- **来源**: knowledge_gap — LRN-20260607-004
- **描述**: 定时器 1 秒 ticker 更新与用户 CRUD 操作并发时，SQLite 单写者限制可能引发 `SQLITE_BUSY` 错误。需要压力测试验证。
- **建议方案**: 启用 WAL 模式、添加写入重试逻辑、或引入写入队列。
- **优先级**: 中（当前未见实际报错，但属于潜在风险）

<!-- HUMAN_REVIEW -->

---

### [FEAT-20260607-002] 配置 CI/CD 流水线

- **来源**: AGENTS.md — 未检测到 CI 配置
- **描述**: 项目无 `.github/workflows/` 或其他 CI 配置。所有测试和构建均为手动执行。
- **建议方案**: 添加 GitHub Actions 工作流：Go test + Frontend type check + build。
- **优先级**: 低（个人项目，当前手动流程可接受）

<!-- HUMAN_REVIEW -->

---

### [FEAT-20260607-003] 生产环境数据库 Migration 策略

- **来源**: insight — LRN-20260607-009
- **描述**: 当前使用 GORM AutoMigrate（只加不删）。生产环境需要支持列删除/重命名的 migration 方案。
- **建议方案**: 引入 golang-migrate 或类似工具管理版本化 SQL migration。
- **优先级**: 低（当前处于开发阶段，schema 变更频繁）
