# 领域能力目录

本目录存放可复用的领域能力（Skills）。每个 skill 是一个独立目录，包含：

```
skill-name/
├── SKILL.md          # 必需：skill 定义（YAML frontmatter + 工作流）
├── scripts/          # 可选：辅助脚本
├── assets/           # 可选：模板资产
└── learning.md       # 可选：历史经验记录
```

## 什么是 Skill？

Skill 是一段封装好的操作流程，AI 编码助手可按需加载执行。它不像 AGENTS.md 那样始终存在于上下文中，而是当用户请求匹配其描述时触发。

## 当前 Skills

| Skill 名称 | 描述 | 触发短语 | 状态 |
|-----------|------|---------|------|
| [auto-schedule](auto-schedule/SKILL.md) | 根据 todo.json + habit.md 自动生成整周日程 ICS 文件 | "安排日程"、"生成日程"、"排一下任务"、"帮我规划" | 已就绪 |
| [revise-schedule](revise-schedule/SKILL.md) | 在现有日程基础上进行针对性修订，而非从零生成 | "修订日程"、"调整日程"、"优化日程"、"重新安排" | 已就绪 |
| [openspec-apply-change](openspec-apply-change/SKILL.md) | 实现 OpenSpec change 中的任务 | "implement change"、"apply change" | 已就绪 |
| [openspec-archive-change](openspec-archive-change/SKILL.md) | 归档已完成的 OpenSpec change | "archive change" | 已就绪 |
| [openspec-explore](openspec-explore/SKILL.md) | 浏览 OpenSpec 规范结构 | "explore spec"、"show spec" | 已就绪 |
| [openspec-propose](openspec-propose/SKILL.md) | 提出新的 OpenSpec change | "propose change"、"new change" | 已就绪 |
| [openspec-sync-specs](openspec-sync-specs/SKILL.md) | 同步 OpenSpec 规范文件 | "sync specs" | 已就绪 |

## 按用途分类

### 日程管理（TickTask 核心领域）

- **auto-schedule** — 从零生成日程。输入 `config/todo.json` + `config/habit.md`，输出 `config/schedule.ics`。含校验脚本 `validate_schedule.py`。
- **revise-schedule** — 增量修订日程。输入现有 `config/schedule.ics` + 用户修订指令，输出修订后的 `config/schedule.ics`。复用 auto-schedule 的校验脚本。

### OpenSpec 工作流（通用变更管理）

- **openspec-propose** → **openspec-apply-change** → **openspec-archive-change**：完整的变更生命周期。
- **openspec-explore** 和 **openspec-sync-specs**：规范浏览和维护。

## 何时将能力封装为 Skill？

- 操作需要多个步骤（如"生成一周日程"需要读取配置、生成 ICS、校验、记录经验）
- 流程依赖特定脚本或配置文件（如 `validate_schedule.py`、`todo.json`、`habit.md`）
- 希望跨会话复用同一操作模式（如每次说"安排日程"都走相同流程）
- 操作有明确的输入/输出规范和校验机制

## 未封装为 Skill 的自动化操作

以下操作虽可自动化，但因流程简单或已充分记录在 AGENTS.md / CLAUDE.md 中，暂不封装：

| 操作 | 理由 |
|------|------|
| `make dev` / `make build` / `make test` | 标准开发命令，已在 CLAUDE.md 和 AGENTS.md 的 Build & Test Commands 中记录 |
| `scripts/build.sh` / `scripts/start.sh` | 被 Makefile 封装的基础设施脚本，非领域操作 |
| `npm run build` / `npm run dev` | 前端标准构建流程，无需额外封装 |

<!-- HUMAN_REVIEW: 如需将新的常见操作封装为 skill，请在此目录下创建新的 skill-name/SKILL.md -->
