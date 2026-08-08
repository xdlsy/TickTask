import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

export const CASES = [
  // classify_task (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '帮我把『整理周报』归类到象限',
    check: (r) => [called(r, 'classify_task'), 'called classify_task'],
  },
  {
    cat: 'tool-matrix',
    prompt: '这个任务重要不紧急，归哪个象限',
    check: (r) => [called(r, 'classify_task') || askedClarify(r), 'classify_task or clarify (ambiguous 这个任务)'],
  },
  {
    cat: 'tool-matrix',
    prompt: '给『修复登录 bug』分类',
    check: (r) => [called(r, 'classify_task'), 'called classify_task'],
  },

  // stop_pomodoro (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '有番茄钟在跑，停掉它',
    check: (r) => [called(r, 'stop_pomodoro') && pending(r, 'stop_pomodoro'), 'called stop_pomodoro with pending_confirmation'],
    note: 'needs-confirm',
  },

  // generate_schedule (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '帮我生成今天的日程',
    check: (r) => [called(r, 'generate_schedule') && pending(r, 'generate_schedule'), 'called generate_schedule with pending_confirmation'],
    note: 'needs-confirm',
  },

  // save_worklog (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '保存今天的工作日志：完成了登录修复',
    check: (r) => [called(r, 'save_worklog') && pending(r, 'save_worklog'), 'called save_worklog with pending_confirmation'],
    note: 'needs-confirm',
  },

  // list_tasks (read - auto)
  {
    cat: 'tool-matrix',
    prompt: '看看我的任务列表',
    check: (r) => [called(r, 'list_tasks'), 'called list_tasks'],
  },

  // list_schedule (read - auto)
  {
    cat: 'tool-matrix',
    prompt: '今天有什么安排',
    check: (r) => [called(r, 'list_schedule'), 'called list_schedule'],
  },

  // get_daily_insights (read - auto)
  {
    cat: 'tool-matrix',
    prompt: '给我今天的任务洞察',
    check: (r) => [called(r, 'get_daily_insights'), 'called get_daily_insights'],
  },

  // get_timer_status (read - auto)
  {
    cat: 'tool-matrix',
    prompt: '番茄钟现在是什么状态',
    check: (r) => [called(r, 'get_timer_status'), 'called get_timer_status'],
  },

  // structure_worklog (read - auto)
  {
    cat: 'tool-matrix',
    prompt: '帮我整理一下工作日志的结构',
    check: (r) => [called(r, 'structure_worklog') || askedClarify(r), 'structure_worklog or clarify'],
  },

  // create_task (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '新建一个任务：完成代码审查',
    check: (r) => [called(r, 'create_task') && pending(r, 'create_task'), 'called create_task with pending_confirmation'],
    note: 'needs-confirm',
  },

  // update_task (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '把第一个任务标记为已完成',
    check: (r) => [called(r, 'update_task') && pending(r, 'update_task'), 'called update_task with pending_confirmation'],
    note: 'needs-confirm',
  },

  // delete_task (dangerous - pending)
  {
    cat: 'tool-matrix',
    prompt: '删除所有已完成的任务',
    check: (r) => [called(r, 'delete_task') && pending(r, 'delete_task'), 'called delete_task with pending_confirmation'],
    note: 'needs-confirm',
  },

  // delete_schedule (dangerous - pending)
  {
    cat: 'tool-matrix',
    prompt: '取消明天所有的日程安排',
    check: (r) => [called(r, 'delete_schedule') && pending(r, 'delete_schedule'), 'called delete_schedule with pending_confirmation'],
    note: 'needs-confirm',
  },

  // start_pomodoro (write - pending)
  {
    cat: 'tool-matrix',
    prompt: '开始一个25分钟的番茄钟',
    check: (r) => [called(r, 'start_pomodoro') && pending(r, 'start_pomodoro'), 'called start_pomodoro with pending_confirmation'],
    note: 'needs-confirm',
  },
];
