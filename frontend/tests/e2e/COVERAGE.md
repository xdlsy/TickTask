# 前端 E2E 交互覆盖矩阵

图例:✅ 已覆盖(用例ID) · ⏳ 待补 · ❌ 不可行(记因) · 🐞 疑似/已知 bug · 🛠 已修复 · ℹ️ 说明

> P0(2026-08-09):基础 CRUD + Dashboard 修复。P1(2026-08-10):row-more + WorkLog + toast。P2(2026-08-13):拖拽 + Settings 全 + 筛选排序 + tooltip + row 详情。
> 计划见 `docs/superpowers/plans/2026-08-{09,10,13}-*.md`。

## Tasks (/tasks)
| 交互组件 | 状态 |
|---|---|
| 添加任务 → 创建弹窗 → 保存 | ✅ `task-ui-crud.spec.ts` TASK-UI-001 |
| 列表视图下拉「编辑」/「删除」/「标记完成」 | ✅ TASK-UI-002/003/004 |
| 四象限视图 row `.row-more`「更多」 | ✅ `task-quadrant-row-menu.spec.ts` ROW-UI-001~003 🛠 |
| 四象限 row 点击 → TaskPomodoroDetail | ✅ `task-row-detail.spec.ts` ROW-UI-004 |
| 拖拽任务跨象限持久化 | ✅ `task-drag.spec.ts` DRAG-UI-001 |
| 列表筛选(状态/象限)/ 排序 | ✅ `task-list-filters.spec.ts` LIST-UI-001/002 |

## Dashboard (/)
| 交互组件 | 状态 |
|---|---|
| TaskCard 下拉「编辑」/「完成」(含 toast)/「删除」 | ✅ `dashboard-task-edit.spec.ts` DASH-EDIT-E2E-001/002/003 🛠 |
| 快速操作「开始番茄」→ 跳转计时器 | ✅ DASH-E2E-004(既有) |

## Schedule (/schedule)
| 交互组件 | 状态 |
|---|---|
| 点空槽 → 新建 / 点事件块 → 编辑 / 对话框删除 | ✅ `schedule-ui-crud.spec.ts` SCH-UI-001/002/003 |
| 日/周/月切换、今天 | ✅ SCH-E2E-001/002(既有) |
| 拖拽/缩放事件 | ❌ 未实现(schedule 组件无 `draggable`/`resizable`) |

## Timer (/timer)
| 交互组件 | 状态 |
|---|---|
| 「开始专注」从零启动 → 运行态 | ✅ `timer-start.spec.ts` TMR-UI-001 |
| 暂停 / 继续 / 完成 / 放弃 | ✅ TMR-E2E-001~005(既有) |

## Settings (/settings)
| 交互组件 | 状态 |
|---|---|
| 工作时长数字字段 → 保存 → 重载持久 | ✅ `settings-form-roundtrip.spec.ts` SET-UI-001 |
| 3 开关 + 缓冲比例滑块 往返持久 | ✅ `settings-toggles.spec.ts` SET-UI-002 |
| 导出全部数据(JSON 下载) | ✅ `settings-data.spec.ts` SET-UI-003 |
| 清空全部数据(确认 → 取消,非破坏) | ✅ SET-UI-004 |
| AI Key / 服务商 / 模型 / 测试连接 | ⏳ P3(需 AI 配置) |
| 导入向导(文件 + 冲突解决) | ⏳ P3(需备份文件) |

## Analytics (/analytics)
| 交互组件 | 状态 |
|---|---|
| 时间筛选(今日/本周/本月) | ✅ ANALY-E2E-003(既有) |
| 图表柱状 tooltip(hover 显时长) | ✅ `analytics-tooltip.spec.ts` ANALY-UI-001 |
| 「获取洞察」(AI 卡片) | ℹ️ 按钮在 `v-if=configured` 卡片内,未配 AI 不渲染——非 bug |
| 图表下钻 | ⏳ P3 |

## WorkLog (/work-log)
| 交互组件 | 状态 |
|---|---|
| WorkItemForm 录入条目 | ✅ `work-log-ui.spec.ts` WL-UI-001 |
| TodayPanorama 编辑 / 删除(popconfirm) | ✅ WL-UI-002 / WL-UI-003 |
| 生成「本周周报」→ ReportDetail | ✅ WL-UI-004 |
| BatchTableEditor 批量入库 | ❌ 不可达(草稿只来自被禁用的 BrainDump) |
| BrainDumpInput 脑暴入口 | 🐞 `v-if="false"` 有意禁用 |

---

## P0(2026-08-09)修复的真实 bug
| Bug | 根因 | 修复 |
|---|---|---|
| 任意任务编辑 → 400 | 后端 `encodeTags/decodeTags` 空 stub;`TaskResponse.tags` 返回 string 与 `UpdateTaskInput.tags`(`[]string`)冲突 | `backend/internal/service/task_service.go`:JSON 编解码 + `TaskResponse.Tags` 改 `[]string` |
| 仪表盘点「编辑/完成/删除」无反应 | Dashboard(路由页)把事件 `$emit` 抛给空父级,未渲染 TaskForm | `frontend/src/views/Dashboard.vue`:本地 TaskForm + store handler |

## P1(2026-08-10)修复 / 新增
| 项 | 类型 | 说明 |
|---|---|---|
| 四象限 row `.row-more` 死入口 | 修复 | `TaskCard.vue` row 模式补 `el-dropdown`,复用 `handleCommand`(ROW-UI-001~003) |
| 任务完成/删除无反馈 | 增强 | Dashboard + QuadrantView 加 `ElMessage.success` |
| Analytics「获取洞察」疑似无反馈 | 更正 | 非 bug(按钮在 `v-if=configured` 卡片内) |

## P2(2026-08-13)新增
| 项 | 说明 |
|---|---|
| 任务拖拽跨象限 | DRAG-UI-001(Playwright `dragTo` 实测可用) |
| Settings 开关+滑块往返 / 导出 / 清空-取消 | SET-UI-002/003/004 |
| 列表筛选(状态/象限)+ 排序 | LIST-UI-001/002 |
| Analytics 图表 tooltip | ANALY-UI-001 |
| 四象限 row 点按 → TaskPomodoroDetail | ROW-UI-004 |

## 下批(P3)候选
1. BrainDumpInput:需产品决策是否重新启用 brain-dump→AI 流(启用后可连带测 BatchTableEditor)。
2. AI 依赖入口:Analytics「获取洞察」、Settings「测试连接」(需配置 AI)。
3. Settings 导入向导(需 JSON 备份文件 + 冲突解决 UI)。
4. Analytics 图表下钻。
5. 日程拖拽/缩放:目前**未实现**,待产品加功能后再补。
