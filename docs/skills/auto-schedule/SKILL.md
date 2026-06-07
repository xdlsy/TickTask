---
name: auto-schedule
description: 自动生成日程的 skill。当用户用中文表达"安排日程"、"生成日程"、"排一下任务"、"帮我规划今天/明天/这周"、"自动排日程"、"导出日历"等需求时触发。即使用户没有明确说"生成日程"，只要涉及根据任务清单自动排布时间表、导出 .ics 日历文件的场景，都应该使用此 skill。
---

# 自动日程生成器

## ⛔ 核心规则：使用子 Agent

**必须通过 `Agent` 工具派发子 Agent 生成 ICS，禁止在主对话中直接生成。** 原因：多文件读写 + 校验 + 可能重试会导致上下文爆炸。

Agent prompt 模板：
> 读取 config/todo.json、config/habit.md、docs/skills/auto-schedule/learning.md。为 [日期范围] 生成整周 ICS 写入 config/schedule.ics。每个任务的 preferred_start_time~preferred_end_time 是硬约束。生成后运行 python3 docs/skills/auto-schedule/scripts/validate_schedule.py --tasks config/todo.json --ics config/schedule.ics 校验，不匹配则修正重试（最多2次）。

## ⛔ 安全约束

| 允许 | 禁止 |
|------|------|
| ✅ Read 任意文件 | ❌ 修改源码（.go/.vue/.ts/.py/.js） |
| ✅ Write `config/schedule.ics` | ❌ 创建/删除文件 |
| ✅ Edit `learning.md`（去重后） | ❌ 编辑 config/todo.json、config/habit.md |

## 工作流程

### 1. 加载上下文
- 读取 `config/todo.json` — 关注 `preferred_start_time`/`preferred_end_time`
- 读取 `config/habit.md` — 工作时间、午餐时段
- 读取 `docs/skills/auto-schedule/learning.md` — 历史错误，避免重犯

### 2. 生成 ICS

调度优先级（严格遵守）：
1. **偏好时段硬约束** — `preferred_start_time`~`preferred_end_time` 内安排，这是最高优先级
2. 固定时间任务 → 直接锁定
3. 精力匹配（deep_work=上午，shallow_work=下午）
4. 优先级排序（high > medium > low）
5. 午餐保护（不占用 lunch_start~lunch_end）
6. daily 任务每天一个实例，weekly 任务仅在对应星期几
7. 只输出任务，不输出午餐/休息

**整周7天缺一不可。** 忽略 todo.json 中的 date 字段和系统日期。

### 3. 校验（不可跳过）

```bash
python3 docs/skills/auto-schedule/scripts/validate_schedule.py --tasks config/todo.json --ics config/schedule.ics
```
- 退出码 0 → 通过，继续下一步
- 退出码 1 → 修正后重试（最多2次），仍失败则记录 learning.md（如果已经有重复经验，则添加至当前SKILL.md中，作为硬约束）
- 退出码 2 → 检查文件路径

### 4. 记录经验

如有不匹配：**先读 learning.md，查重后追加**。相同任务名+时段组合不重复记录。格式：`| 日期 | 任务名 | 偏好时段 | 实际时段 | 原因 |`

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
