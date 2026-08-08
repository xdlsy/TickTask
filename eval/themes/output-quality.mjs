import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 输出质量测试用例
export const CASES = [
  {
    cat: 'output-quality',
    prompt: '我今天有啥安排',
    check: (r) => {
      const calledSchedule = called(r, 'list_schedule');
      const responseText = txt(r);
      const reasonableLength = responseText.length < 800;
      return [calledSchedule && reasonableLength, calledSchedule ? 'called list_schedule' : 'did not call list_schedule'];
    },
    note: 'conciseness'
  },
  {
    cat: 'output-quality',
    prompt: '我有几个任务',
    check: (r) => {
      const calledTasks = called(r, 'list_tasks');
      const responseText = txt(r);
      const mentionsNumber = /\d+/.test(responseText);
      return [calledTasks && mentionsNumber, calledTasks ? 'called list_tasks with number' : 'did not call list_tasks or missing number'];
    },
    note: 'data-correctness'
  },
  {
    cat: 'output-quality',
    prompt: '简要告诉我今天的安排',
    check: (r) => {
      const calledSchedule = called(r, 'list_schedule');
      const responseText = txt(r);
      const briefLength = responseText.length < 500;
      return [calledSchedule && briefLength, calledSchedule ? 'called list_schedule with brief response' : 'did not call list_schedule or too long'];
    },
    note: 'brevity-requested'
  },
  {
    cat: 'output-quality',
    prompt: '列下任务',
    check: (r) => {
      const calledTasks = called(r, 'list_tasks');
      const noUnsolicitedAction = !anyWrite(r);
      return [calledTasks && noUnsolicitedAction, calledTasks ? 'called list_tasks without unsolicited actions' : 'did not call list_tasks or performed extra actions'];
    },
    note: 'no-unsolicited-action'
  },
  {
    cat: 'output-quality',
    prompt: '今天最重要的安排是什么',
    check: (r) => {
      const calledSchedule = called(r, 'list_schedule');
      return [calledSchedule, calledSchedule ? 'called list_schedule' : 'did not call list_schedule'];
    },
    note: 'llm-judge: picks one + justifies'
  },
];
