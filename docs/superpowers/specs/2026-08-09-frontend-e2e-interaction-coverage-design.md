# 前端交互组件 E2E 覆盖设计

- 日期:2026-08-09
- 状态:已批准(待用户审阅规格)
- 范围:TickTask 前端 7 个页面(`/`、`/tasks`、`/timer`、`/schedule`、`/analytics`、`/work-log`、`/settings`)的交互组件 E2E 测试覆盖

## 1. 背景与动机

现有 Playwright E2E 套件(23 个 spec、~50 个带真实点击的用例)在**交互覆盖**上存在系统性盲区,曾导致「仪表盘点任务『编辑』无任何反应」这类致命回归漏网(见 `frontend/tests/e2e/dashboard-task-edit.spec.ts`,当前为 red,证实用例可拦截)。

探索阶段梳理出的盲区:
- **WorkLog 页:0 覆盖**(完全白地)
- **任务 UI 内 CRUD**:`/tasks` 下拉「编辑/删除」、表单「创建」均未覆盖
- **日程弹窗内增删改**:全是 API 路径,没走 UI(EventForm 对话框、点事件块、拖拽改时间)
- **Settings**:开关 / 滑块 / 数字字段几乎没碰,只点了「保存设置」
- **拖拽**:任务跨象限、日程事件挪动 / 缩放 —— 基本未测

顺带扫到的**疑似源码 bug**(SUSPECT):
- 仪表盘 `edit-task / complete-task / delete-task / start-timer` 的 `$emit` 指向空(编辑那条已确认是真 bug)
- Analytics 未配 AI 时点「获取洞察」无任何反馈
- WorkLog `BrainDumpInput` 被 `v-if="false"` 关闭(功能疑似停用)

## 2. 策略决策(已与用户确认)

| 决策点 | 选择 |
|---|---|
| 策略与粒度 | **分层:P0 流程 + 覆盖矩阵**。先写关键路径流程用例,每个交互组件落进覆盖矩阵文档(标注 已覆盖/待补/疑似bug),分批推进。 |
| 修 bug 策略 | **即时修复(TDD red→green)**。测试变红且属源码 bug 时当场修源码,边写边稳。 |
| 批次切法 | **按页面纵向(A)**:每页 P0 写完 → 实跑 → 修 red → 下一页。上下文连贯,稳一页进一页。 |

## 3. 数据准备边界(关键约定)

- **被测交互 = 100% 真实 UI 点击 / 输入**(满足「而不是调接口做判定」)。
- **前置造数据**(造一个任务 / 日程 / 会话当 fixtures)用现有 `taskFactory / scheduleFactory / sessionFactory` 经 API 准备 —— 确定性 + 自动清理,沿用现有 22 个 spec 的约定。
- 「创建」类**作为被测交互**时,UI 走完(如点「添加任务」→ 填表 → 保存);仅**清理**走 API。

> 判定走 UI;造数据 / 清理可走 API。

## 4. P0 首批用例(~11 条)

每个用例遵循 TDD cadence:**写 → 实跑(真实 Chrome)→ red 且属源码 bug 则即时修 → 复跑 green**。源码修复按 Conventional Commits 单独提交,与测试分离。

| # | 页面 | 流程 | 覆盖现状 | 备注 |
|---|---|---|---|---|
| 1 | Tasks | UI 创建任务:填标题 / 象限 → 保存 → 出现 | 未覆盖 | — |
| 2 | Tasks | 下拉「编辑」改标题 → 保存 → 更新 | 未覆盖(/tasks) | — |
| 3 | Tasks | 下拉「删除」+ 确认 → 消失 | 未覆盖 | — |
| 4 | Tasks | 下拉「完成」→ 状态切换 | 未覆盖 | — |
| 5 | Dashboard | 下拉「编辑」→ 弹窗 | 已写(red) | **疑似 bug,即时修** |
| 6 | Dashboard | 下拉「完成 / 删除」 | 未覆盖 | **疑似 bug(emit 指向空),即时修** |
| 7 | Schedule | 点空槽 → EventForm 创建 → 出现 | 仅 API 路径 | 改走 UI |
| 8 | Schedule | 点事件块 → 编辑 → 保存 | 仅 API 路径 | 改走 UI |
| 9 | Schedule | 对话框内删除事件 + 确认 | 仅 API 路径 | 改走 UI |
| 10 | Timer | 「开始专注」从零启动 → 运行态 | 仅间接覆盖 | 补独立用例 |
| 11 | Settings | 改数字字段 + 开关 → 保存 → 生效(往返) | 未覆盖 | — |

页面推进顺序(纵向 A):Tasks(#1–4)→ Dashboard(#5–6)→ Schedule(#7–9)→ Timer(#10)→ Settings(#11)。每页 P0 全绿后再进下一页。

## 5. 覆盖矩阵(交付物)

新增 `frontend/tests/e2e/COVERAGE.md`:**页面 × 交互组件 × {已覆盖用例ID / 待补 / 疑似bug}**。P0 首批落地后即更新。

进矩阵标 **P1/P2**(分批后续推进,不在本批实现):
- WorkLog 全页(今日条目增删改、批量入库、生成周/月/半年/年报)
- 拖拽(任务跨象限、日程事件挪动 / 缩放)—— HTML5 drag 在 Playwright 易抖动,单独评估
- Analytics 下钻 / 图表 tooltip
- Settings 全字段(服务商 / 模型 / API Key / 测试连接 / 导入导出 / 清空)
- 顶部导航 / 路由(部分已在 navigation.spec 覆盖,补交互向)

## 6. 文件与约定

- 新 spec 命名 `{page}-{flow}.spec.ts`,放 `frontend/tests/e2e/`。
- 沿用 `tests/support/fixtures`(`taskFactory` 等已带自动清理)。
- 真实 Chrome(`channel: 'chrome'`),`@p0` 标签,Given/When/Then 注释风格(同现有)。
- 选择器优先 `getByRole` / 稳定 class / 文案;Element Plus 多卡片下拉用 `:visible` 限定当前展开项(见 dashboard-task-edit 经验)。

## 7. 验收标准

- P0 首批 ~11 条用例全部 green(含修复暴露的源码 bug)。
- `frontend/tests/e2e/COVERAGE.md` 落地,每个交互组件有明确状态。
- 不破坏现有套件:新增用例独立可跑,不在 main 上引入新的持续 red。

## 8. 非目标(YAGNI)

- 不重写已覆盖的用例;只在「改走 UI」处新增。
- 不做性能 / a11y / 错误注入类用例(已有专门 spec)。
- 不碰后端 Go 代码(除非 P0 bug 根因在后端)。
