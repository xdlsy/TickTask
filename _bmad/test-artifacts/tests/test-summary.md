# 番茄钟界面按钮测试 — Test Automation Summary

**日期:** 2026-05-26
**测试框架:** Vitest + @vue/test-utils + jsdom

---

## 测试范围：7 个按钮全覆盖

| # | 按钮 | CSS 选择器 | 功能 | 测试数 |
|---|------|-----------|------|--------|
| 1 | 开始专注 | `.start-btn` | 启动 25 分钟工作计时 | 5 |
| 2 | 暂停 | `.pause-btn` | 暂停当前计时 | 3 |
| 3 | 继续 | `.resume-btn` | 恢复暂停的计时 | 3 |
| 4 | 完成 | `.complete-btn` | 手动完成计时 | 3 |
| 5 | 放弃 | `.abandon-btn` | 放弃计时（含确认弹窗） | 4 |
| 6 | 短休息 | `.quick-btn.short` | 启动 5 分钟短休息 | 3 |
| 7 | 长休息 | `.quick-btn.long` | 启动 15 分钟长休息 | 3 |

---

## 测试用例清单

### TimerControls.test.ts — 27 tests

#### 开始按钮可见性 (4)
- [x] 无会话时显示开始按钮
- [x] 会话已完成时显示开始按钮
- [x] 会话已放弃时显示开始按钮
- [x] 会话活跃时隐藏开始按钮

#### 操作按钮可见性 (5)
- [x] 运行时显示暂停按钮
- [x] 暂停时显示继续按钮
- [x] 运行时显示完成和放弃按钮
- [x] 暂停时显示完成和放弃按钮
- [x] 暂停和继续按钮不会同时出现

#### 快速操作按钮可见性 (3)
- [x] 无会话时始终显示快速操作按钮
- [x] 会话运行时始终显示快速操作按钮
- [x] 会话暂停时始终显示快速操作按钮

#### 开始专注 (2)
- [x] 点击后调用 createSession(null, 'work')
- [x] 失败时显示错误消息

#### 短休息 (2)
- [x] 点击后调用 createSession(null, 'short_break')
- [x] 失败时显示错误消息

#### 长休息 (2)
- [x] 点击后调用 createSession(null, 'long_break')
- [x] 失败时显示错误消息

#### 暂停 (2)
- [x] 点击后调用 controlSession('pause')
- [x] 失败时显示错误消息

#### 继续 (2)
- [x] 点击后调用 controlSession('resume')
- [x] 失败时显示错误消息

#### 完成 (2)
- [x] 点击后调用 controlSession('complete') 并显示成功消息
- [x] 失败时显示错误消息

#### 放弃 (3)
- [x] 确认后调用 controlSession('abandon', 'other')
- [x] 用户取消确认时不调用 controlSession
- [x] 确认后 API 失败时显示错误消息

### TimerDisplay.test.ts — 19 tests

- [x] 时间格式化、标签（准备开始/专注中/已暂停/计时器）
- [x] 关联任务名称显示
- [x] 尺寸 prop（默认/自定义）
- [x] 计算属性（radius/circumference/strokeDashoffset）
- [x] SVG 图标（工作/短休息/长休息）
- [x] 颜色（work→primary / short_break→sage / long_break→gold）

---

## 运行结果

```
✓ src/components/timer/TimerDisplay.test.ts (19 tests)
✓ src/components/timer/TimerControls.test.ts (27 tests)

Test Files  2 passed (2)
     Tests  46 passed (46)
```

---

## 测试覆盖总结

- **番茄钟按钮:** 7/7（全部覆盖）
- **Happy path:** 每个按钮至少 1 个正向用例 ✅
- **错误处理:** 每个按钮至少 1 个错误用例 ✅
- **边界用例:** 按钮互斥性、跨状态可见性 ✅
- **用户流程:** 确认弹窗、取消操作 ✅
