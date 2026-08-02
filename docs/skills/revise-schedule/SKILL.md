---
name: revise-schedule
description: 修订已有日程的 skill。当用户用中文表达"修订日程"、"调整日程"、"优化日程"、"修改日程"、"重新安排"、"更新日程"等需求时触发。与 auto-schedule 不同，此 skill 在现有日程基础上进行针对性修订，而非从零生成。
---

# 日程修订器

## ⛔ 核心规则：使用子 Agent

**必须通过 `Agent` 工具派发子 Agent 修订 ICS，禁止在主对话中直接修改。** 原因：多文件读写 + 校验 + 可能重试会导致上下文爆炸。

Agent prompt 模板：
> 读取以下文件：config/schedule.ics（当前日程）、config/todo.json（任务清单）、config/habit.md（作息习惯）、docs/skills/revise-schedule/learning.md（历史经验）。为 [日期范围] 修订现有日程，修订指令：[用户的具体修订要求]。修订时遵守：
> 1. 每个任务的 preferred_start_time~preferred_end_time 是硬约束
> 2. 已有日程的时间安排尽量保留，仅对需要调整的部分进行修改
> 3. 固定时间任务不移动
> 4. 午餐时段不占用
> 生成修订后的 ICS 写入 config/schedule.ics。完成后运行 python3 docs/skills/auto-schedule/scripts/validate_schedule.py --tasks config/todo.json --ics config/schedule.ics 校验，不匹配则修正重试（最多2次）。

## ⛔ 安全约束

| 允许 | 禁止 |
|------|------|
| ✅ Read 任意文件 | ❌ 修改源码（.go/.vue/.ts/.py/.js） |
| ✅ Write `config/schedule.ics` | ❌ 创建/删除文件（除 schedule.ics 外） |
| ✅ Edit `learning.md`（去重后） | ❌ 编辑 config/todo.json、config/habit.md |

## 工作流程

### 1. 加载上下文
- 读取 `config/schedule.ics` — **当前已有日程，这是修订的基线**
- 读取 `config/todo.json` — 关注 `preferred_start_time`/`preferred_end_time`
- 读取 `config/habit.md` — 工作时间、午餐时段
- 读取 `docs/skills/revise-schedule/learning.md` — 历史错误，避免重犯
- 理解用户的修订指令（如"把XX任务移到下午"、"优化上午的安排"、"为紧急任务腾出时间"等）

### 2. 修订策略

修订优先级（严格遵守）：
1. **保留现有安排** — 未被用户指令涉及的任务保持原有时间不变
2. **偏好时段硬约束** — `preferred_start_time`~`preferred_end_time` 内安排，这是最高优先级
3. 固定时间任务不动 → 仅在用户明确要求时才调整
4. 精力匹配（deep_work=上午，shallow_work=下午）
5. 优先级排序（high > medium > low）
6. 午餐保护（不占用 lunch_start~lunch_end）
7. daily 任务每天一个实例，weekly 任务仅在对应星期几
8. 只输出任务，不输出午餐/休息

**修订模式：**
- 如果用户要求"优化"：检查偏好时段匹配、精力匹配、时间利用率，做最小调整
- 如果用户要求"重新安排XX"：仅移动指定任务，其他不变
- 如果用户要求"腾出时间给XX"：找到可压缩/可移动的任务，为新任务让路
- 如果用户没有明确指令：检查当前日程是否有冲突或不合理之处，提出优化建议并执行

**整周7天缺一不可。** 忽略 todo.json 中的 date 字段和系统日期。

### 3. 校验（不可跳过）

```bash
python3 docs/skills/auto-schedule/scripts/validate_schedule.py --tasks config/todo.json --ics config/schedule.ics
```
- 退出码 0 → 通过，继续下一步
- 退出码 1 → 修正后重试（最多2次），仍失败则记录 learning.md（如果已经有重复经验，则添加至当前 SKILL.md 中作为硬约束）
- 退出码 2 → 检查文件路径

### 4. 记录经验

如有不匹配：**先读 learning.md，查重后追加**。相同任务名+时段组合不重复记录。格式：`| 日期 | 任务名 | 偏好时段 | 实际时段 | 原因 |`

## 与 auto-schedule 的区别

| 维度 | auto-schedule | revise-schedule |
|------|--------------|-----------------|
| 输入 | 仅任务清单 | 任务清单 + 已有日程 |
| 策略 | 从零排程 | 在现有基础上修订 |
| 变动范围 | 全部重新安排 | 最小化变动，仅改必要部分 |
| 触发词 | 生成日程、安排日程 | 修订日程、调整日程、优化日程 |

## ICS 格式

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//TickTask//EN
CALSCALE:GREGORIAN
METHOD:PUBLISH
BEGIN:VEVENT
DTSTART:20260601T091500
DTEND:20260601T093500
SUMMARY:早例会
DESCRIPTION:早例会 | high优先级 | shallow_work
END:VEVENT
END:VCALENDAR
```
- 日期时间：`YYYYMMDDTHHMMSS`（本地时间）
- 不含 VTIMEZONE
- 不含午餐/休息事件
