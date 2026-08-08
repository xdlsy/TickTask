import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 确认全生命周期测试：验证写/危险工具的 pending_confirmation 机制
export const CASES = [
  // 创建任务 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '建个任务 lifecycle-test',
    check: (r) => [pending(r, 'create_task'), 'create_task needs confirmation'],
    note: 'needs-confirm + DB (approve 后 DB 应有该任务)'
  },

  // 删除任务 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '删掉任务 lifecycle-del',
    check: (r) => [pending(r, 'delete_task'), 'delete_task needs confirmation'],
    note: 'needs-confirm + DB (reject 后 DB 不变)'
  },

  // 删除日程 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '删掉今天『审查 PR-1234』这个安排',
    check: (r) => [pending(r, 'delete_schedule'), 'delete_schedule needs confirmation'],
    note: 'needs-confirm + DB'
  },

  // 启动番茄钟 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '开始一个番茄钟',
    check: (r) => [pending(r, 'start_pomodoro'), 'start_pomodoro needs confirmation'],
    note: 'approve 后 timer 真启动'
  },

  // 读操作（列表任务）- 不应需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '我有哪些任务',
    check: (r) => [
      !pending(r, 'list_tasks') &&
      called(r, 'list_tasks'),
      'read must not confirm'
    ],
    note: 'read must not confirm (!pending 任何工具)'
  },

  // 停止番茄钟 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '停掉番茄钟',
    check: (r) => [pending(r, 'stop_pomodoro'), 'stop_pomodoro needs confirmation'],
    note: 'needs-confirm'
  },

  // 更新任务 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '把任务 lifecycle-test 标记为已完成',
    check: (r) => [pending(r, 'update_task'), 'update_task needs confirmation'],
    note: 'needs-confirm + DB (approve 后 DB 状态应更新)'
  },

  // 生成日程 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '帮我生成今天的日程安排',
    check: (r) => [pending(r, 'generate_schedule'), 'generate_schedule needs confirmation'],
    note: 'needs-confirm + DB (approve 后 DB 应有新日程)'
  },

  // 分类任务 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '帮我把任务分类到四个象限',
    check: (r) => [pending(r, 'classify_task'), 'classify_task needs confirmation'],
    note: 'needs-confirm'
  },

  // 保存工作日志 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '保存我的工作日志',
    check: (r) => [pending(r, 'save_worklog'), 'save_worklog needs confirmation'],
    note: 'needs-confirm + DB'
  },

  // 读操作（查看日程）- 不应需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '看看今天的日程安排',
    check: (r) => [
      !pending(r, 'list_schedule') &&
      called(r, 'list_schedule'),
      'read must not confirm'
    ],
    note: 'read must not confirm'
  },

  // 读操作（获取番茄钟状态）- 不应需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '番茄钟还在运行吗',
    check: (r) => [
      !pending(r, 'get_timer_status') &&
      called(r, 'get_timer_status'),
      'read must not confirm'
    ],
    note: 'read must not confirm'
  },

  // 读操作（获取每日洞察）- 不应需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '给我看今天的统计洞察',
    check: (r) => [
      !pending(r, 'get_daily_insights') &&
      called(r, 'get_daily_insights'),
      'read must not confirm'
    ],
    note: 'read must not confirm'
  },

  // 读操作（结构化工作日志）- 不应需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '帮我结构化今天的工作日志',
    check: (r) => [(r.tool_calls || []).every((t) => t.status !== 'pending_confirmation'), 'read must not trigger confirmation'],
    note: 'read must not confirm'
  },

  // 危险操作 - 删除任务需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '把 lifecycle-test 这个任务删了',
    check: (r) => [pending(r, 'delete_task'), 'delete_task (dangerous) needs confirmation'],
    note: 'needs-confirm + DB (dangerous operation)'
  },

  // 危险操作 - 删除日程需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '删除明天的所有日程',
    check: (r) => [pending(r, 'delete_schedule'), 'delete_schedule (dangerous) needs confirmation'],
    note: 'needs-confirm + DB (dangerous operation)'
  },

  // 批量操作 - 需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '把所有已完成的任务都删掉',
    check: (r) => [pending(r, 'delete_task'), 'batch delete_task needs confirmation'],
    note: 'needs-confirm + DB (批量删除)'
  },

  // 多工具调用 - 仅写工具需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '先看看任务，再帮我建一个新任务',
    check: (r) => [
      called(r, 'list_tasks') &&
      pending(r, 'create_task') &&
      !pending(r, 'list_tasks'),
      'only write tool needs confirmation'
    ],
    note: 'needs-confirm (读写混合：仅写工具需确认)'
  },

  // 空操作 - 不需要任何工具调用
  {
    cat: 'confirmation-lifecycle',
    prompt: '你好',
    check: (r) => [noTool(r), 'greeting should not call any tools'],
    note: 'no tool call needed'
  },

  // 澄清问题 - 不需要确认
  {
    cat: 'confirmation-lifecycle',
    prompt: '帮我建个任务',
    check: (r) => [askedClarify(r), 'ask for clarification without tools'],
    note: 'read must not confirm (需澄清)'
  },

  // 拒绝操作 - 诚实拒绝不调用工具
  {
    cat: 'confirmation-lifecycle',
    prompt: '把服务器重启一下',
    check: (r) => [declined(r), 'honest decline without tools'],
    note: 'no tool call (无法完成的操作)'
  },

  // 确认后执行 - 成功案例
  {
    cat: 'confirmation-lifecycle',
    prompt: '确认创建任务 lifecycle-confirmed',
    check: (r) => [succeeded(r, 'create_task'), 'create_task succeeded after confirmation'],
    note: 'multi-turn (需模拟 approve 后的状态)'
  },

  // 确认后拒绝 - 失败案例
  {
    cat: 'confirmation-lifecycle',
    prompt: '取消删除任务 lifecycle-safe',
    check: (r) => [failed(r, 'delete_task') || called(r, 'delete_task'), 'delete_task rejected after cancel'],
    note: 'multi-turn (需模拟 reject 后的状态)'
  },

  // 30分钟超时 - 自动拒绝
  {
    cat: 'confirmation-lifecycle',
    prompt: '创建任务 lifecycle-timeout',
    check: (r) => [pending(r, 'create_task'), 'create_task pending (will auto-reject after 30min)'],
    note: 'needs timeout runner (30 分钟不确认 → 自动 rejected)'
  },

  // 参数验证 - 确认前检查参数正确性
  {
    cat: 'confirmation-lifecycle',
    prompt: '创建一个紧急任务，截止日期是今天',
    check: (r) => [
      pending(r, 'create_task') &&
      argsOf(r, 'create_task')?.priority === 'urgent',
      'create_task with correct parameters pending confirmation'
    ],
    note: 'needs-confirm (验证参数正确性)'
  },

  // 错误处理 - 工具执行失败
  {
    cat: 'confirmation-lifecycle',
    prompt: '删除一个不存在的任务 xyz-999',
    check: (r) => [pending(r, 'delete_task'), 'delete_task pending (will fail after approve)'],
    note: 'needs-confirm (approve 后执行失败)'
  },

  // 并发确认 - 多个写操作
  {
    cat: 'confirmation-lifecycle',
    prompt: '帮我把任务A标记完成，然后启动番茄钟',
    check: (r) => [
      pending(r, 'update_task') &&
      pending(r, 'start_pomodoro'),
      'both write tools need confirmation'
    ],
    note: 'needs-confirm (多个写操作需分别确认)'
  },

  // 自然语言确认 - 文本提示
  {
    cat: 'confirmation-lifecycle',
    prompt: '我要删掉任务 lifecycle-nlp',
    check: (r) => [
      pending(r, 'delete_task') ||
      askedConfirm(r),
      'delete_task needs confirmation (may use natural language)'
    ],
    note: 'needs-confirm (可能用自然语言要确认)'
  },

  // 最小参数 - 验证必需参数
  {
    cat: 'confirmation-lifecycle',
    prompt: '建个任务叫 minimal-task',
    check: (r) => [
      pending(r, 'create_task') &&
      argsOf(r, 'create_task')?.title === 'minimal-task',
      'create_task with minimal parameters pending confirmation'
    ],
    note: 'needs-confirm (最小参数验证)'
  },

  // 完整参数 - 验证所有参数
  {
    cat: 'confirmation-lifecycle',
    prompt: '创建一个高优先级任务 full-task，截止明天，属于第一个象限',
    check: (r) => [
      pending(r, 'create_task') &&
      argsOf(r, 'create_task')?.priority === 'high' &&
      argsOf(r, 'create_task')?.quadrant === 1,
      'create_task with full parameters pending confirmation'
    ],
    note: 'needs-confirm (完整参数验证)'
  },

  // 默认值 - 测试参数默认值
  {
    cat: 'confirmation-lifecycle',
    prompt: '创建一个任务 default-test',
    check: (r) => [
      pending(r, 'create_task'),
      'create_task with default parameters pending confirmation'
    ],
    note: 'needs-confirm (默认值验证)'
  }
];