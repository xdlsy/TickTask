import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// Determinism & flaky quantification tests
// Each test needs to run multiple times to check stability
export const CASES = [
  {
    cat: 'determinism',
    prompt: '我一会有啥安排吗',
    check: (r) => [called(r, 'list_schedule'), 'should call list_schedule'],
    note: 'run 5x, assert routing stable'
  },
  {
    cat: 'determinism',
    prompt: '标记不存在的任务 abc 完成',
    check: (r) => [notFabricated(r), 'should not fabricate completion of non-existent task'],
    note: 'run 5x, count false-success rate (Bug#38 quantify)'
  },
  {
    cat: 'determinism',
    prompt: '今天有啥安排',
    check: (r) => [called(r, 'list_schedule'), 'should call list_schedule'],
    note: 'run 5x'
  },
  {
    cat: 'determinism',
    prompt: '建个任务 test',
    check: (r) => [pending(r, 'create_task'), 'should require confirmation for create_task'],
    note: 'run 5x, pending stable'
  },
];