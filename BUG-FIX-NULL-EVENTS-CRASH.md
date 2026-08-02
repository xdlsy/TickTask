# Bug 修复：日程修订确认时 "应用修订失败，请重试"

**修复日期：** 2026-06-06
**分支：** `evolve/ai-scheduling-enhancements`
**提交：** `9eea638`
**工作流：** bmad-quick-dev (one-shot)

---

## 一、Bug 现象

用户在 Schedule.vue 页面完成日程修订预览后，点击"确定"按钮，弹出错误提示：

> **"应用修订失败，请重试"**

该错误信息是前端兜底文案，真实错误被隐藏。

---

## 二、根因分析

### 错误链路追踪

```
用户点击确定
  → Schedule.vue:378  scheduleStore.applyRevision()
  → schedule.ts:219   api.applyRevision()        POST /api/schedules/revise/apply
  → schedule.go:222   handler.ApplyRevision()
  → schedule_service.go:729  service.ApplyRevision()
      ├── ReadScheduleICS()    读取 config/schedule.ics
      ├── ParseICS()           解析 ICS → 返回 0 个事件（文件不存在/为空/解析失败）
      ├── var allEvents []ScheduleEvent    ← nil slice！
      └── return allEvents, nil            ← JSON: "events": null
  → 前端收到 {"applied": true, "events": null}
  → schedule.ts:220  appliedEvents = null
  → schedule.ts:222  null.map(...) → TypeError!
  → error.response === undefined (TypeError 不是 AxiosError)
  → Schedule.vue:382  error?.response?.data?.error || '应用修订失败，请重试'
  → 兜底文案，真实原因被吞没
```

### 双重根因

| 层 | 问题 | 位置 |
|---|------|------|
| **后端** | Go `var x []T` (nil slice) → JSON `null` | `schedule_service.go:758` ApplyRevision |
| **后端** | 同上模式 | `schedule_service.go:492` GenerateSchedule |
| **前端** | 未对 `null` 做防御，`.map()` 抛 TypeError | `schedule.ts:220` applyRevision |
| **前端** | 同上模式 | `schedule.ts:144` generateScheduleFromTasks |

### 为什么错误消息被吞没？

TypeError 没有 `response` 属性，`error?.response?.data?.error` 为 `undefined`，`||` 短路到兜底文案。后端真正的错误信息（如 `"读取修订后日程: no such file"`）只有在后端显式返回 500 时才能被前端展示，但 `ParseICS` **从不返回 error**（始终 `return events, nil`），所以空 ICS 文件不会触发 500。

---

## 三、修复工作流（bmad-quick-dev one-shot）

```
① /bmad-quick-dev "前台日程修订失败"
    │
    ├─ step-01-clarify-and-route: 意图识别 → 单目标 Bug → one-shot 路径
    │
    ├─ step-oneshot:
    │   ├─ Implement: 代码库追踪 → 定位根因 → 4 处修复
    │   ├─ Review: 对抗性审查 → 发现同模式漏洞 × 2 → 补修
    │   ├─ Generate Spec Trace → spec-fix-schedule-revision-null-events.md
    │   └─ Commit → 9eea638
    │
    └─ ✅ 完成
```

### 修复清单

| # | 文件 | 行号 | 修改 | 类型 |
|---|------|------|------|------|
| 1 | `backend/.../schedule_service.go` | 758 | `var allEvents []ScheduleEvent` → `make([]ScheduleEvent, 0)` | 根因修复 |
| 2 | `backend/.../schedule_service.go` | 492 | 同上（GenerateSchedule） | 同模式修复 |
| 3 | `frontend/src/stores/schedule.ts` | 220 | `res.data.events` → `(res.data.events ?? [])` | 防御层 |
| 4 | `frontend/src/stores/schedule.ts` | 144 | 同上（generateScheduleFromTasks） | 同模式修复 |

### 排查方法

```
1. 搜索错误文案 → 定位 Schedule.vue:382 兜底逻辑
2. 追踪调用链 → store → api client → backend handler → service
3. 阅读 service.ApplyRevision() → 发现 nil slice 声明
4. 阅读 ParseICS() → 发现从不返回 error（第60行: return events, nil）
5. curl 测试 → 确认返回 "events": null
6. 推理 TypeError 路径 → 确认兜底文案被触发
```

---

## 四、验证

```bash
# 正常场景
curl -X POST http://localhost:8080/api/schedules/revise/apply
→ {"applied":true,"events":[...]}  # ✅ 数组，非 null

# 异常场景（ICS 文件缺失）
rm config/schedule.ics
curl -X POST http://localhost:8080/api/schedules/revise/apply
→ {"error":"读取修订后日程: ..."}  # ✅ 明确的错误信息
```

---

## 五、经验教训

1. **Go nil slice vs empty slice**：`var x []T` 序列化为 `null`，`make([]T, 0)` 序列化为 `[]`。API 返回值应始终用后者。
2. **前端防御性编程**：后端返回值类型不等于运行时类型。`as ScheduleEvent[]` 只是编译期断言，`?? []` 是运行时保障。
3. **错误消息链路完整性**：`catch` 中不应依赖 `error.response`（Axios 特有），应优先检查 `error instanceof TypeError` 或直接用 `String(error)`。
4. **ParseICS 不应吞错误**：始终 `return events, nil` 隐藏了解析失败。应在事件数为 0 时检查输入是否有 VEVENT 块。
5. **对抗性审查的价值**：review 阶段在 `GenerateSchedule` 和 `generateScheduleFromTasks` 中发现了完全相同的漏洞，避免了下一次同样的 Bug。

---

*本报告由 bmad-quick-dev (one-shot) 工作流在 2026-06-06 自动生成 — 覆盖 定位 → 修复 → 审查 → 验证 → 提交 闭环*
