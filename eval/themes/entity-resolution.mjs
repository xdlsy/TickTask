import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  askedClarify, notFabricated, askedConfirm,
  argsOf, argOf,
} from '../lib/helpers.mjs';

// 每条用例：{ cat, prompt, check, note? }
// check(r) 返回 [pass: boolean, reason: string]。r 见 _SPEC.md
export const CASES = [
  {
    cat: 'entity-resolution',
    prompt: '删掉『测试』',
    check: (r) => {
      // When there are two tasks named "测试", should ask for clarification
      // NOT blindly delete (assert: NOT pending(delete_task) OR askedClarify)
      if (anyDangerous(r)) {
        return [false, 'should NOT blindly delete when ambiguous task name exists'];
      }
      if (askedClarify(r)) {
        return [true, 'asked for clarification on ambiguous task name'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks first to resolve ambiguity'];
      }
      return [false, 'should handle ambiguous task name (two tasks with same name)'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删掉修复 bug 的那个任务',
    check: (r) => {
      // Partial match should resolve to delete_task (pending) or list_tasks first
      if (pending(r, 'delete_task')) {
        return [true, 'resolved partial match to delete_task'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks to find partial match'];
      }
      return [false, 'should handle partial match by deletion or listing'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '把那个会改到下午',
    check: (r) => {
      // Vague reference should ask for clarification
      if (askedClarify(r)) {
        return [true, 'asked for clarification on vague "that one" reference'];
      }
      if (anyWrite(r)) {
        return [false, 'should NOT execute action on vague reference'];
      }
      return [false, 'should ask for clarification on vague reference'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删掉最重要的那个任务',
    check: (r) => {
      // Should resolve by quadrant OR ask clarification
      // Assert: either pending(delete_task) OR askedClarify, NOT blind deletion
      if (anyDangerous(r) && !askedClarify(r)) {
        // If it went to deletion without clarification, that's wrong unless it resolved by quadrant
        const args = argsOf(r, 'delete_task');
        if (args && args.task_id) {
          // If it resolved to a specific task, verify it used quadrant logic
          return [true, 'resolved "most important" to specific task (likely by quadrant)'];
        }
        return [false, 'should NOT blindly delete without quadrant resolution or clarification'];
      }
      if (askedClarify(r)) {
        return [true, 'asked for clarification on "most important"'];
      }
      if (pending(r, 'delete_task')) {
        return [true, 'pending delete_task after resolving importance'];
      }
      return [false, 'should handle "most important" by quadrant or clarification'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '我有哪些任务',
    check: (r) => {
      // Multi-turn scenario placeholder: first turn lists tasks
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks (first turn of multi-turn)'];
      }
      return [false, 'should list tasks in first turn'];
    },
    note: 'multi-turn'
  },

  {
    cat: 'entity-resolution',
    prompt: '把第一个删了',
    check: (r) => {
      // Multi-turn scenario placeholder: second turn deletes based on previous context
      if (pending(r, 'delete_task')) {
        return [true, 'pending delete_task (second turn of multi-turn)'];
      }
      if (askedClarify(r)) {
        return [true, 'asked for clarification without context'];
      }
      return [false, 'should handle "delete the first" in multi-turn context'];
    },
    note: 'multi-turn'
  },

  {
    cat: 'entity-resolution',
    prompt: '删掉任务',
    check: (r) => {
      // No specific task provided - should ask clarification or not blindly delete
      if (anyDangerous(r)) {
        return [false, 'should NOT delete without specifying which task'];
      }
      if (askedClarify(r)) {
        return [true, 'asked for clarification when no task specified'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks when no task specified'];
      }
      return [false, 'should ask which task to delete'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删除优先级最高的任务',
    check: (r) => {
      // Should resolve by priority or ask clarification
      if (askedClarify(r)) {
        return [true, 'asked for clarification on priority resolution'];
      }
      if (pending(r, 'delete_task')) {
        return [true, 'resolved by priority and pending deletion'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks to determine priority'];
      }
      return [false, 'should handle priority-based entity resolution'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '完成昨天创建的那个任务',
    check: (r) => {
      // Temporal reference should resolve or ask clarification
      if (askedClarify(r)) {
        return [true, 'asked for clarification on temporal reference'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks to resolve temporal reference'];
      }
      if (pending(r, 'update_task')) {
        const args = argsOf(r, 'update_task');
        if (args && args.status === 'completed') {
          return [true, 'resolved temporal reference and completed task'];
        }
      }
      return [false, 'should handle temporal reference'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删除所有完成的任务',
    check: (r) => {
      // Bulk action should either:
      // 1. List tasks first then bulk delete, OR
      // 2. Ask for confirmation (askedConfirm), OR
      // 3. Reject as not supported with multiple delete_task calls
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks before bulk operation'];
      }
      if (askedConfirm(r)) {
        return [true, 'asked for confirmation on bulk delete'];
      }
      if (askedClarify(r) && has(r, /不支持|不能|无法/)) {
        return [true, 'declined bulk operation as not supported'];
      }
      if (anyDangerous(r)) {
        // If it attempted deletion, verify it's confirmation-pending
        return [pending(r, 'delete_task'), 'should be pending confirmation for bulk delete'];
      }
      return [false, 'should handle bulk operation request appropriately'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删掉第二个任务',
    check: (r) => {
      // Ordinal reference without context should ask clarification
      if (askedClarify(r)) {
        return [true, 'asked for clarification on ordinal without context'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks to resolve ordinal reference'];
      }
      return [false, 'should handle ordinal reference ("second task")'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '把红色的任务删除',
    check: (r) => {
      // Attribute-based resolution (color doesn't exist in schema)
      if (askedClarify(r)) {
        return [true, 'asked for clarification on non-existent attribute'];
      }
      if (has(r, /没有.*颜色|不支持|无法/)) {
        return [true, 'declined non-existent attribute filter'];
      }
      if (anyDangerous(r)) {
        return [false, 'should NOT delete based on non-existent attribute'];
      }
      return [false, 'should handle non-existent attribute'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删掉id是5的任务',
    check: (r) => {
      // Direct ID reference should resolve to pending delete_task
      if (pending(r, 'delete_task')) {
        const args = argsOf(r, 'delete_task');
        if (args && args.task_id === 5) {
          return [true, 'resolved ID reference correctly'];
        }
        return [true, 'pending delete_task with ID reference'];
      }
      return [false, 'should resolve explicit ID reference to deletion'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删除紧急且重要的任务',
    check: (r) => {
      // Quadrant-based resolution (Q1: urgent + important)
      if (askedClarify(r)) {
        return [true, 'asked for clarification on quadrant reference'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks to find Q1 tasks'];
      }
      if (pending(r, 'delete_task')) {
        return [true, 'resolved quadrant and pending deletion'];
      }
      return [false, 'should handle quadrant-based entity resolution'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '把安排在今天的任务删掉',
    check: (r) => {
      // Temporal + attribute reference
      if (askedClarify(r)) {
        return [true, 'asked for clarification on today\'s tasks'];
      }
      if (called(r, 'list_tasks') || called(r, 'list_schedule')) {
        return [true, 'listed tasks/schedule to resolve today\'s tasks'];
      }
      if (pending(r, 'delete_task')) {
        return [true, 'resolved today\'s tasks and pending deletion'];
      }
      return [false, 'should handle temporal + attribute reference'];
    }
  },

  {
    cat: 'entity-resolution',
    prompt: '删除任务',
    check: (r) => {
      // Empty/missing entity specification should not blindly delete
      if (anyDangerous(r)) {
        return [false, 'should NOT delete without entity specification'];
      }
      if (askedClarify(r)) {
        return [true, 'asked which task to delete'];
      }
      if (called(r, 'list_tasks')) {
        return [true, 'listed tasks to help user choose'];
      }
      return [false, 'should ask for task specification'];
    }
  },
];
