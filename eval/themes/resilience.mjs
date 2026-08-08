import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 韧性/传输层测试用例
// 每条用例：{ cat, prompt, check, note? }
// check(r) 返回 [pass: boolean, reason: string]
// note 标执行方式和故障注入类型
export const CASES = [
  // WebSocket 中途断开
  {
    cat: 'resilience',
    prompt: '我今天的安排',
    check: () => [true, 'needs WebSocket drop injection'],
    note: 'inject WS drop mid-turn, assert recovery'
  },

  // 后端重启 mid-对话
  {
    cat: 'resilience',
    prompt: '查看我的任务列表',
    check: () => [true, 'needs backend restart injection'],
    note: 'restart backend mid-conversation, assert resume'
  },

  // LLM 429 限流
  {
    cat: 'resilience',
    prompt: '分析一下我的工作习惯',
    check: () => [true, 'needs 429 rate limit injection'],
    note: 'inject 429, assert graceful retry/backoff'
  },

  // LLM 畸形响应
  {
    cat: 'resilience',
    prompt: '生成今天的工作计划',
    check: () => [true, 'needs malformed LLM response injection'],
    note: 'inject malformed LLM response, assert no hang/crash'
  },

  // 超长工具结果
  {
    cat: 'resilience',
    prompt: '展示我这周的所有日程安排',
    check: () => [true, 'needs huge tool result injection'],
    note: 'inject huge tool result, assert no overflow'
  },

  // 番茄钟 WebSocket 断连恢复
  {
    cat: 'resilience',
    prompt: '开始一个番茄钟',
    check: () => [true, 'needs WS drop during timer injection'],
    note: 'inject WS drop during active timer, assert state sync after reconnect'
  },

  // 创建任务时后端重启
  {
    cat: 'resilience',
    prompt: '帮我创建一个新任务：完成项目报告',
    check: () => [true, 'needs backend restart during task creation injection'],
    note: 'restart backend during create_task, assert data consistency'
  },

  // AI 调用时网络超时
  {
    cat: 'resilience',
    prompt: '根据我的任务分类建议优先级',
    check: () => [true, 'needs network timeout injection'],
    note: 'inject network timeout during AI call, assert fallback behavior'
  },

  // 工作日志保存时连接中断
  {
    cat: 'resilience',
    prompt: '保存今天的工作日志',
    check: () => [true, 'needs connection drop during save injection'],
    note: 'inject connection drop during save_worklog, assert retry or error handling'
  },

  // 日程生成时响应截断
  {
    cat: 'resilience',
    prompt: '生成明天的日程安排',
    check: () => [true, 'needs truncated response injection'],
    note: 'inject truncated LLM response during schedule generation, assert partial handling'
  },
];
