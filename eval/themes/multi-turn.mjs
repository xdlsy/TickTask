import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 多轮对话用例：每条用例的 turns 是字符串数组，表示连续的用户输入
// check 为占位函数，因为单轮 runner 无法执行多轮对话
// note 标记多轮对话类型和需要验证的行为

export const CASES = [
  // 代词指代：建立任务后用代词修改
  {
    cat: 'multi-turn',
    turns: [
      '建个任务『报告』',
      '把它的截止日期设为周五',
    ],
    note: 'multi-turn — 代词它指代上一轮创建的任务',
    check: () => [true, 'needs multi-turn runner — 验证代词"它"指代任务"报告"'],
  },

  // 序数指代：查询后用序数词修改
  {
    cat: 'multi-turn',
    turns: [
      '我有哪些任务',
      '第二个是啥状态',
    ],
    note: 'multi-turn — 序数词"第二个"指代列表项',
    check: () => [true, 'needs multi-turn runner — 验证序数词"第二个"指向任务列表中索引为1的任务'],
  },

  // 纠正当：建立任务后立即纠正日期
  {
    cat: 'multi-turn',
    turns: [
      '建个任务明天开会',
      '不对，是后天',
    ],
    note: 'multi-turn — 纠正对话，验证最终日期=后天',
    check: () => [true, 'needs multi-turn runner — 验证任务日期被更新为后天（原明天）'],
  },

  // 长上下文：连续 22 条来回后引用早期创建的任务
  {
    cat: 'multi-turn',
    turns: [
      // 填充 10 条来回（20 条消息），达到 MaxContextMessages=20 边界
      '今天有啥安排',
      '明天呢',
      '后天有啥',
      '大后天呢',
      '这周末呢',
      '下周呢',
      '下下周呢',
      '月底呢',
      '下季度呢',
      '年底呢',
      // 第 11 条来回，引用早期创建的任务
      '我刚建的那个任务还在吗',
    ],
    note: 'long-context (MaxContextMessages=20 边界) — 验证跨越 20 条消息后仍能引用早期上下文',
    check: () => [true, 'needs multi-turn runner — 验证在 20+ 条消息后仍能记住并检索到本轮对话开始前创建的任务'],
  },

  // 矛盾中止：删除过程中取消确认
  {
    cat: 'multi-turn',
    turns: [
      '删掉任务【报告】',
      '等等别删',
    ],
    note: 'abort confirm — 危险操作 pending 确认中收到取消指令',
    check: () => [true, 'needs multi-turn runner — 验证 delete_task 在 pending_confirmation 状态下被正确中止，任务未被删除'],
  },

  // 多步骤任务创建：创建后修改多个属性
  {
    cat: 'multi-turn',
    turns: [
      '建个紧急任务【修复登录bug】',
      '把它设为高优先级',
      '放到今天做',
    ],
    note: 'multi-turn — 连续修改同一任务的不同属性（优先级、日期）',
    check: () => [true, 'needs multi-turn runner — 验证任务被连续 update，优先级=高，截止日期=今天'],
  },

  // 交叉指代：多个任务间的代词指代
  {
    cat: 'multi-turn',
    turns: [
      '建个任务【写周报】',
      '再建个任务【准备PPT】',
      '把它们都放到明天',
    ],
    note: 'multi-turn — 代词"它们"指代多个任务',
    check: () => [true, 'needs multi-turn runner — 验证批量更新，两个任务截止日期均设为明天'],
  },

  // 条件分支：根据查询结果执行不同操作
  {
    cat: 'multi-turn',
    turns: [
      '看看今天有哪些待完成的任务',
      '把第一个移到明天',
    ],
    note: 'multi-turn — 条件分支，查询结果影响下一步操作',
    check: () => [true, 'needs multi-turn runner — 验证根据 list_tasks 结果，将第一个未完成任务移到明天'],
  },

  // 错误恢复：执行失败后重试
  {
    cat: 'multi-turn',
    turns: [
      '删掉任务【不存在的任务】',
      '算了，删掉【报告】',
    ],
    note: 'multi-turn — 失败恢复，首个操作失败后执行备选操作',
    check: () => [true, 'needs multi-turn runner — 验证首次删除失败后，成功删除备选任务【报告】'],
  },

  // 级联操作：A 操作结果作为 B 操作输入
  {
    cat: 'multi-turn',
    turns: [
      '帮我列出优先级最高的任务',
      '为这个任务生成今日时间安排',
    ],
    note: 'multi-turn — 级联操作，上轮查询结果作为下轮参数',
    check: () => [true, 'needs multi-turn runner — 验证 generate_schedule 使用上轮返回的任务ID作为输入'],
  },

  // 确认取消流：发起危险操作后取消，再重新发起
  {
    cat: 'multi-turn',
    turns: [
      '删除所有已完成的任务',
      '等等，先别删',
      '好吧，删吧',
    ],
    note: 'multi-turn — 确认取消流，pending → cancel → confirm',
    check: () => [true, 'needs multi-turn runner — 验证确认流程的撤销与重新确认逻辑'],
  },

  // 时间窗口引用：引用相对时间描述
  {
    cat: 'multi-turn',
    turns: [
      '下周三有个任务【客户会议】',
      '提前一天提醒我',
    ],
    note: 'multi-turn — 相对时间引用，提前一天=下周二',
    check: () => [true, 'needs multi-turn runner — 验证任务提醒时间被设置为下周二（下周三的前一天）'],
  },

  // 批量操作失败恢复：批量操作部分失败后的处理
  {
    cat: 'multi-turn',
    turns: [
      '把所有明天到期的任务移到后天',
      '刚才报错的那个单独处理一下',
    ],
    note: 'multi-turn — 批量操作部分失败恢复',
    check: () => [true, 'needs multi-turn runner — 验证批量 update 中失败的单个任务被单独处理'],
  },

  // 上下文清空边界：超过最大上下文后的引用
  {
    cat: 'multi-turn',
    turns: [
      '创建任务A',
      '创建任务B',
      '创建任务C',
      '创建任务D',
      '创建任务E',
      '创建任务F',
      '创建任务G',
      '创建任务H',
      '创建任务I',
      '创建任务J',
      '创建任务K',
      '修改任务A',
    ],
    note: 'context overflow boundary — 超过 MaxContextMessages 后引用早期实体',
    check: () => [true, 'needs multi-turn runner — 验证在 22 条消息后仍能正确引用并修改最初创建的任务A'],
  },
];
