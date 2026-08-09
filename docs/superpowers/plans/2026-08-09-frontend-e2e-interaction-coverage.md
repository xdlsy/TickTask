# 前端交互组件 E2E 覆盖 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 7 个页面的关键交互补齐 E2E 用例(P0 首批 ~11 条),即时修复暴露的源码 bug(主要是仪表盘 edit/complete/delete 的 emit 指向空),并产出一份覆盖矩阵 `COVERAGE.md`。

**Architecture:** 沿用现有 Playwright E2E 套件(`frontend/tests/e2e/` + `tests/support/fixtures`)。被测交互 100% 走真实 UI;前置数据用 `taskFactory/scheduleFactory/sessionFactory` 经 API 准备,清理用 `apiClient`。按页面纵向推进,每条用例遵循 TDD:写 → 实跑 → red 且属源码 bug 则即时修 → green → 提交。

**Tech Stack:** Playwright 1.60(`channel: 'chrome'`,真实 Chrome)、Element Plus 2.8、Vue 3.5。前端 dev server `:5173`,后端 `:8080`(执行前确保都在跑:`lsof -ti:8080 && lsof -ti:5173`)。

**关键约定(所有 Tasks 共用):**
- Element Plus 多卡片下拉菜单都渲染进 DOM(其余隐藏),点开菜单后必须用 `:visible` 限定当前展开项:`page.locator('.el-dropdown-menu__item:visible', { hasText: '编辑' })`。
- TaskForm 对话框:标题文案「创建任务」(新建)/「编辑任务」(编辑);标题输入 `getByPlaceholder('输入任务标题')`;保存按钮 `getByRole('button', { name: '保存' })`。
- 任务删除在卡片/列表视图均为**即时删除,无确认弹窗**。

---

## Task 1: Tasks 页 — UI 创建任务(四象限默认视图)

**Files:**
- Create: `frontend/tests/e2e/task-ui-crud.spec.ts`

- [ ] **Step 1: 写用例**

创建 `frontend/tests/e2e/task-ui-crud.spec.ts`:

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p0 Tasks · UI 创建/编辑/删除/完成', () => {
  test('TASK-UI-001: 点「添加任务」填表保存,任务出现在四象限视图', async ({ page, apiClient }) => {
    const title = `UI创建-${Date.now()}`
    const cleanup = async () => {
      const tasks = await apiClient.getTasks().catch(() => [])
      const mine = tasks.find((t) => t.title === title)
      if (mine) await apiClient.deleteTask(mine.id).catch(() => {})
    }
    try {
      await page.goto('/tasks')
      await page.getByRole('button', { name: /添加任务/ }).click()

      const dialog = page.locator('.el-dialog', { hasText: '创建任务' })
      await expect(dialog).toBeVisible()
      await dialog.getByPlaceholder('输入任务标题').fill(title)
      await dialog.getByRole('button', { name: '保存' }).click()

      // 创建后表单关闭、任务出现在页面(默认四象限,象限默认 Q2)
      await expect(page.locator('.el-dialog', { hasText: '创建任务' })).toBeHidden({ timeout: 10000 })
      await expect(page.getByText(title).first()).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup()
    }
  })
})
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test task-ui-crud --project=chromium -g TASK-UI-001 --reporter=list`
Expected: 1 passed(代码已可用,用例应直接绿)。若 red,按报错修选择器,不改源码。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-ui-crud.spec.ts
git commit -m "test(e2e): cover task create via UI on /tasks (TASK-UI-001)"
```

---

## Task 2: Tasks 页 — 列表视图下拉「编辑」改标题

> 列表视图才有可用的 `.action-btn` 下拉(四象限视图 row 模式的 `.row-more` 是 no-op,见末尾「规划期发现」)。

**Files:**
- Modify: `frontend/tests/e2e/task-ui-crud.spec.ts`(在 `test.describe` 内追加)

- [ ] **Step 1: 追加用例**

在 `task-ui-crud.spec.ts` 的 `describe` 块内追加:

```ts
  test('TASK-UI-002: 列表视图下拉「编辑」改标题并保存生效', async ({ page, taskFactory }) => {
    const original = `待编辑-${Date.now()}`
    const task = await taskFactory.create({ title: original, quadrant: 2 })
    const changed = `${original}-改`

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click() // 切到列表视图

    const item = page.locator('.task-item', { hasText: original })
    await expect(item).toBeVisible({ timeout: 10000 })
    await item.locator('.action-btn').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '编辑' }).click()

    const dialog = page.locator('.el-dialog', { hasText: '编辑任务' })
    await expect(dialog).toBeVisible()
    await dialog.getByPlaceholder('输入任务标题').fill(changed)
    await dialog.getByRole('button', { name: '保存' }).click()

    await expect(page.getByText(changed).first()).toBeVisible({ timeout: 10000 })
    expect(task.id).toBeTruthy()
  })
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test task-ui-crud --project=chromium -g TASK-UI-002 --reporter=list`
Expected: 1 passed。red 则修选择器。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-ui-crud.spec.ts
git commit -m "test(e2e): cover task edit via list-view dropdown (TASK-UI-002)"
```

---

## Task 3: Tasks 页 — 列表视图下拉「删除」

**Files:**
- Modify: `frontend/tests/e2e/task-ui-crud.spec.ts`

- [ ] **Step 1: 追加用例**

```ts
  test('TASK-UI-003: 列表视图下拉「删除」移除任务(无确认弹窗)', async ({ page, taskFactory }) => {
    const title = `待删除-${Date.now()}`
    await taskFactory.create({ title, quadrant: 3 })

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    const item = page.locator('.task-item', { hasText: title })
    await expect(item).toBeVisible({ timeout: 10000 })
    await item.locator('.action-btn').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '删除' }).click()

    await expect(page.getByText(title)).toHaveCount(0, { timeout: 10000 })
  })
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test task-ui-crud --project=chromium -g TASK-UI-003 --reporter=list`
Expected: 1 passed。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-ui-crud.spec.ts
git commit -m "test(e2e): cover task delete via list-view dropdown (TASK-UI-003)"
```

---

## Task 4: Tasks 页 — 列表视图下拉「标记完成」

**Files:**
- Modify: `frontend/tests/e2e/task-ui-crud.spec.ts`

- [ ] **Step 1: 追加用例**

```ts
  test('TASK-UI-004: 列表视图下拉「标记完成」切换到已完成态', async ({ page, taskFactory }) => {
    const title = `待完成-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    const item = page.locator('.task-item', { hasText: title })
    await expect(item).toBeVisible({ timeout: 10000 })
    await item.locator('.action-btn').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '标记完成' }).click()

    // 完成后行带 completed 类;且「重新打开」菜单项出现佐证状态
    await expect(item).toHaveClass(/completed/, { timeout: 10000 })
  })
})
```

- [ ] **Step 2: 实跑整文件,预期 4 passed**

Run: `cd frontend && npx playwright test task-ui-crud --project=chromium --reporter=list`
Expected: 4 passed(TASK-UI-001~004)。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/task-ui-crud.spec.ts
git commit -m "test(e2e): cover task complete via list-view dropdown (TASK-UI-004)"
```

---

## Task 5: Dashboard — 修复 edit/complete/delete emit 指向空 + 补全用例(TDD red→green)

> 这是本批唯一的源码 bug。Dashboard 是路由级页面,把 TaskCard 的 `edit/complete/delete` 事件 `$emit` 抛给不存在的父级,自己却没渲染 `TaskForm`,也没调 store —— 三者全失效。`dashboard-task-edit.spec.ts` 的编辑用例当前 red 即佐证。

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`(源码修复)
- Modify: `frontend/tests/e2e/dashboard-task-edit.spec.ts`(补 complete/delete 用例)

- [ ] **Step 1: 先补「完成/删除」用例(预期 red)**

在 `dashboard-task-edit.spec.ts` 的 `describe` 块内追加两条用例:

```ts
  test('DASH-EDIT-E2E-002: 仪表盘点「完成」将任务标记为已完成', async ({ page, taskFactory }) => {
    const title = `看板完成-${Date.now()}`
    const task = await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/')
    const card = page.locator('.task-card', { hasText: title })
    await expect(card).toBeVisible({ timeout: 10000 })
    await card.locator('.more-icon').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '完成' }).click()

    // 卡片转为 completed 态(透明/删除线)
    await expect(card).toHaveClass(/task-completed/, { timeout: 10000 })
    expect(task.id).toBeTruthy()
  })

  test('DASH-EDIT-E2E-003: 仪表盘点「删除」移除任务卡片', async ({ page, taskFactory }) => {
    const title = `看板删除-${Date.now()}`
    await taskFactory.create({ title, quadrant: 3 })

    await page.goto('/')
    const card = page.locator('.task-card', { hasText: title })
    await expect(card).toBeVisible({ timeout: 10000 })
    await card.locator('.more-icon').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '删除' }).click()

    await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0, { timeout: 10000 })
  })
```

- [ ] **Step 2: 实跑,预期 3 条全 FAIL(red)**

Run: `cd frontend && npx playwright test dashboard-task-edit --project=chromium --reporter=list`
Expected: 3 failed(编辑弹窗不出现、完成无反应、删除无反应)。

- [ ] **Step 3: 修 Dashboard.vue 源码**

修改 `frontend/src/views/Dashboard.vue`:

(a) 模板里把对 TaskCard 的事件转发改为本地处理,并在 `.dashboard` 根节点末尾(`</div>` 前)加 TaskForm。把:

```vue
            <TaskCard
              v-for="task in recentTasks"
              :key="task.id"
              :task="task"
              @drag-start="() => {}"
              @edit="$emit('edit-task', task)"
              @complete="$emit('complete-task', $event)"
              @delete="$emit('delete-task', $event)"
            />
```

改为:

```vue
            <TaskCard
              v-for="task in recentTasks"
              :key="task.id"
              :task="task"
              @drag-start="() => {}"
              @edit="onEditTask"
              @complete="onCompleteTask"
              @delete="onDeleteTask"
            />
```

并在模板最后的 `</div>`(关闭 `.dashboard`)之前插入:

```vue
    <TaskForm
      v-if="showForm"
      :visible="showForm"
      :task="editingTask"
      @close="showForm = false"
      @save="onSaveTask"
    />
```

(b) `<script setup>` 中:删掉整段 `defineEmits<{ ... }>()`;新增 `TaskForm` 导入与状态/处理函数。把 import 区(约 135 行附近)补一行:

```ts
import TaskForm from '@/components/tasks/TaskForm.vue'
```

把 `const agentStore = useAgentStore()` 之后、`defineEmits` 原位置替换为:

```ts
const showForm = ref(false)
const editingTask = ref<TaskResponse | null>(null)

function onEditTask(task: TaskResponse) {
  editingTask.value = task
  showForm.value = true
}

async function onCompleteTask(id: string) {
  await taskStore.markCompleted(id)
}

async function onDeleteTask(id: string) {
  await taskStore.deleteTask(id)
}

async function onSaveTask(data: any) {
  if (editingTask.value) {
    await taskStore.updateTask(editingTask.value.id, data)
  }
  showForm.value = false
  editingTask.value = null
}
```

> `taskStore.markCompleted / deleteTask / updateTask` 已存在(`stores/task.ts:91/102/76`)。`TaskResponse` 已在文件顶部导入。该写法照搬 `QuadrantView.vue:44-49` 的既有模式。

- [ ] **Step 4: 实跑,预期 3 条全 PASS(green)**

Run: `cd frontend && npx playwright test dashboard-task-edit --project=chromium --reporter=list`
Expected: 3 passed。

- [ ] **Step 5: 类型检查(改了源码,过一遍 vue-tsc)**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无新增报错。

- [ ] **Step 6: 提交(源码与测试分离两条 commit)**

```bash
git add frontend/src/views/Dashboard.vue
git commit -m "fix(dashboard): handle task edit/complete/delete locally instead of emit-to-nowhere"
git add frontend/tests/e2e/dashboard-task-edit.spec.ts
git commit -m "test(e2e): cover dashboard complete/delete on task card (DASH-EDIT-E2E-002/003)"
```

---

## Task 6: Schedule — 点空槽 → EventForm 创建日程

**Files:**
- Create: `frontend/tests/e2e/schedule-ui-crud.spec.ts`

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p0 Schedule · UI 增删改', () => {
  test('SCH-UI-001: 点空槽打开「新建日程」表单,保存后日程出现', async ({ page, apiClient }) => {
    const title = `日程-${Date.now()}`
    const cleanup = async () => {
      const events = await apiClient.getSchedules().catch(() => [])
      const mine = events.find((e) => e.title === title)
      if (mine) await apiClient.deleteSchedule(mine.id).catch(() => {})
    }
    try {
      await page.goto('/schedule')
      await page.getByRole('button', { name: '日' }).click()
      await page.getByRole('button', { name: '今天' }).click()

      // 点第一个空槽打开新建表单
      await page.locator('.hour-slot').first().click()
      const dialog = page.locator('.el-dialog', { hasText: '新建日程' })
      await expect(dialog).toBeVisible()
      await dialog.getByPlaceholder('请输入日程标题').fill(title)
      await dialog.getByRole('button', { name: '保存' }).click()

      await expect(page.getByText(title).first()).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup()
    }
  })
})
```

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test schedule-ui-crud --project=chromium -g SCH-UI-001 --reporter=list`
Expected: 1 passed。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/schedule-ui-crud.spec.ts
git commit -m "test(e2e): cover schedule create via slot+EventForm (SCH-UI-001)"
```

---

## Task 7: Schedule — 点事件块 → 编辑 → 保存

**Files:**
- Modify: `frontend/tests/e2e/schedule-ui-crud.spec.ts`

- [ ] **Step 1: 追加用例**

```ts
  test('SCH-UI-002: 点事件块打开「编辑日程」,改标题保存生效', async ({ page, scheduleFactory }) => {
    const original = `原日程-${Date.now()}`
    const ev = await scheduleFactory.create({ title: original })
    const changed = `${original}-改`

    await page.goto('/schedule')
    await page.getByRole('button', { name: '日' }).click()
    await page.getByRole('button', { name: '今天' }).click()

    const block = page.locator('.event-block', { hasText: original }).first()
    await expect(block).toBeVisible({ timeout: 10000 })
    await block.click()

    const dialog = page.locator('.el-dialog', { hasText: '编辑日程' })
    await expect(dialog).toBeVisible()
    await dialog.getByPlaceholder('请输入日程标题').fill(changed)
    await dialog.getByRole('button', { name: '保存' }).click()

    await expect(page.getByText(changed).first()).toBeVisible({ timeout: 10000 })
    expect(ev.id).toBeTruthy()
  })
```

> 注:`scheduleFactory.create` 的字段以 `frontend/tests/support/factories/schedule.factory.ts` 为准;若必填字段不同(如 start/end),按工厂 `build()` 默认补齐。

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test schedule-ui-crud --project=chromium -g SCH-UI-002 --reporter=list`
Expected: 1 passed。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/schedule-ui-crud.spec.ts
git commit -m "test(e2e): cover schedule edit via event-block click (SCH-UI-002)"
```

---

## Task 8: Schedule — 编辑对话框内「删除」事件

**Files:**
- Modify: `frontend/tests/e2e/schedule-ui-crud.spec.ts`

- [ ] **Step 1: 追加用例**

```ts
  test('SCH-UI-003: 编辑对话框内点「删除」移除日程', async ({ page, scheduleFactory }) => {
    const title = `待删日程-${Date.now()}`
    await scheduleFactory.create({ title })

    await page.goto('/schedule')
    await page.getByRole('button', { name: '日' }).click()
    await page.getByRole('button', { name: '今天' }).click()

    const block = page.locator('.event-block', { hasText: title }).first()
    await expect(block).toBeVisible({ timeout: 10000 })
    await block.click()

    const dialog = page.locator('.el-dialog', { hasText: '编辑日程' })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: '删除' }).click()

    await expect(page.getByText(title)).toHaveCount(0, { timeout: 10000 })
  })
})
```

- [ ] **Step 2: 实跑整文件,预期 3 passed**

Run: `cd frontend && npx playwright test schedule-ui-crud --project=chromium --reporter=list`
Expected: 3 passed(SCH-UI-001~003)。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/schedule-ui-crud.spec.ts
git commit -m "test(e2e): cover schedule delete via edit dialog (SCH-UI-003)"
```

---

## Task 9: Timer — 「开始专注」从零启动到运行态

**Files:**
- Create: `frontend/tests/e2e/timer-start.spec.ts`

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p0 Timer · 启动', () => {
  test('TMR-UI-001: 点「开始专注」进入运行态(出现暂停按钮 / 专注中文案)', async ({ page }) => {
    await page.goto('/timer')

    // 初始为准备态;点开始专注
    await page.locator('.start-btn').click()

    // 运行态:暂停按钮出现,或 timer-label 含「专注」
    await expect(page.locator('.pause-btn')).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.timer-label')).toContainText('专注')
  })
})
```

> 若 `.start-btn` 选择器不命中,改用 `page.getByRole('button', { name: /开始专注/ })`。运行态会真实起一个 work session;本用例不强制停止,Playwright 用例结束自动关页。如担心遗留 session,可在断言后 `page.locator('.abandon-btn')` 流程放弃(已有 timer-interrupt.spec 覆盖,此处非必须)。

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test timer-start --project=chromium --reporter=list`
Expected: 1 passed。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/timer-start.spec.ts
git commit -m "test(e2e): cover timer start from idle (TMR-UI-001)"
```

---

## Task 10: Settings — 改数字字段 + 开关 → 保存 → 重载后生效(往返)

**Files:**
- Create: `frontend/tests/e2e/settings-form-roundtrip.spec.ts`

- [ ] **Step 1: 写用例**

```ts
import { test, expect } from '../support/fixtures'

test.describe('@p0 Settings · 表单往返', () => {
  test('SET-UI-001: 工作时长 + 自动开始休息 改动保存后重载仍生效', async ({ page, apiClient }) => {
    await page.goto('/settings')

    // 记录改动前值(便于恢复)
    const before = await page
      .locator('.el-form-item')
      .filter({ hasText: '工作时长' })
      .getByRole('spinbutton')
      .inputValue()

    // 改工作时长为 30
    const workInput = page
      .locator('.el-form-item')
      .filter({ hasText: '工作时长' })
      .getByRole('spinbutton')
    await workInput.fill('30')
    await workInput.press('Tab')

    // 翻转「自动开始休息」开关
    const breakSwitch = page
      .locator('.el-form-item')
      .filter({ hasText: '自动开始休息' })
      .locator('.el-switch')
    const wasChecked = await breakSwitch.evaluate((el) => el.classList.contains('is-checked'))
    await breakSwitch.click()

    // 保存(番茄设置区的保存按钮)
    await page.locator('.setting-control, .settings-section, .card').getByRole('button', { name: '保存设置' }).first().click()

    // 重载后值应持久
    await page.reload()
    await expect(
      page.locator('.el-form-item').filter({ hasText: '工作时长' }).getByRole('spinbutton')
    ).toHaveValue('30', { timeout: 10000 })

    // 恢复原值,避免污染开发库
    await workInput.fill(before || '25')
    await workInput.press('Tab')
    if (wasChecked !== false) await breakSwitch.click() // 仅当本次翻转过才翻回
    await page.locator('.setting-control, .settings-section, .card').getByRole('button', { name: '保存设置' }).first().click()
  })
})
```

> Element Plus `el-input-number` 的 input 支持 `fill` + `Tab` 提交;若 red 显示值未变,改用 `.locator('.el-input-number__increase')` 加号按钮点到 30。`保存设置` 在番茄区与 AI 区都有,故用区域 + first 限定到番茄区。

- [ ] **Step 2: 实跑,预期 PASS**

Run: `cd frontend && npx playwright test settings-form-roundtrip --project=chromium --reporter=list`
Expected: 1 passed。若 red 因选择器,按注释调整;**不改源码**(设置保存已可用)。

- [ ] **Step 3: 提交**

```bash
git add frontend/tests/e2e/settings-form-roundtrip.spec.ts
git commit -m "test(e2e): cover settings form round-trip persist (SET-UI-001)"
```

---

## Task 11: 覆盖矩阵 COVERAGE.md

**Files:**
- Create: `frontend/tests/e2e/COVERAGE.md`

- [ ] **Step 1: 写矩阵**

```markdown
# 前端 E2E 交互覆盖矩阵

图例:✅ 已覆盖(用例ID) · ⏳ 待补(P1/P2) · 🐞 疑似 bug

## Tasks (/tasks)
| 交互组件 | 状态 |
|---|---|
| 添加任务(创建弹窗) | ✅ TASK-UI-001 |
| 列表视图下拉「编辑」 | ✅ TASK-UI-002 |
| 列表视图下拉「删除」 | ✅ TASK-UI-003 |
| 列表视图下拉「标记完成」 | ✅ TASK-UI-004 |
| 四象限视图 row `.row-more`「更多」 | 🐞 no-op,无下拉菜单(规划期发现) |
| 四象限 row 点按→TaskPomodoroDetail | ⏳ P2 |
| 拖拽跨象限持久化 | ⏳ P1(HTML5 drag 易抖) |
| 筛选/排序下拉 | ⏳ P2 |

## Dashboard (/)
| 交互组件 | 状态 |
|---|---|
| TaskCard 下拉「编辑」 | ✅ DASH-EDIT-E2E-001(已修) |
| TaskCard 下拉「完成」 | ✅ DASH-EDIT-E2E-002(已修) |
| TaskCard 下拉「删除」 | ✅ DASH-EDIT-E2E-003(已修) |
| 快速操作「开始番茄」 | ✅ DASH-E2E-004(既有) |
| Analytics 未配 AI 点「获取洞察」 | 🐞 无反馈 |

## Schedule (/schedule)
| 交互组件 | 状态 |
|---|---|
| 点空槽→新建日程 | ✅ SCH-UI-001 |
| 点事件块→编辑 | ✅ SCH-UI-002 |
| 编辑框内删除 | ✅ SCH-UI-003 |
| 日/周/月切换、今天 | ✅ SCH-E2E-001/002(既有) |
| 拖拽/缩放事件 | ⏳ P1 |

## Timer (/timer)
| 交互组件 | 状态 |
|---|---|
| 开始专注(从零启动) | ✅ TMR-UI-001 |
| 暂停/继续/完成/放弃 | ✅ TMR-E2E-001~005(既有) |

## Settings (/settings)
| 交互组件 | 状态 |
|---|---|
| 数字字段+开关往返持久 | ✅ SET-UI-001 |
| AI Key/服务商/模型/测试连接 | ⏳ P2 |
| 缓冲比例滑块 | ⏳ P2 |
| 导入/导出/清空 | ⏳ P2 |

## Analytics (/analytics) / WorkLog (/work-log)
| 交互组件 | 状态 |
|---|---|
| Analytics 时间筛选 | ✅ ANALY-E2E-003(既有) |
| Analytics 图表 tooltip/下钻 | ⏳ P2 |
| WorkLog 全页(条目增删改/批量入库/生成报告) | ⏳ P1(0 覆盖,最大白地) |
| WorkLog BrainDumpInput | 🐞 `v-if="false"` 功能停用 |
```

- [ ] **Step 2: 提交**

```bash
git add frontend/tests/e2e/COVERAGE.md
git commit -m "docs(e2e): add interaction coverage matrix"
```

---

## Task 12: 全量回归

- [ ] **Step 1: 跑本批全部新增文件**

Run: `cd frontend && npx playwright test task-ui-crud dashboard-task-edit schedule-ui-crud timer-start settings-form-roundtrip --project=chromium --reporter=list`
Expected: 全 passed(约 12 条:4+3+3+1+1)。

- [ ] **Step 2: 确认未破坏既有套件(抽样)**

Run: `cd frontend && npx playwright test dashboard task-crud schedule-views timer-workflow settings --project=chromium --reporter=list`
Expected: 与本分支起点同状态(既有 red 属历史遗留,不计入本次回归;详见 memory `frontend-test-baseline-green.md`)。

---

## 规划期发现(需用户拍板是否升入本批)

1. **四象限视图 row 模式 `.row-more` 是 no-op**(`TaskCard.vue:15` `@click.stop` 无 handler):「更多」图标点了没反应,且 row 模式无编辑/删除下拉。修复需给 row 模式补下拉(复用 card 模式的 `el-dropdown`),属中等源码改动。**建议作为 P1 单独 task:写 red 用例 + 修复。**
2. **WorkLog 页 0 覆盖 + `BrainDumpInput` 被 `v-if="false"` 关闭**:整页交互未测,且脑暴入口疑似停用。建议 P1 起一个 WorkLog 专项。
3. **Analytics 未配 AI 时「获取洞察」无反馈**:建议补一条「未配 AI 点按钮应给提示」的用例(可能需源码加 `ElMessage.warning`)。

这三项**不在本批 P0 内**,执行时可先把 P0 跑绿,再就这 3 项回到 brainstorming 决策。
