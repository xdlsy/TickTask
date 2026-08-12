# P2 前端交互覆盖设计

- 日期:2026-08-13
- 状态:已批准(待用户审阅规格)
- 前置:P0(@ b9865c2)+ P1(@ 46f19b1)已合并并推送。本批按 `COVERAGE.md` 的 P2 候选继续,只做**可行项**。

## 1. 范围

### 实现(可行项)
| 用例 | 文件 | 流程 |
|---|---|---|
| 任务跨象限拖拽 | `task-drag.spec.ts` | 建 Q2 任务,`dragTo` 到 Q1 → 出现在 `.quad-q1` |
| Settings 开关+滑块往返 | `settings-toggles.spec.ts` | 翻 3 开关 + 调 buffer_ratio → 保存 → 重载持久 → 还原 |
| Settings 导出 | `settings-data.spec.ts` | 点「导出全部数据」→ 触发 JSON 下载 |
| Settings 清空(非破坏) | 同上 | 点「清空全部数据」→ 确认弹窗 → 取消,数据不动 |
| 列表筛选(状态/象限) | `task-list-filters.spec.ts` | 设筛选 → 列表收敛 |
| 列表排序 | 同上 | 切排序 → 顺序变化 |
| Analytics 图表 tooltip | `analytics-tooltip.spec.ts` | hover 柱状条 → `.bar-tooltip` 显时长 |
| 四象限 row 点按 → 详情 | `task-row-detail.spec.ts` | 点 row → TaskPomodoroDetail 弹出 |

### 跳过 + 记因(写进 COVERAGE)
- **日程拖拽/缩放**:未实现(schedule 组件无 `draggable`/`resizable`)。
- **WorkLog BatchTableEditor**:不可达 —— 草稿只来自 `store.structureBrainDump`(`WorkLog.vue:123`),而 BrainDump 被 `v-if="false"` 禁用。
- **BrainDump→AI 流**:P1 决策保持禁用。
- **Analytics「获取洞察」/ Settings「测试连接」**:需 AI 配置,默认环境不可测。

## 2. 关键技术点

- **拖拽**:`QuadrantView` 用 Vue 状态(`draggedTask.value`)而非 dataTransfer 载荷,Playwright `dragTo` 应能触发 `dragstart`→`drop` → `moveTask`。HTML5 drag 有抖动风险:用例若偶发 red,允许适度等待/重试选择器,**但不改成 API 调用**;若确属 Playwright 不可靠,标记 `test.fixme` 并在 COVERAGE 注明,不强行留红。
- **滑块往返**:用 `.el-slider__thumb` 的 `aria-valuenow` 断言持久值;改值用点击跑道位置或拖 thumb。
- **开关往返**:用 `.el-switch.is-checked`(或 `aria-checked`)断言。
- **导出**:Playwright `waitForEvent('download')` 捕获,断言 `suggestedFilename()` 含 `.json`。
- **清空**:只测「弹确认 → 取消」非破坏路径,绝不在 E2E 里真正清库。
- **Analytics tooltip**:需先有数据(经 sessionFactory/apiClient 造会话),再 hover `.chart-bar-wrapper` 断言 `.bar-tooltip` 可见。

## 3. 约定(继承 P0/P1)

判定 100% 真实 UI;造数据/清理可走 API;Element Plus 多菜单 `:visible`、文案 `exact`;TDD 即时修;真实 Chrome;新 spec `{feature}.spec.ts`,`@p2` 标签;Given/When/Then 注释。

## 4. 验收

- 本批新增用例全 green(允许拖拽用例在确属工具不可靠时 `test.fixme`)。
- `COVERAGE.md` 更新:可行项 ✅ + 用例 ID;跳过项 ❌ + 原因。
- 不破坏既有套件。

## 5. 非目标

BrainDump 重启用;BatchTableEditor(不可达);日程拖拽(未实现);AI 依赖型入口。
