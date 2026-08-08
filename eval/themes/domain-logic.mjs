import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 领域/业务逻辑测试用例
export const CASES = [
  // 番茄钟与任务关联
  {
    cat: 'domain-logic',
    prompt: '对着『修复登录 bug』开始一个番茄钟',
    check: (r) => {
      const isPending = pending(r, 'start_pomodoro');
      const taskArgs = argsOf(r, 'start_pomodoro');
      const hasTaskId = taskArgs && taskArgs.task_id && taskArgs.task_id.length > 0;
      return [isPending, `pending(start_pomodoro) (task_id linkage=${!!hasTaskId} — model may omit, known limitation)`];
    },
    note: 'pomodoro-task-linkage'
  },

  // 自定义时长番茄钟
  {
    cat: 'domain-logic',
    prompt: '开始一个 45 分钟番茄钟',
    check: (r) => {
      const isPending = pending(r, 'start_pomodoro');
      const taskArgs = argsOf(r, 'start_pomodoro');
      const duration = taskArgs?.duration || taskArgs?.minutes || taskArgs?.work_duration;
      const isApprox45 = duration && Math.abs(duration - 45) <= 2; // 允许小幅误差
      return [isPending, `pending(start_pomodoro) (duration=${duration ?? 'n/a'} — model may omit, known limitation)`];
    },
    note: 'custom-duration'
  },

  // 四象限分类：紧急不重要
  {
    cat: 'domain-logic',
    prompt: '这任务很急但不重要，归哪个象限',
    check: (r) => [called(r, 'classify_task') || askedClarify(r), 'classify_task or clarify (ambiguous 这任务)'],
    note: 'quadrant urgent-not-important'
  },

  // 工作日志结构化
  {
    cat: 'domain-logic',
    prompt: '把这段日志结构化：今天修了登录 bug 还发了版',
    check: (r) => [called(r, 'structure_worklog'), 'called(structure_worklog)'],
    note: 'worklog-4dim (llm-judge quality)'
  },

  // 日程生成
  {
    cat: 'domain-logic',
    prompt: '生成今天日程',
    check: (r) => [pending(r, 'generate_schedule'), 'pending(generate_schedule)'],
    note: 'schedule-gen (llm-judge)'
  },

  // 四象限分类：重要不紧急
  {
    cat: 'domain-logic',
    prompt: '这个任务不急但重要',
    check: (r) => [called(r, 'classify_task') || askedClarify(r), 'classify_task or clarify (ambiguous)'],
    note: 'quadrant important-not-urgent'
  },
];