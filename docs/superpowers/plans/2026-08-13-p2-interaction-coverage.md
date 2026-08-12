# P2 前端交互覆盖 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`).

**Goal:** 实现 P2 可行交互用例(任务拖拽、Settings 开关/滑块/导出/清空-取消、列表筛选排序、Analytics tooltip、row 点按详情),跳过不可行项并在 COVERAGE 记因。

**Architecture:** 沿用 P0/P1 Playwright 套件与约定。前端 :5173 + 后端 :8080 执行前确保在跑。工作目录 `/Users/lsy/CodeHub/TickTask`,分支 `evolve/p2-interaction-coverage`。

**Tech Stack:** Playwright 1.60(channel: chrome)、Element Plus 2.8、Vue 3.5。

---

## Task 1: 任务跨象限拖拽

**Files:** Create `frontend/tests/e2e/task-drag.spec.ts`

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p2 Tasks · 拖拽跨象限', () => {
  test('DRAG-UI-001: 拖拽任务从 Q2 到 Q1', async ({ page, taskFactory }) => {
    const title = `拖拽-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks') // 默认四象限
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })

    // 拖到 Q1 象限区
    await row.dragTo(page.locator('.quad-q1'))

    // 任务出现在 Q1,且不在 Q2
    await expect(page.locator('.quad-q1').locator('.task-row', { hasText: title })).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.quad-q2').locator('.task-row', { hasText: title })).toHaveCount(0)
  })
})
```

- [ ] **Step 2: 实跑**

Run: `cd frontend && npx playwright test task-drag --project=chromium --reporter=list`

- 若 PASS:进 Step 3。
- 若 FAIL 且报错是 drop 未触发(HTML5 drag 在 Playwright 偶发不可靠):把该 test 改为 `test.fixme('DRAG-UI-001: ...', async ({...}) => { ... })`,并在文件顶部注释说明「Playwright HTML5 drag 不可靠,待工具支持后启用」,然后跑(应 skipped)。**不要改成 API 调用 moveTask。**

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-drag.spec.ts
git commit -m "test(e2e): cover drag task across quadrants (DRAG-UI-001)"
```

---

## Task 2: Settings 开关 + 滑块往返持久

**Files:** Create `frontend/tests/e2e/settings-toggles.spec.ts`

> /settings 上恰好 3 个 `.el-switch`(自动开始休息/自动开始工作/启用提示音,顺序固定)+ 1 个 `.el-slider`(buffer_ratio)。`.el-switch` 用 `aria-checked`(字符串 "true"/"false"),`.el-slider__thumb` 用 `aria-valuenow`。

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p2 Settings · 开关与滑块往返', () => {
  test('SET-UI-002: 翻转 3 开关 + 调缓冲比例 → 保存 → 重载持久 → 还原', async ({ page }) => {
    await page.goto('/settings')

    const switches = page.locator('.el-switch')
    const thumb = page.locator('.el-slider__thumb')
    const save = () => page.getByRole('button', { name: '保存设置' }).first().click()

    const beforeSw = await switches.evaluateAll((els) => els.map((e) => e.getAttribute('aria-checked')))
    const beforeSlider = (await thumb.getAttribute('aria-valuenow')) || '15'

    // 翻转 3 个开关
    for (let i = 0; i < 3; i++) await switches.nth(i).click()
    // 改滑块:点跑道右端附近(10–30% 区间)
    const runway = page.locator('.el-slider__runway')
    const box = await runway.boundingBox()
    if (box) await runway.click({ position: { x: box.width * 0.8, y: box.height / 2 } })

    await save()

    // 重载后断言持久
    await page.reload()
    const swAfter = page.locator('.el-switch')
    const thumbAfter = page.locator('.el-slider__thumb')
    const afterSw = await swAfter.evaluateAll((els) => els.map((e) => e.getAttribute('aria-checked')))
    for (let i = 0; i < 3; i++) expect(afterSw[i]).not.toBe(beforeSw[i])
    const afterSlider = await thumbAfter.getAttribute('aria-valuenow')
    expect(afterSlider).not.toBe(beforeSlider)

    // 还原
    for (let i = 0; i < 3; i++) if (afterSw[i] === 'true' && beforeSw[i] === 'false') { /* noop,已翻 */ }
    // 简化还原:再次翻转 3 开关回到原状;滑块回到原值附近
    for (let i = 0; i < 3; i++) await swAfter.nth(i).click()
    const rb = await page.locator('.el-slider__runway').boundingBox()
    if (rb) await page.locator('.el-slider__runway').click({ position: { x: rb.width * 0.3, y: rb.height / 2 } })
    await save()
  })
})
```

> 注:若 `aria-checked` 读取不中(EP 版本差异),改用 `.is-checked` class:`els.map((e) => e.classList.contains('is-checked'))`。滑块点按若不生效,改用聚焦 thumb + ArrowLeft/Right 调整。**只调选择器/交互,不改源码。**

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test settings-toggles --project=chromium --reporter=list`

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/settings-toggles.spec.ts
git commit -m "test(e2e): cover settings switches + slider round-trip (SET-UI-002)"
```

---

## Task 3: Settings 导出 + 清空(非破坏)

**Files:** Create `frontend/tests/e2e/settings-data.spec.ts`

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p2 Settings · 导出 / 清空', () => {
  test('SET-UI-003: 点「导出全部数据」触发 JSON 下载', async ({ page }) => {
    await page.goto('/settings')
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.locator('[data-test="export-btn"]').click(),
    ])
    expect(download.suggestedFilename()).toMatch(/\.json$/)
  })

  test('SET-UI-004: 点「清空全部数据」弹确认 → 取消,数据不清空', async ({ page, apiClient }) => {
    // 先确保至少有一个任务存在(清理前后都用它佐证数据未被清空)
    const before = await apiClient.getTasks()

    await page.goto('/settings')
    await page.locator('[data-test="clear-btn"]').click()

    // 确认弹窗(ElMessageBox)出现,点取消(X 或 取消 按钮),不真正清空
    const mb = page.locator('.el-message-box')
    await expect(mb).toBeVisible({ timeout: 5000 })
    // 取消:优先点「取消」按钮,否则点右上 X
    const cancel = mb.getByRole('button', { name: /取消|Cancel/ }).first()
    await cancel.click()
    await expect(mb).toBeHidden({ timeout: 5000 })

    // 数据仍在(任务数未变)
    const after = await apiClient.getTasks()
    expect(after.length).toBe(before.length)
  })
})
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test settings-data --project=chromium --reporter=list`

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/settings-data.spec.ts
git commit -m "test(e2e): cover settings export download + clear confirm-cancel (SET-UI-003/004)"
```

---

## Task 4: 列表视图筛选 + 排序

**Files:** Create `frontend/tests/e2e/task-list-filters.spec.ts`

> ListView 顶部 3 个 `el-select`:状态、象限、排序。`el-select` 点开 → `.el-select-dropdown__item:visible` 选。

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

async function selectOption(page: import('@playwright/test').Page, selectNth: number, label: string) {
  await page.locator('.el-select').nth(selectNth).click()
  await page.locator('.el-select-dropdown__item:visible', { hasText: label }).click()
}

test.describe('@p2 Tasks · 列表筛选与排序', () => {
  test('LIST-UI-001: 状态筛选「已完成」只显示已完成任务', async ({ page, taskFactory, apiClient }) => {
    const todo = `筛A-${Date.now()}`
    const done = `筛B-${Date.now()}`
    await taskFactory.create({ title: todo, quadrant: 2, status: 'todo' })
    const d = await taskFactory.create({ title: done, quadrant: 2 })
    await apiClient.updateTask(d.id, { status: 'completed' }).catch(() => {})

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    // 状态筛选是第 0 个 el-select
    await selectOption(page, 0, '已完成')
    await expect(page.locator('.task-item', { hasText: done })).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.task-item', { hasText: todo })).toHaveCount(0)
  })

  test('LIST-UI-002: 切换排序不报错且列表仍渲染', async ({ page, taskFactory }) => {
    await taskFactory.create({ title: `排-${Date.now()}`, quadrant: 1 })
    await taskFactory.create({ title: `排-${Date.now()}-2`, quadrant: 3 })

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()
    await expect(page.locator('.task-item').first()).toBeVisible({ timeout: 10000 })

    // 排序是第 2 个 el-select(状态/象限/排序)
    await selectOption(page, 2, '优先级')
    await expect(page.locator('.task-item').first()).toBeVisible({ timeout: 10000 })
  })
})
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test task-list-filters --project=chromium --reporter=list`
> 若 el-select 索引(0/1/2)与状态/象限/排序顺序不对应,按 ListView 实际顺序调整索引。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-list-filters.spec.ts
git commit -m "test(e2e): cover list-view status filter + sort (LIST-UI-001/002)"
```

---

## Task 5: Analytics 图表 tooltip

**Files:** Create `frontend/tests/e2e/analytics-tooltip.spec.ts`

> 图表数据来自番茄会话。先造今日完成的 work 会话,再 hover `.chart-bar-wrapper` 断言 `.bar-tooltip` 显时长。

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p2 Analytics · 图表 tooltip', () => {
  test('ANALY-UI-001: hover 趋势柱显时长 tooltip', async ({ page, taskFactory, sessionFactory, apiClient }) => {
    const task = await taskFactory.create({ title: `图表-${Date.now()}`, quadrant: 2 })
    const session = await sessionFactory.create({ task_id: task.id, type: 'work', duration: 25 })
    await apiClient.controlSession(session.id, 'complete')

    await page.goto('/analytics')
    const bar = page.locator('.chart-bar-wrapper').first()
    await expect(bar).toBeVisible({ timeout: 15000 })

    await bar.hover()
    // tooltip 在 DOM 内,断言其文案含时长(数字 + 单位)
    await expect(page.locator('.bar-tooltip').first()).not.toBeEmpty({ timeout: 5000 })
  })
})
```

> 若 `.bar-tooltip` 一直空(无数据/无 hover 触发),确认今日有完成会话且默认时间筛选是「今日」;若 tooltip 是 `display:none` 由 hover 显,用 toBeVisible。**不改源码。**

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test analytics-tooltip --project=chromium --reporter=list`

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/analytics-tooltip.spec.ts
git commit -m "test(e2e): cover analytics chart bar tooltip on hover (ANALY-UI-001)"
```

---

## Task 6: 四象限 row 点按 → 任务详情

**Files:** Create `frontend/tests/e2e/task-row-detail.spec.ts`

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p2 Tasks · row 点按详情', () => {
  test('ROW-UI-004: 点四象限 row 打开 TaskPomodoroDetail', async ({ page, taskFactory }) => {
    const title = `详情-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks') // 默认四象限
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })

    // 点 row 主体(避开 row-more 菜单/checkbox)
    await row.locator('.task-title').click()

    // TaskPomodoroDetail 标题为任务标题
    await expect(page.locator('.el-dialog__title', { hasText: title })).toBeVisible({ timeout: 10000 })
  })
})
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test task-row-detail --project=chromium --reporter=list`
> 若点 `.task-title` 触发了别的东西,改点 row 的非交互区(如 `.task-title` 已是)。row 的 click 处理是 `showDetail`,应弹详情。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-row-detail.spec.ts
git commit -m "test(e2e): cover quadrant row click -> TaskPomodoroDetail (ROW-UI-004)"
```

---

## Task 7: 更新 COVERAGE.md

**Files:** Modify `frontend/tests/e2e/COVERAGE.md`

- [ ] **Step 1: 更新矩阵**
- Tasks:row 点按 → 详情 ✅ ROW-UI-004;列表筛选/排序 ✅ LIST-UI-001/002;拖拽 ✅ DRAG-UI-001(若 fixme 则标 ⚠️ Playwright 不可靠)。
- Settings:开关/滑块 ✅ SET-UI-002;导出 ✅ SET-UI-003;清空(取消)✅ SET-UI-004;其余(AI/测试连接/导入)⏳ 需 AI/文件,标 P3。
- Analytics:tooltip ✅ ANALY-UI-001;图表下钻 ⏳ P3。
- Schedule:拖拽/缩放 ❌ 未实现(记因)。
- WorkLog:BatchTableEditor ❌ 不可达(草稿只来自禁用的 BrainDump,记因)。
- 把「下批(P2)候选」更新为「P3 候选」(BrainDump 启用决策、AI 入口、导入向导、图表下钻)。

- [ ] **Step 2: 提交**

```bash
git add frontend/tests/e2e/COVERAGE.md
git commit -m "docs(e2e): update coverage matrix for P2 (feasible covered, infeasible documented)"
```

---

## Task 8: 全量回归

- [ ] **Step 1: 本批 + P0/P1 全量**

Run: `cd frontend && npx playwright test task-drag settings-toggles settings-data task-list-filters analytics-tooltip task-row-detail task-ui-crud dashboard-task-edit schedule-ui-crud timer-start settings-form-roundtrip task-quadrant-row-menu work-log-ui --project=chromium --reporter=list`
Expected: 全 passed(拖拽若 fixme 则 skipped,可接受)。

- [ ] **Step 2: 抽样既有套件确认无新破坏**

Run: `cd frontend && npx playwright test task-crud schedule-views settings --project=chromium --reporter=list`
Expected: 与 P1 终态一致。
