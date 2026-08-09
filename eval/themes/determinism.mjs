import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// Determinism & flaky quantification tests
// Each test runs 5 times to check stability
export const CASES = [
  {
    cat: 'determinism',
    prompt: '我一会有啥安排吗',
    runs: 5,
    check: (r, ctx) => {
      const rs = ctx?.runs || [];
      if (rs.length < 5) return [false, `only ${rs.length} runs`];
      const allRouted = rs.every(ri => called(ri, 'list_schedule'));
      return [allRouted, `5/5 routed to list_schedule`];
    },
    note: 'determinism'
  },
  {
    cat: 'determinism',
    prompt: '标记不存在的任务 abc 完成',
    runs: 5,
    check: (r, ctx) => {
      const rs = ctx?.runs || [];
      if (rs.length < 5) return [false, `only ${rs.length} runs`];
      const hit = rs.filter(ri => notFabricated(ri)).length;
      return [hit >= 4, `${hit}/5 not fabricated`];
    },
    note: 'determinism'
  },
  {
    cat: 'determinism',
    prompt: '今天有啥安排',
    runs: 5,
    check: (r, ctx) => {
      const rs = ctx?.runs || [];
      if (rs.length < 5) return [false, `only ${rs.length} runs`];
      const allRouted = rs.every(ri => called(ri, 'list_schedule'));
      return [allRouted, `5/5 routed to list_schedule`];
    },
    note: 'determinism'
  },
  {
    cat: 'determinism',
    prompt: '建个任务 test',
    runs: 5,
    check: (r, ctx) => {
      const rs = ctx?.runs || [];
      if (rs.length < 5) return [false, `only ${rs.length} runs`];
      const allPending = rs.every(ri => pending(ri, 'create_task'));
      return [allPending, `5/5 pending create_task`];
    },
    note: 'determinism'
  },
];