# P1 前端交互覆盖设计

- 日期:2026-08-10
- 状态:已批准(待用户审阅规格)
- 前置:P0 批次已合并(`main` @ b9865c2,12 条 E2E + 2 个 bug 修复)。本批按 `frontend/tests/e2e/COVERAGE.md` 的 P1 候选继续。

## 1. 范围

| 项 | 类型 | 说明 |
|---|---|---|
| ① TaskCard row-more 死入口 | 源码修复 + E2E | 四象限(row)模式 `.row-more` 是 no-op;补真实下拉菜单 |
| ② WorkLog 整页 E2E | 新增 E2E | 0 覆盖;手动 WorkItemForm 增改 + TodayPanorama 删 + 生成周报 |
| ③ Analytics「获取洞察」无反馈 | 源码修复 + E2E | 未配 AI 时点击无任何提示;加 `ElMessage.warning` |
| ④ 任务完成/删除无 toast | 源码增强 + E2E | Dashboard 与 QuadrantView 操作生效但无反馈;加 `ElMessage.success` |

> BrainDumpInput **保持 `v-if="false"` 禁用**(组件/store/API 已就绪,仅 UI 关闭)——本轮不碰、不测。

## 2. 决策(已与用户确认)

- BrainDump / brain-dump→AI 流:**保持禁用**,只测手动条目流。
- row-more:**修复成可用菜单**(不删除图标)。
- ③④ 小修:**纳入本批**。

## 3. row-more 修复设计

**文件**:`frontend/src/components/tasks/TaskCard.vue`(row 模式分支,模板第 15-17 行附近)。

把当前 no-op 的:
```html
<span class="row-more" @click.stop>
  <svg ...三圆点.../>
</span>
```
包进与 card 模式一致的 `el-dropdown`:
```html
<el-dropdown @command="handleCommand" trigger="click">
  <span class="row-more" @click.stop>
    <svg ...三圆点.../>
  </span>
  <template #dropdown>
    <el-dropdown-menu>
      <el-dropdown-item v-if="task.status !== 'completed'" command="startTimer">开始番茄</el-dropdown-item>
      <el-dropdown-item command="edit">编辑</el-dropdown-item>
      <el-dropdown-item command="ai-classify" :disabled="aiClassifying">AI 智能分类</el-dropdown-item>
      <el-dropdown-item v-if="task.status !== 'completed'" command="complete">完成</el-dropdown-item>
      <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
    </el-dropdown-menu>
  </template>
</el-dropdown>
```
- 复用已有 `handleCommand`(card 模式在用),不新增 emit(`edit/complete/delete` 已定义;QuadrantView 已绑定 handler)。
- row 菜单与 card 菜单内容一致,行为一致。

**E2E**(`task-quadrant-row-menu.spec.ts`,`/tasks` 默认四象限):
- ROW-UI-001:row `.row-more` →「编辑」→ QuadrantView TaskForm 弹出。
- ROW-UI-002:row 菜单「完成」→ 任务转已完成态。
- ROW-UI-003:row 菜单「删除」→ 任务消失。

## 4. WorkLog E2E 设计

**文件**:`frontend/tests/e2e/work-log-ui.spec.ts`。

WorkItemForm / TodayPanorama / ReportActions 自带 `data-test` 属性,选择器稳定。无 workLog factory → **条目用 UI 建当被测对象**,清理走 `apiClient`(若无 work-log 删除方法,则 UI 删除兜底,并在用例内注明)。

| 用例 | 流程 |
|---|---|
| WL-UI-001 | WorkItemForm 填 activity+时段+象限 →「添加」→ 出现在 TodayPanorama |
| WL-UI-002 | TodayPanorama `[data-test=edit-btn]` → WorkItemForm 改 activity → 保存 → 更新 |
| WL-UI-003 | TodayPanorama `[data-test=delete-btn]` → popconfirm 确认 → 条目消失 |
| WL-UI-004 | ReportActions 生成「本周周报」→ ReportDetail 展示(.rd-title 含「周报」) |

不涉及 AI(周报读 items)。

## 5. 小修设计

**③ Analytics**(`frontend/src/views/Analytics.vue`):「获取洞察」按钮 click 处理,在 `if (!agentStore.status.configured)` 分支加 `ElMessage.warning('请先在设置中配置 AI')`(若无该分支则补)。E2E:未配 AI(默认态)点按钮 → 出现 warning 提示。

**④ 任务 toast**(`Dashboard.vue` 的 `onCompleteTask`/`onDeleteTask` + `QuadrantView.vue` 的 `onCompleteTask`/`onDeleteTask`):成功后 `ElMessage.success('任务已完成' / '任务已删除')`。E2E:在已有 complete/delete 用例上断言 toast 出现(或新增轻量断言)。

## 6. 约定(继承 P0)

判定 100% 真实 UI;造数据/清理可走 API;Element Plus 多菜单用 `:visible`、按钮文案用 `exact`;TDD 即时修;真实 Chrome;新 spec `{feature}.spec.ts`,`@p1` 标签。

## 7. 验收

- ① row-more 三条用例 green,row 菜单可用。
- ② WorkLog 四条用例 green。
- ③④ 修复 + 对应断言 green。
- 全量:本批新增用例全 green,不破坏既有套件。
- `COVERAGE.md` 更新(row-more 移出疑似、标记 ③④ 已修、WorkLog 标 ✅)。

## 8. 非目标

BrainDump / brain-dump AI 流;WorkLog BatchTableEditor 批量入库(P2);拖拽;Settings 其余字段。
