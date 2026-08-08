import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

export const CASES = [
  {
    cat: 'i18n-timezone',
    prompt: '今天的安排',
    check: (r) => {
      const isCalled = called(r, 'list_schedule');
      const args = argsOf(r, 'list_schedule');
      const todayStr = today();
      // Check that from/to covers today (either exact match or range includes today)
      const coversToday = args && (
        (args.from === todayStr && args.to === todayStr) ||
        (args.from <= todayStr && args.to >= todayStr)
      );
      return [isCalled && coversToday, `list_schedule called with today=${todayStr}, got from=${args?.from}, to=${args?.to}`];
    },
    note: 'timezone: seed UTC vs server local'
  },
  {
    cat: 'i18n-timezone',
    prompt: '8/9/2026 有啥安排',
    check: (r) => [called(r, 'list_schedule'), 'list_schedule should be called with US date format'],
    note: 'US-date-format'
  },
  {
    cat: 'i18n-timezone',
    prompt: '2026/08/09 到 2026/08/11 的安排',
    check: (r) => {
      const isCalled = called(r, 'list_schedule');
      const args = argsOf(r, 'list_schedule');
      const hasFromDate = args && argOf(r, 'list_schedule', 'from');
      const hasToDate = args && argOf(r, 'list_schedule', 'to');
      return [isCalled && hasFromDate && hasToDate, `list_schedule with from=${argOf(r, 'list_schedule', 'from')}, to=${argOf(r, 'list_schedule', 'to')}`];
    },
    note: 'slash-date-format'
  },
  {
    cat: 'i18n-timezone',
    prompt: "What's my schedule today?",
    check: (r) => [called(r, 'list_schedule'), 'list_schedule should be called in English'],
    note: 'english'
  },
  {
    cat: 'i18n-timezone',
    prompt: '帮我 cancel 今天的 meeting',
    check: (r) => {
      // Should either decline or ask for clarification since no cancel/meeting tool exists
      const isDeclined = declined(r);
      const isAsking = askedClarify(r);
      return [isDeclined || isAsking, `Should decline or ask clarify (no cancel/meeting tool), got: ${isDeclined ? 'declined' : isAsking ? 'asked' : 'other'}`];
    },
    note: 'mixed-lang out-of-scope'
  },
  {
    cat: 'i18n-timezone',
    prompt: 'show me unfinished tasks',
    check: (r) => [called(r, 'list_tasks'), 'list_tasks should be called in English'],
    note: 'english'
  },
];
