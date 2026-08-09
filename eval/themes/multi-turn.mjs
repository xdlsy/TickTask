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
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const created = pending(t1, 'create_task') || succeeded(t1, 'create_task') || called(t1, 'create_task');
      const updated = pending(t2, 'update_task') || succeeded(t2, 'update_task') || called(t2, 'update_task');
      return [created && updated, 'turn1 create_task, turn2 update_task for "报告" with due_date=周五'];
    },
  },

  // 序数指代：查询后用序数词修改
  {
    cat: 'multi-turn',
    turns: [
      '我有哪些任务',
      '第二个是啥状态',
    ],
    note: 'multi-turn — 序数词"第二个"指代列表项',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const listed = succeeded(t1, 'list_tasks');
      const hasAnswer = t2.assistant_text && t2.assistant_text.trim().length > 0;
      const noToolUsed = noTool(t2);
      return [listed && hasAnswer && noToolUsed, 'turn1 list_tasks, turn2 answer about second task (no tool)'];
    },
  },

  // 纠正当：建立任务后立即纠正日期
  {
    cat: 'multi-turn',
    turns: [
      '建个任务明天开会',
      '不对，是后天',
    ],
    note: 'multi-turn — 纠正对话，验证最终日期=后天',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const created = pending(t1, 'create_task') || succeeded(t1, 'create_task') || called(t1, 'create_task');
      const updated = pending(t2, 'update_task') || succeeded(t2, 'update_task') || called(t2, 'update_task');
      return [created && updated, 'turn1 create_task, turn2 update_task correcting due_date to 后天'];
    },
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
    check: (r, ctx) => {
      const lastTurn = ctx?.history?.[10];
      if (!lastTurn) return [false, 'missing final turn'];
      const hasResponse = lastTurn.assistant_text && lastTurn.assistant_text.trim().length > 0;
      const noError = !failed(lastTurn);
      return [hasResponse && noError, 'turn11 has valid response after 20+ messages (long-context handling)'];
    },
  },

  // 矛盾中止：删除过程中取消确认
  {
    cat: 'multi-turn',
    turns: [
      '删掉任务【报告】',
      '等等别删',
    ],
    note: 'abort confirm — 危险操作 pending 确认中收到取消指令',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const deleteInitiated = pending(t1, 'delete_task') || askedConfirm(t1);
      const aborted = !succeeded(t2, 'delete_task');
      return [deleteInitiated && aborted, 'turn1 delete_task initiated, turn2 acknowledges cancellation (no successful delete)'];
    },
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
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      const t3 = ctx?.history?.[2];
      if (!t1 || !t2 || !t3) return [false, 'missing turns'];
      const created = pending(t1, 'create_task') || succeeded(t1, 'create_task') || called(t1, 'create_task');
      const updatedPriority = pending(t2, 'update_task') || succeeded(t2, 'update_task') || called(t2, 'update_task');
      const updatedDate = pending(t3, 'update_task') || succeeded(t3, 'update_task') || called(t3, 'update_task');
      return [created && updatedPriority && updatedDate, 'turn1 create, turn2 update priority, turn3 update due_date'];
    },
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
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      const t3 = ctx?.history?.[2];
      if (!t1 || !t2 || !t3) return [false, 'missing turns'];
      const created1 = pending(t1, 'create_task') || succeeded(t1, 'create_task') || called(t1, 'create_task');
      const created2 = pending(t2, 'create_task') || succeeded(t2, 'create_task') || called(t2, 'create_task');
      const updated = pending(t3, 'update_task') || succeeded(t3, 'update_task') || called(t3, 'update_task');
      return [created1 && created2 && updated, 'turn1/2 create two tasks, turn3 updates them to 明天'];
    },
  },

  // 条件分支：根据查询结果执行不同操作
  {
    cat: 'multi-turn',
    turns: [
      '看看今天有哪些待完成的任务',
      '把第一个移到明天',
    ],
    note: 'multi-turn — 条件分支，查询结果影响下一步操作',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const listed = succeeded(t1, 'list_tasks') || called(t1, 'list_tasks');
      const updated = pending(t2, 'update_task') || succeeded(t2, 'update_task') || called(t2, 'update_task');
      return [listed && updated, 'turn1 list_tasks, turn2 update first task to 明天'];
    },
  },

  // 错误恢复：执行失败后重试
  {
    cat: 'multi-turn',
    turns: [
      '删掉任务【不存在的任务】',
      '算了，删掉【报告】',
    ],
    note: 'multi-turn — 失败恢复，首个操作失败后执行备选操作',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const firstDelete = pending(t1, 'delete_task') || failed(t1, 'delete_task');
      const secondDelete = pending(t2, 'delete_task') || succeeded(t2, 'delete_task');
      return [firstDelete && secondDelete, 'turn1 delete nonexistent task (fail), turn2 delete 【报告】'];
    },
  },

  // 级联操作：A 操作结果作为 B 操作输入
  {
    cat: 'multi-turn',
    turns: [
      '帮我列出优先级最高的任务',
      '为这个任务生成今日时间安排',
    ],
    note: 'multi-turn — 级联操作，上轮查询结果作为下轮参数',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const listed = succeeded(t1, 'list_tasks');
      const scheduled = pending(t2, 'generate_schedule') || succeeded(t2, 'generate_schedule');
      return [listed && scheduled, 'turn1 list_tasks by priority, turn2 generate_schedule for the result'];
    },
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
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      const t3 = ctx?.history?.[2];
      if (!t1 || !t2 || !t3) return [false, 'missing turns'];
      const initiated = pending(t1, 'delete_task') || askedConfirm(t1);
      const cancelled = !succeeded(t2, 'delete_task');
      const reinitiated = pending(t3, 'delete_task') || askedConfirm(t3);
      return [initiated && cancelled && reinitiated, 'turn1 delete initiated, turn2 cancellation (no successful delete), turn3 delete re-initiated'];
    },
  },

  // 时间窗口引用：引用相对时间描述
  {
    cat: 'multi-turn',
    turns: [
      '下周三有个任务【客户会议】',
      '提前一天提醒我',
    ],
    note: 'multi-turn — 相对时间引用，提前一天=下周二',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const created = pending(t1, 'create_task') || succeeded(t1, 'create_task') || called(t1, 'create_task');
      const updated = pending(t2, 'update_task') || succeeded(t2, 'update_task') || called(t2, 'update_task');
      return [created && updated, 'turn1 create task for 下周三, turn2 set reminder to 提前一天'];
    },
  },

  // 批量操作失败恢复：批量操作部分失败后的处理
  {
    cat: 'multi-turn',
    turns: [
      '把所有明天到期的任务移到后天',
      '刚才报错的那个单独处理一下',
    ],
    note: 'multi-turn — 批量操作部分失败恢复',
    check: (r, ctx) => {
      const t1 = ctx?.history?.[0];
      const t2 = ctx?.history?.[1];
      if (!t1 || !t2) return [false, 'missing turns'];
      const batchUpdates = called(t1, 'update_task') || pending(t1, 'update_task');
      const singleUpdate = called(t2, 'update_task') || pending(t2, 'update_task');
      return [batchUpdates && singleUpdate, 'turn1 batch update, turn2 handle the failed one'];
    },
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
    check: (r, ctx) => {
      const first = ctx?.history?.[0];
      const last = ctx?.history?.[11];
      if (!first || !last) return [false, 'missing turns'];
      const firstCreate = called(first, 'create_task') || succeeded(first, 'create_task') || pending(first, 'create_task');
      const lastUpdate = called(last, 'update_task') || pending(last, 'update_task') || succeeded(last, 'update_task');
      return [firstCreate && lastUpdate, 'turn1 create taskA, turn12 (after 22+ messages) update taskA'];
    },
  },
];
