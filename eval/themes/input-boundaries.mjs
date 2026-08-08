import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 每条用例：{ cat, prompt, check, note? }
// check(r) 返回 [pass: boolean, reason: string]。r 见下。note 标执行方式（可选）。
export const CASES = [
  // 1. 200字长标题 → 应成功触发 create_task pending
  {
    cat: 'input-boundaries',
    prompt: `建个任务叫${'A'.repeat(200)}`,
    check: (r) => [pending(r, 'create_task'), 'create_task should be pending for 200-char title'],
    note: 'long-title-200'
  },

  // 2. 500字超长标题 → 应优雅处理（pending/askedClarify/失败但崩），断言 !error
  {
    cat: 'input-boundaries',
    prompt: `建个任务叫${'B'.repeat(500)}`,
    check: (r) => [
      (!r.error && (pending(r, 'create_task') || askedClarify(r) || failed(r, 'create_task') || noTool(r))),
      'should handle gracefully (pending/clarify/fail) without crash for 500-char title'
    ],
    note: 'overlong-title-500'
  },

  // 3. Emoji 标题 → 应正常触发 create_task pending
  {
    cat: 'input-boundaries',
    prompt: '建个任务『🎉🚀 测试 emoji 🐛』',
    check: (r) => [pending(r, 'create_task'), 'create_task should be pending for emoji title'],
    note: 'emoji-in-title'
  },

  // 4. SQL 注入测试 → 应触发 create_task pending（参数化查询防注入），DB 不应受损
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"\'; DROP TABLE tasks; --"',
    check: (r) => [
      pending(r, 'create_task') || (noTool(r) && notFabricated(r)),
      'create_task pending (SQL stored as literal text) OR agent declines suspicious title (safe)'
    ],
    note: 'SQL-in-arg + DB intact'
  },

  // 5. XSS 攻击测试 → 应触发 create_task pending（内容作为纯文本存储，不执行）
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"<script>alert(1)</script>"',
    check: (r) => [
      pending(r, 'create_task') && argOf(r, 'create_task', 'title').includes('<script>alert(1)</script>'),
      'create_task pending and XSS payload in args (stored as text, not executed)'
    ],
    note: 'XSS-in-arg'
  },

  // 6. 无标题（"建个任务"）→ 应追问澄清或拒绝盲建
  {
    cat: 'input-boundaries',
    prompt: '建个任务',
    check: (r) => [
      askedClarify(r) || (!called(r, 'create_task') && !anyWrite(r)),
      'should ask clarification or not blindly create without title'
    ],
    note: 'missing-title-should-clarify'
  },

  // 7. 全空格标题 → 应拒绝/追问，不盲建
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"   "',
    check: (r) => [
      askedClarify(r) || (!called(r, 'create_task') && !anyWrite(r)) || declined(r),
      'should reject/ask for whitespace-only title, not blindly create'
    ],
    note: 'whitespace-only-title-reject'
  },

  // 8. 空字符串标题（显式空引号）→ 应拒绝/追问
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫""',
    check: (r) => [
      askedClarify(r) || (!called(r, 'create_task') && !anyWrite(r)) || declined(r),
      'should reject/ask for empty string title'
    ],
    note: 'empty-string-title-reject'
  },

  // 9. 特殊字符组合（换行符+制表符）→ 应优雅处理
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"line1\\nline2\\ttab"',
    check: (r) => [
      !r.error && (pending(r, 'create_task') || askedClarify(r) || declined(r)),
      'should handle newlines and tabs gracefully without crash'
    ],
    note: 'special-chars-whitespace'
  },

  // 10. Unicode 基本多文种平面外的字符（emoji 已测，这里测中日韩等）→ 应正常处理
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫中文测试한국語日本語',
    check: (r) => [pending(r, 'create_task'), 'create_task should be pending for CJK characters'],
    note: 'CJK-characters'
  },

  // 11. 极端长单字（不中断）→ 应优雅处理
  {
    cat: 'input-boundaries',
    prompt: `建个任务叫${'C'.repeat(1000)}`,
    check: (r) => [
      !r.error && (pending(r, 'create_task') || askedClarify(r) || failed(r, 'create_task') || noTool(r)),
      'should handle 1000-char single word gracefully without crash'
    ],
    note: 'extreme-long-word'
  },

  // 12. 标题含引号嵌套 → 应正确解析
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫\'test "quoted" string\'',
    check: (r) => [
      pending(r, 'create_task') && argOf(r, 'create_task', 'title').includes('quoted'),
      'create_task pending with nested quotes parsed correctly'
    ],
    note: 'nested-quotes'
  },

  // 13. 标题含反斜杠转义 → 应保留原始意图
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"path\\\\to\\\\file"',
    check: (r) => [
      pending(r, 'create_task') && argOf(r, 'create_task', 'title').includes('\\\\'),
      'create_task pending with backslashes preserved'
    ],
    note: 'backslash-paths'
  },

  // 14. 零宽度字符（不可见）→ 应优雅处理
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"test​zero‌width"',
    check: (r) => [
      !r.error && (pending(r, 'create_task') || askedClarify(r) || noTool(r)),
      'should handle zero-width characters gracefully'
    ],
    note: 'zero-width-chars'
  },

  // 15. 控制字符（除\\n\\r\\t外）→ 应优雅处理
  {
    cat: 'input-boundaries',
    prompt: '建个任务叫"testcontrol"',
    check: (r) => [
      !r.error && (pending(r, 'create_task') || askedClarify(r) || noTool(r)),
      'should handle control characters gracefully'
    ],
    note: 'control-chars'
  },
];
