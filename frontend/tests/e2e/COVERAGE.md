# 前端 E2E 交互覆盖矩阵

图例:✅ 已覆盖(用例ID) · ⏳ 待补(P1/P2) · 🐞 疑似/已知 bug · 🛠 已修复

> 本矩阵记录每个页面交互组件的 E2E 覆盖状态。
> P0(2026-08-09):基础 CRUD + Dashboard 修复。P1(2026-08-10):row-more 修复 + WorkLog + toast。
> 计划见 `docs/superpowers/plans/2026-08-09-frontend-e2e-interaction-coverage.md` 与 `2026-08-10-p1-interaction-coverage.md`。

## Tasks (/tasks)
| 交互组件 | 状态 |
|---|---|
| 添加任务 → 创建弹窗 → 保存 | ✅ `task-ui-crud.spec.ts` TASK-UI-001 |
| 列表视图下拉「编辑」改标题 | ✅ TASK-UI-002 |
| 列表视图下拉「删除」 | ✅ TASK-UI-003 |
| 列表视图下拉「标记完成」 | ✅ TASK-UI-004 |
| 四象限视图 row `.row-more`「更多」 | ✅ `task-quadrant-row-menu.spec.ts` ROW-UI-001~003 🛠 |
| 四象限 row 点击 → TaskPomodoroDetail | ⏳ P2 |
| 拖拽任务跨象限持久化 | ⏳ P1(HTML5 drag 易抖) |
| 列表筛选/排序下拉 | ⏳ P2 |

## Dashboard (/)
| 交互组件 | 状态 |
|---|---|
| TaskCard 下拉「编辑」→ 编辑弹窗 | ✅ `dashboard-task-edit.spec.ts` DASH-EDIT-E2E-001 🛠 |
| TaskCard 下拉「完成」(含成功 toast) | ✅ DASH-EDIT-E2E-002 🛠 |
| TaskCard 下拉「删除」 | ✅ DASH-EDIT-E2E-003 🛠 |
| 快速操作「开始番茄」→ 跳转计时器 | ✅ DASH-E2E-004(既有) |

## Schedule (/schedule)
| 交互组件 | 状态 |
|---|---|
| 点空槽 → 新建日程表单 → 保存 | ✅ `schedule-ui-crud.spec.ts` SCH-UI-001 |
| 点事件块 → 编辑 → 保存 | ✅ SCH-UI-002 |
| 编辑框内「删除」 | ✅ SCH-UI-003 |
| 日/周/月切换、今天 | ✅ SCH-E2E-001/002(既有) |
| 拖拽/缩放事件 | ⏳ P1 |

## Timer (/timer)
| 交互组件 | 状态 |
|---|---|
| 「开始专注」从零启动 → 运行态 | ✅ `timer-start.spec.ts` TMR-UI-001 |
| 暂停 / 继续 / 完成 / 放弃 | ✅ TMR-E2E-001~005(既有) |

## Settings (/settings)
| 交互组件 | 状态 |
|---|---|
| 数字字段(工作时长)改动 → 保存 → 重载持久 | ✅ `settings-form-roundtrip.spec.ts` SET-UI-001 |
| 开关(自动开始休息/工作、提示音) | ⏳ P2 |
| 缓冲比例滑块 | ⏳ P2 |
| AI Key / 服务商 / 模型 / 测试连接 | ⏳ P2 |
| 导入 / 导出 / 清空 | ⏳ P2 |

## Analytics (/analytics)
| 交互组件 | 状态 |
|---|---|
| 时间筛选(今日/本周/本月) | ✅ ANALY-E2E-003(既有) |
| 「获取洞察」(AI 卡片) | ℹ️ 按钮在 `v-if=configured` 卡片内,未配 AI 不渲染——非 bug,无需覆盖 |
| 图表 tooltip / 下钻 | ⏳ P2 |

## WorkLog (/work-log)
| 交互组件 | 状态 |
|---|---|
| WorkItemForm 录入条目 | ✅ `work-log-ui.spec.ts` WL-UI-001 |
| TodayPanorama 编辑 / 删除(popconfirm) | ✅ WL-UI-002 / WL-UI-003 |
| 生成「本周周报」→ ReportDetail | ✅ WL-UI-004 |
| BatchTableEditor 批量入库 | ⏳ P2 |
| BrainDumpInput 脑暴入口 | 🐞 `v-if="false"` 功能停用(有意禁用,暂不测) |

---

## P0(2026-08-09)修复的真实 bug
| Bug | 根因 | 修复 |
|---|---|---|
| 任意任务编辑 → 400 | 后端 `encodeTags/decodeTags` 是空 stub;`TaskResponse.tags` 返回 string 与 `UpdateTaskInput.tags`(`[]string`)冲突 | `backend/internal/service/task_service.go`:JSON 编解码 + `TaskResponse.Tags` 改 `[]string` |
| 仪表盘点「编辑/完成/删除」无反应 | Dashboard(路由页)把 TaskCard 事件 `$emit` 抛给空父级,未渲染 TaskForm | `frontend/src/views/Dashboard.vue`:本地 TaskForm + store handler |

## P1(2026-08-10)修复 / 新增
| 项 | 类型 | 说明 |
|---|---|---|
| 四象限 row `.row-more` 死入口 | 修复 | `TaskCard.vue` row 模式补 `el-dropdown`,复用 `handleCommand`(ROW-UI-001~003) |
| 任务完成/删除无反馈 | 增强 | Dashboard + QuadrantView 加 `ElMessage.success` |
| Analytics「获取洞察」疑似无反馈 | 更正 | 经核实**非 bug**(按钮在 `v-if=configured` 卡片内),从待办移除 |

## 下批(P2)候选
1. 拖拽(任务跨象限、日程挪动/缩放)——单独评估 Playwright HTML5 drag 稳定性。
2. WorkLog BatchTableEditor 批量入库。
3. Settings 其余字段(开关/滑块/AI/导入导出)。
4. 四象限 row 点击 → TaskPomodoroDetail;列表筛选/排序;Analytics 图表下钻。
5. BrainDumpInput(需产品决策是否重新启用 brain-dump→AI 流)。
