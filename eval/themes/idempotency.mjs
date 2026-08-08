import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 幂等性/副作用安全测试用例
export const CASES = [
  // 用例1: 创建同名任务两次（第一次）
  {
    cat: 'idempotency',
    prompt: '建个任务 dup-test',
    check: (r) => [pending(r, 'create_task'), 'first call should pending create_task'],
    note: 'run twice, assert ≤1 created or dedupe'
  },
  // 用例2: 创建同名任务两次（第二次，应该提示已存在或去重）
  {
    cat: 'idempotency',
    prompt: '建个任务 dup-test',
    check: (r) => {
      // 理想情况：askedClarify（说明发现重复）或 pending 计数≤1（只创建一个）
      const clarificationAsked = askedClarify(r);
      const pendingCount = tools(r).filter(t => t.name === 'create_task' && t.status === 'pending_confirmation').length;
      const noDuplication = pendingCount <= 1;

      if (clarificationAsked) {
        return [true, 'asked for clarification about duplicate task'];
      } else if (noDuplication) {
        return [true, 'at most one create_task pending (deduped)'];
      } else {
        return [false, 'should either ask clarification or dedupe, but got multiple pending creates'];
      }
    },
    note: 'run twice, assert ≤1 created or dedupe'
  },

  // 用例3: 开始番茄钟（第一次）
  {
    cat: 'idempotency',
    prompt: '开始番茄钟',
    check: (r) => [pending(r, 'start_pomodoro'), 'first pomodoro should be pending'],
    note: 'conflict - second start should not happen'
  },
  // 用例4: 开始番茄钟（第二次，应该检测到已有运行中的番茄钟）
  {
    cat: 'idempotency',
    prompt: '再开始一个',
    check: (r) => {
      // 第二次不该同时有两个 start，应该：
      // 1. askedClarify（发现冲突）或
      // 2. 先 stop 再 start（pending 包含 stop）或
      // 3. 拒绝第二次 start
      const clarificationAsked = askedClarify(r);
      const hasStopPending = pending(r, 'stop_pomodoro');
      const noDoubleStart = tools(r).filter(t => t.name === 'start_pomodoro' && t.status === 'pending_confirmation').length <= 1;

      if (clarificationAsked) {
        return [true, 'asked for clarification about running pomodoro'];
      } else if (hasStopPending) {
        return [true, 'stops current pomodoro before starting new one'];
      } else if (noDoubleStart) {
        return [true, 'at most one start_pomodoro pending'];
      } else {
        return [false, 'should not have multiple simultaneous start_pomodoro'];
      }
    },
    note: 'conflict - second start should not happen'
  },

  // 用例5: 创建重复任务（相同意图）
  {
    cat: 'idempotency',
    prompt: '建两个一样的任务『dup』',
    check: (r) => {
      // 应该 pending(create_task) 但理想情况是 askedClarify（发现重复意图）或只建一个
      const hasPendingCreate = pending(r, 'create_task');
      const clarificationAsked = askedClarify(r);
      const pendingCount = tools(r).filter(t => t.name === 'create_task' && t.status === 'pending_confirmation').length;

      if (clarificationAsked) {
        return [true, 'asked for clarification about duplicate intent'];
      } else if (hasPendingCreate && pendingCount === 1) {
        return [true, 'only one create_task despite duplicate intent request'];
      } else if (hasPendingCreate) {
        return [true, 'has pending create_task (though not ideal for duplicate intent)'];
      } else {
        return [false, 'should have pending create_task for duplicate intent'];
      }
    },
    note: 'duplicate intent'
  },

  // 用例6: 列出今天的安排（第一次）
  {
    cat: 'idempotency',
    prompt: '列出我今天的安排',
    check: (r) => [called(r, 'list_schedule'), 'should call list_schedule'],
    note: 'stable - repeated calls should be consistent'
  },
  // 用例7: 列出今天的安排（第二次，应该是幂等的）
  {
    cat: 'idempotency',
    prompt: '列出我今天的安排',
    check: (r) => {
      // 重复调用应该是稳定的，都应该调用 list_schedule
      const calledSchedule = called(r, 'list_schedule');
      return [calledSchedule, 'repeated call should still use list_schedule'];
    },
    note: 'stable - repeated calls should be consistent'
  },
];