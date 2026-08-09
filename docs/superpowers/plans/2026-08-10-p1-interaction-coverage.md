# P1 前端交互覆盖 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** 补齐 P1 交互覆盖:① 修复 TaskCard 四象限 row 模式 `.row-more` 死入口;② WorkLog 整页 E2E(手动流);④ 任务完成/删除加成功 toast。③ Analytics「获取洞察」**经核实非 bug**(按钮在 `v-if="agentStore.status.configured"` 卡片内,未配 AI 时根本不渲染),从批次移除并更正 COVERAGE。

**Architecture:** 沿用 P0 的 Playwright 套件与约定(判定 100% 真实 UI;造数据/清理可走 API;Element Plus 多菜单用 `:visible`、文案用 `exact`;TDD 即时修;真实 Chrome)。前端 :5173 + 后端 :8080 执行前确保在跑。

**Tech Stack:** Playwright 1.60(channel: chrome)、Element Plus 2.8、Vue 3.5。

**工作目录:** `/Users/lsy/CodeHub/TickTask`;分支 `evolve/p1-interaction-coverage`(已含设计规格)。

---

## Task 1: row-more 修复 + 四象限 row 菜单 E2E(TDD)

**Files:**
- Modify: `frontend/src/components/tasks/TaskCard.vue`(row 模式 `.row-more`,约 15-17 行)
- Create: `frontend/tests/e2e/task-quadrant-row-menu.spec.ts`

- [ ] **Step 1: 写失败用例**

创建 `frontend/tests/e2e/task-quadrant-row-menu.spec.ts`:

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p1 Tasks · 四象限 row 菜单', () => {
  test('ROW-UI-001: row「...」→「编辑」打开 QuadrantView 编辑弹窗', async ({ page, taskFactory }) => {
    const title = `行菜单编辑-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks') // 默认四象限
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })
    await row.locator('.row-more').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '编辑' }).click()

    await expect(page.locator('.el-dialog__title', { hasText: '编辑任务' })).toBeVisible({ timeout: 10000 })
  })

  test('ROW-UI-002: row 菜单「完成」切换到已完成态', async ({ page, taskFactory }) => {
    const title = `行菜单完成-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks')
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })
    await row.locator('.row-more').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '完成' }).click()

    await expect(row).toHaveClass(/task-completed/, { timeout: 10000 })
  })

  test('ROW-UI-003: row 菜单「删除」移除任务', async ({ page, taskFactory }) => {
    const title = `行菜单删除-${Date.now()}`
    await taskFactory.create({ title, quadrant: 3 })

    await page.goto('/tasks')
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })
    await row.locator('.row-more').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '删除' }).click()

    await expect(page.locator('.task-row', { hasText: title })).toHaveCount(0, { timeout: 10000 })
  })
})
```

- [ ] **Step 2: 实跑,预期 3 条 FAIL(red)**

Run: `cd frontend && npx playwright test task-quadrant-row-menu --project=chromium --reporter=list`
Expected: 3 failed(`.row-more` 当前 no-op,菜单不展开)。

- [ ] **Step 3: 修 TaskCard.vue — row 模式补真实下拉**

`frontend/src/components/tasks/TaskCard.vue`,把 row 模式里这段:

```html
        <span class="row-more" @click.stop>
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>
        </span>
```

替换为(复用 card 模式同款 `el-dropdown` + 已有 `handleCommand`):

```html
        <el-dropdown @command="handleCommand" trigger="click">
          <span class="row-more" @click.stop>
            <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>
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

- [ ] **Step 4: 实跑,预期 3 条 PASS(green)**

Run: `cd frontend && npx playwright test task-quadrant-row-menu --project=chromium --reporter=list`
Expected: 3 passed。

- [ ] **Step 5: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无新增报错。

- [ ] **Step 6: 提交(源码与测试分离)**

```bash
git add frontend/src/components/tasks/TaskCard.vue
git commit -m "fix(taskcard): make quadrant row-mode 'more' menu functional"
git add frontend/tests/e2e/task-quadrant-row-menu.spec.ts
git commit -m "test(e2e): cover quadrant row-mode edit/complete/delete menu (ROW-UI-001/002/003)"
```

---

## Task 2: WorkLog 整页 E2E(手动流)

**Files:**
- Create: `frontend/tests/e2e/work-log-ui.spec.ts`

> WorkItemForm 必填 `start_time`/`end_time`(el-time-select,`data-test=start-input/end-input`)+ `activity`(`data-test=activity-input`)。提交按钮 `data-test=submit-btn`(新增态文案「添加」/编辑态「保存」)。TodayPanorama 用 `el-table`,操作列 `data-test=edit-btn`/`delete-btn`(delete 在 `el-popconfirm`「确定删除?」内)。ReportActions 是 `+ 生成报告 ▼` 下拉。无 workLog factory 与 apiClient 方法 → **条目用 UI 建,清理用 UI 删**(删 `E2E-` 前缀条目)。

- [ ] **Step 1: 写用例文件(含 time-select 辅助 + UI 清理)**

创建 `frontend/tests/e2e/work-log-ui.spec.ts`:

```ts
import { test, expect } from '../support/fixtures'

const PREFIX = 'E2E-'

// el-time-select:点开下拉,选可见的时间项
async function pickTime(page: import('@playwright/test').Page, dataTest: string, time: string) {
  await page.locator(`[data-test="${dataTest}"]`).click()
  await page.locator('.el-select-dropdown__item:visible', { hasText: time }).click()
}

// 用 UI 录入一条工作条目
async function addItem(page: import('@playwright/test').Page, activity: string) {
  await page.goto('/work-log')
  await page.locator('[data-test="activity-input"] input, [data-test="activity-input"]').first().fill(activity)
  await pickTime(page, 'start-input', '09:00')
  await pickTime(page, 'end-input', '10:00')
  await page.locator('[data-test="submit-btn"]').click()
  await expect(page.getByText(activity).first()).toBeVisible({ timeout: 10000 })
}

// 清理:删除今日所有 E2E- 前缀条目(走 UI 删除 + popconfirm)
async function cleanup(page: import('@playwright/test').Page) {
  await page.goto('/work-log')
  let remaining = page.locator('.el-table__row').filter({ hasText: PREFIX })
  while ((await remaining.count()) > 0) {
    const row = remaining.first()
    await row.locator('[data-test="delete-btn"]').click()
    await page.getByRole('button', { name: /确定删除|确定|OK/ }).click()
    await page.waitForTimeout(300)
    remaining = page.locator('.el-table__row').filter({ hasText: PREFIX })
  }
}

test.describe('@p1 WorkLog · 手动条目流', () => {
  test('WL-UI-001: WorkItemForm 录入条目 → 出现在 TodayPanorama', async ({ page }) => {
    const activity = `${PREFIX}录入-${Date.now()}`
    try {
      await addItem(page, activity)
      await expect(page.locator('.el-table__row').filter({ hasText: activity })).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup(page)
    }
  })

  test('WL-UI-002: TodayPanorama「编辑」改 activity → 保存生效', async ({ page }) => {
    const original = `${PREFIX}改前-${Date.now()}`
    const changed = `${original}-改`
    try {
      await addItem(page, original)
      const row = page.locator('.el-table__row').filter({ hasText: original })
      await row.locator('[data-test="edit-btn"]').click()

      // 编辑态 WorkItemForm(submit-btn 文案变「保存」)
      const editInput = page.locator('[data-test="activity-input"] input, [data-test="activity-input"]').first()
      await editInput.fill(changed)
      await page.locator('[data-test="submit-btn"]').click()

      await expect(page.getByText(changed).first()).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup(page)
    }
  })

  test('WL-UI-003: TodayPanorama「删除」+ 确认 → 条目消失', async ({ page }) => {
    const activity = `${PREFIX}删-${Date.now()}`
    await addItem(page, activity)

    const row = page.locator('.el-table__row').filter({ hasText: activity })
    await row.locator('[data-test="delete-btn"]').click()
    await page.getByRole('button', { name: /确定删除|确定|OK/ }).click()

    await expect(page.getByText(activity)).toHaveCount(0, { timeout: 10000 })
  })

  test('WL-UI-004: 生成「本周周报」→ ReportDetail 展示', async ({ page }) => {
    const activity = `${PREFIX}周报-${Date.now()}`
    try {
      await addItem(page, activity)
      await page.locator('.action-btn', { hasText: '生成报告' }).click()
      await page.locator('.el-dropdown-menu__item:visible', { hasText: '本周周报' }).click()

      // ReportDetail 出现且标题含「周报」
      await expect(page.locator('.rd-title, [class*="rd-title"]').first()).toContainText('周报', { timeout: 15000 })
    } finally {
      await cleanup(page)
    }
  })
})
```

> 已知风险点(执行时若 red 按此调整,不改源码):
> - `activity-input` 若是 `el-input`,`[data-test="activity-input"]` 是外层;`.fill()` 需定位到内层 `input`,故用 `[data-test="activity-input"] input` 回退(已写)。若仍不中,改 `.getByPlaceholder(...)`(查 WorkItemForm placeholder)。
> - `el-time-select` 下拉项为 `.el-select-dropdown__item`;若渲染为别的 class,改用可见项 + 文案。
> - 周报生成可能因本周无 items 而空;本用例先录一条保证有数据。生成的报告记录是派生数据,接受留存。

- [ ] **Step 2: 实跑,逐条调试到全 PASS**

Run: `cd frontend && npx playwright test work-log-ui --project=chromium --reporter=list`
Expected: 4 passed。单条 red 时按上面「已知风险点」调选择器;**不改源码**(WorkLog 功能已可用)。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/work-log-ui.spec.ts
git commit -m "test(e2e): cover WorkLog manual item CRUD + weekly report (WL-UI-001..004)"
```

---

## Task 3: 任务完成/删除成功 toast + 断言

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`(`onCompleteTask`/`onDeleteTask`)
- Modify: `frontend/src/components/tasks/QuadrantView.vue`(`onCompleteTask`/`onDeleteTask`,约 48-49 行)
- Modify: `frontend/tests/e2e/dashboard-task-edit.spec.ts`(DASH-EDIT-E2E-002 加 toast 断言)

- [ ] **Step 1: Dashboard 加 toast**

`frontend/src/views/Dashboard.vue`,把:

```ts
async function onCompleteTask(id: string) {
  await taskStore.markCompleted(id)
}

async function onDeleteTask(id: string) {
  await taskStore.deleteTask(id)
}
```

改为:

```ts
async function onCompleteTask(id: string) {
  await taskStore.markCompleted(id)
  ElMessage.success('任务已完成')
}

async function onDeleteTask(id: string) {
  await taskStore.deleteTask(id)
  ElMessage.success('任务已删除')
}
```

(`ElMessage` 已在 Dashboard 顶部导入。)

- [ ] **Step 2: QuadrantView 加 toast**

`frontend/src/components/tasks/QuadrantView.vue`,把:

```ts
async function onCompleteTask(id: string) { await taskStore.markCompleted(id) }
async function onDeleteTask(id: string) { await taskStore.deleteTask(id) }
```

改为:

```ts
async function onCompleteTask(id: string) { await taskStore.markCompleted(id); ElMessage.success('任务已完成') }
async function onDeleteTask(id: string) { await taskStore.deleteTask(id); ElMessage.success('任务已删除') }
```

确认顶部已 `import { ElMessage } from 'element-plus'`;若无则补。

- [ ] **Step 3: 给 row 菜单完成用例加 toast 断言(QuadrantView 侧)**

`frontend/tests/e2e/task-quadrant-row-menu.spec.ts`,在 ROW-UI-002 的状态断言后补一行:

```ts
    await expect(row).toHaveClass(/task-completed/, { timeout: 10000 })
    await expect(page.locator('.el-message--success', { hasText: '已完成' })).toBeVisible({ timeout: 5000 })
```

- [ ] **Step 4: 给 Dashboard 完成用例加 toast 断言**

`frontend/tests/e2e/dashboard-task-edit.spec.ts`,在 DASH-EDIT-E2E-002 的 `toHaveClass(/task-completed/...)` 后补:

```ts
    await expect(card).toHaveClass(/task-completed/, { timeout: 10000 })
    await expect(page.locator('.el-message--success', { hasText: '已完成' })).toBeVisible({ timeout: 5000 })
```

- [ ] **Step 5: 类型检查 + 实跑相关用例**

Run: `cd frontend && npx vue-tsc --noEmit && npx playwright test dashboard-task-edit task-quadrant-row-menu --project=chromium --reporter=list`
Expected: vue-tsc 干净;相关用例全 PASS。

- [ ] **Step 6: 提交(源码与测试分离)**

```bash
git add frontend/src/views/Dashboard.vue frontend/src/components/tasks/QuadrantView.vue
git commit -m "feat(task): success toast on complete/delete (Dashboard + QuadrantView)"
git add frontend/tests/e2e/dashboard-task-edit.spec.ts frontend/tests/e2e/task-quadrant-row-menu.spec.ts
git commit -m "test(e2e): assert complete/delete success toast"
```

---

## Task 4: 更新 COVERAGE.md

**Files:**
- Modify: `frontend/tests/e2e/COVERAGE.md`

- [ ] **Step 1: 更新矩阵**

- Tasks 区:「四象限视图 row `.row-more`」由 🐞 改为 ✅ `task-quadrant-row-menu.spec.ts` ROW-UI-001~003 🛠。
- WorkLog 区:条目增删改 + 周报由 ⏳ P1 改为 ✅ `work-log-ui.spec.ts` WL-UI-001~004。
- Analytics 区:移除/更正「未配 AI 点获取洞察无反馈」(经核实:按钮在 `v-if=configured` 卡片内,未配 AI 不渲染,非 bug)。
- Dashboard 区:补注 ✅ 完成/删除带 toast。
- P1 候选:移除已完成的 row-more、Analytics(非 bug)、toast;保留拖拽、BatchTableEditor、BrainDump(继续禁用)。

- [ ] **Step 2: 提交**

```bash
git add frontend/tests/e2e/COVERAGE.md
git commit -m "docs(e2e): update coverage matrix for P1 (row-more fixed, WorkLog covered, Analytics non-bug)"
```

---

## Task 5: 全量回归

- [ ] **Step 1: 跑本批全部新增/改动文件**

Run: `cd frontend && npx playwright test task-quadrant-row-menu work-log-ui dashboard-task-edit --project=chromium --reporter=list`
Expected: 全 passed(row 3 + worklog 4 + dashboard 3 = 10)。

- [ ] **Step 2: 抽样既有套件确认无新破坏**

Run: `cd frontend && npx playwright test task-ui-crud task-crud schedule-ui-crud --project=chromium --reporter=list`
Expected: 与 P0 终态一致(既有 flaky 不计)。

---

## 规划期更正

- **③ Analytics「获取洞察」非 bug**:按钮在 `v-if="agentStore.status.configured"` 的卡片内(`Analytics.vue:252`),未配 AI 时不渲染,不存在「点了无反馈」。Agent A 早期盘点有误,本批移除并在 COVERAGE 更正。
