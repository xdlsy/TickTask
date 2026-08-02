# docs/learnings/ — 经验库

经验库承载开发与运维中的踩坑教训，支持持续记录和检索。

## 文件结构

| 文件 | 用途 | 条目数 |
|------|------|--------|
| [LEARNINGS.md](LEARNINGS.md) | 学习记录：纠正、洞察、知识盲区、最佳实践 | 14 |
| [ERRORS.md](ERRORS.md) | 错误日志：命令失败和集成错误 | 2 |
| [FEATURE_REQUESTS.md](FEATURE_REQUESTS.md) | 功能请求：待实现的功能需求 | 3 |

## 经验索引

### 后端（Go）

| ID | 分类 | 摘要 |
|----|------|------|
| LRN-20260607-001 | correction | Go nil slice → JSON null → 前端 TypeError |
| LRN-20260607-004 | knowledge_gap | SQLite 单写者限制未压力测试 |
| LRN-20260607-007 | best_practice | Repository 构造函数返回接口类型 |
| LRN-20260607-009 | insight | GORM AutoMigrate 只加不删 |
| LRN-20260607-011 | best_practice | AI 响应需剥离 markdown 代码围栏 |
| LRN-20260607-012 | best_practice | backend/internal/ 编译器强制私有 |
| LRN-20260607-013 | insight | Handler 不是单例 |
| LRN-20260607-014 | insight | Settings 的 JSON 序列化存储模式 |

### 前端（Vue/TS）

| ID | 分类 | 摘要 |
|----|------|------|
| LRN-20260607-005 | best_practice | 类型必须添加到 src/types/index.ts |
| LRN-20260607-006 | best_practice | Pinia store 测试需 beforeEach 隔离 |
| LRN-20260607-008 | insight | WebSocket slow-client 静默断开 |

### 开发流程

| ID | 分类 | 摘要 |
|----|------|------|
| LRN-20260607-002 | best_practice | Go 代码修改后必须手动重启 |
| LRN-20260607-003 | best_practice | 后端重启后需刷新前端清空 stale 状态 |
| LRN-20260607-010 | best_practice | API key 明文存储，绝不提交 git |

## 使用方式

- **遇到错误** → 先查 `ERRORS.md`，再搜 `LEARNINGS.md`
- **开发新功能** → 检查相关 `LRN-*` 条目避免重复踩坑
- **发现新问题** → 使用 `aidoc-learning` skill 格式记录到此目录

## 格式规范

条目 ID 格式：`[LRN-YYYYMMDD-XXX]`
- 分类：correction | insight | knowledge_gap | best_practice
- AI 推断的条目标记 `<!-- HUMAN_REVIEW -->`，需人工确认
