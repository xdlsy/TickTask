<template>
  <div class="batch-table-editor">
    <div class="bte-header">
      <h3 class="bte-title">AI 草稿 · 待补齐 ({{ items.length }} 条)</h3>
    </div>

    <table class="bte-table">
      <thead>
        <tr>
          <th style="width: 110px">时段</th>
          <th>活动 *</th>
          <th style="width: 110px">象限</th>
          <th>内容</th>
          <th>解决了什么问题</th>
          <th>已产生的结果</th>
          <th>对后续的影响</th>
          <th style="width: 30px"></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(item, idx) in items"
          :key="idx"
          class="bte-row"
          :class="{ 'bte-row-error': errorIndices.includes(idx) }"
          data-test="draft-row"
        >
          <td>
            <div class="bte-time-pair">
              <input class="bte-cell" :value="item.start_time" @input="onCellInput(idx, 'start_time', $event)" placeholder="09:00" />
              <span>-</span>
              <input class="bte-cell" :value="item.end_time" @input="onCellInput(idx, 'end_time', $event)" placeholder="10:00" />
            </div>
          </td>
          <td><input class="bte-cell" :value="item.activity" @input="onCellInput(idx, 'activity', $event)" placeholder="做了什么" /></td>
          <td>
            <div class="bte-quadrant">
              <button
                v-for="q in [1, 2, 3, 4]"
                :key="q"
                type="button"
                class="bte-quad-btn"
                :class="{ active: item.quadrant === q }"
                @click="onQuadrantClick(idx, q as Quadrant)"
              >Q{{ q }}</button>
            </div>
          </td>
          <td><input class="bte-cell" :value="item.content" @input="onCellInput(idx, 'content', $event)" /></td>
          <td><input class="bte-cell" :value="item.problem_solved" @input="onCellInput(idx, 'problem_solved', $event)" /></td>
          <td><input class="bte-cell" :value="item.result" @input="onCellInput(idx, 'result', $event)" /></td>
          <td><input class="bte-cell" :value="item.impact" @input="onCellInput(idx, 'impact', $event)" /></td>
          <td><button class="bte-delete" data-test="delete-btn" @click="onDelete(idx)">×</button></td>
        </tr>
        <tr>
          <td colspan="8" class="bte-add-cell">
            <button class="bte-add" data-test="add-row" @click="onAdd">+ 加一条</button>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="bte-summary">
      <label class="bte-label">今日小结</label>
      <textarea
        class="bte-summary-input"
        :value="summary"
        @input="$emit('update:summary', ($event.target as HTMLTextAreaElement).value)"
      />
    </div>

    <div class="bte-actions">
      <button class="bte-btn bte-btn-secondary" data-test="discard-btn" @click="$emit('discard')">放弃草稿</button>
      <button class="bte-btn bte-btn-primary" data-test="save-btn" :disabled="saving" @click="onSave">
        {{ saving ? '保存中…' : '批量入库' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkLogStore } from '@/stores/workLog'
import type { Quadrant } from '@/types'

export interface DraftWorkItem {
  activity: string
  start_time: string
  end_time: string
  quadrant: Quadrant
  content: string
  problem_solved: string
  result: string
  impact: string
}

const props = defineProps<{
  date: string
  items: DraftWorkItem[]
  summary: string
}>()

const emit = defineEmits<{
  'update:items': [items: DraftWorkItem[]]
  'update:summary': [summary: string]
  save: []
  discard: []
}>()

const store = useWorkLogStore()
const saving = ref(false)
const errorIndices = ref<number[]>([])

function clone(): DraftWorkItem[] {
  return props.items.map(it => ({ ...it }))
}

function onCellInput(idx: number, key: keyof DraftWorkItem, ev: Event) {
  const next = clone()
  ;(next[idx] as any)[key] = (ev.target as HTMLInputElement).value
  emit('update:items', next)
}

function onQuadrantClick(idx: number, q: Quadrant) {
  const next = clone()
  next[idx].quadrant = q
  emit('update:items', next)
}

function onAdd() {
  emit('update:items', [
    ...props.items,
    {
      activity: '',
      start_time: '09:00',
      end_time: '10:00',
      quadrant: 2 as Quadrant,
      content: '',
      problem_solved: '',
      result: '',
      impact: '',
    },
  ])
}

function onDelete(idx: number) {
  emit('update:items', props.items.filter((_, i) => i !== idx))
}

function validate(): number[] {
  const bad: number[] = []
  props.items.forEach((it, idx) => {
    if (!it.activity.trim()) bad.push(idx)
    else if (!it.start_time || !it.end_time) bad.push(idx)
    else if (it.start_time >= it.end_time) bad.push(idx)
  })
  return bad
}

async function onSave() {
  const bad = validate()
  if (bad.length > 0) {
    errorIndices.value = bad
    ElMessage.error(`第 ${bad.map(i => i + 1).join(', ')} 行必填字段缺失或时段无效`)
    return
  }
  errorIndices.value = []
  saving.value = true
  try {
    const payload = props.items.map(it => ({
      activity: it.activity,
      start_time: it.start_time,
      end_time: it.end_time,
      quadrant: it.quadrant,
      content: it.content,
      problem_solved: it.problem_solved,
      result: it.result,
      impact: it.impact,
    }))
    const { successCount, failureIndices } = await store.addWorkItemsBatch(props.date, payload)
    if (failureIndices.length > 0) {
      ElMessage.error(`${successCount}/${props.items.length} 条成功，失败 ${failureIndices.length} 条`)
      const remaining = failureIndices.map(i => props.items[i])
      emit('update:items', remaining)
      return
    }
    if (props.summary.trim()) {
      await store.updateSummary(props.date, props.summary)
    }
    ElMessage.success(`已入库 ${successCount} 条`)
    emit('save')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.batch-table-editor {
  position: relative;
  background: var(--gradient-card);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-lg);
  padding: 16px 20px;
  margin-bottom: 16px;
  box-shadow: var(--shadow-card);
}
/* 草稿卡:顶端琥珀细线,提示「AI 待入库」的特殊态 */
.batch-table-editor::before {
  content: '';
  position: absolute;
  left: 20px;
  right: 20px;
  top: 0;
  height: 1px;
  background: linear-gradient(90deg, var(--accent-primary), transparent 70%);
  opacity: 0.5;
}
.bte-header {
  margin-bottom: 12px;
}
.bte-title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 16px;
  font-weight: 420;
  margin: 0;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}
.bte-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.bte-table th, .bte-table td {
  padding: 7px 9px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}
.bte-table th {
  background: var(--bg-elevated);
  font-family: var(--font-mono);
  font-weight: 500;
  color: var(--text-muted);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.bte-row {
  transition: background var(--transition-fast);
}
.bte-row:hover {
  background: rgba(239, 231, 215, 0.025);
}
.bte-row-error {
  background: var(--crimson-fill);
}
.bte-row-error:hover {
  background: var(--crimson-fill);
}
.bte-cell {
  width: 100%;
  border: 1px solid transparent;
  background: transparent;
  font-family: var(--font-body);
  font-size: 12px;
  color: var(--text-primary);
  padding: 4px 6px;
  border-radius: var(--radius-sm);
  outline: none;
  transition: background var(--transition-fast), border-color var(--transition-fast);
}
.bte-cell::placeholder {
  color: var(--text-muted);
}
.bte-cell:focus {
  background: rgba(239, 231, 215, 0.05);
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px rgba(230, 162, 60, 0.10);
}
.bte-time-pair {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}
.bte-quadrant {
  display: flex;
  gap: 3px;
}
.bte-quad-btn {
  width: 22px;
  height: 20px;
  border: 1px solid var(--border-accent);
  background: var(--bg-elevated);
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}
.bte-quad-btn:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}
.bte-quad-btn.active {
  background: var(--accent-primary);
  color: var(--bg-primary);
  border-color: var(--accent-primary);
  font-weight: 600;
}
.bte-delete {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 2px 4px;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}
.bte-delete:hover {
  color: var(--accent-crimson);
  background: var(--crimson-fill);
}
.bte-add-cell {
  text-align: center;
  padding: 8px;
}
.bte-add {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  padding: 5px 14px;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}
.bte-add:hover {
  color: var(--accent-primary);
  background: var(--accent-fill);
}
.bte-summary {
  margin-top: 14px;
  padding: 12px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}
.bte-label {
  display: block;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-muted);
  letter-spacing: 0.16em;
  text-transform: uppercase;
  margin-bottom: 8px;
}
.bte-summary-input {
  width: 100%;
  min-height: 50px;
  border: none;
  background: transparent;
  font-family: var(--font-body);
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--text-primary);
  resize: vertical;
  outline: none;
}
.bte-summary-input::placeholder {
  color: var(--text-muted);
}
.bte-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}
.bte-btn {
  border: none;
  border-radius: var(--radius-sm);
  padding: 7px 16px;
  font-size: 12px;
  font-family: var(--font-body);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.bte-btn-primary {
  background: var(--accent-primary);
  color: var(--bg-primary);
  font-weight: 600;
}
.bte-btn-primary:hover:not(:disabled) {
  background: var(--accent-secondary);
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.24);
}
.bte-btn-primary:disabled {
  background: rgba(239, 231, 215, 0.08);
  color: var(--text-muted);
  cursor: not-allowed;
}
.bte-btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border: 1px solid var(--border-accent);
}
.bte-btn-secondary:hover {
  border-color: var(--accent-crimson);
  color: var(--accent-crimson);
  background: var(--crimson-fill);
}
</style>
