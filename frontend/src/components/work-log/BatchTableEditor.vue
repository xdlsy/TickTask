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
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--accent-tertiary, #D98A75);
  border-radius: var(--radius-md);
  padding: 14px 18px;
  margin-bottom: 16px;
}
.bte-header {
  margin-bottom: 10px;
}
.bte-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  margin: 0;
}
.bte-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.bte-table th, .bte-table td {
  padding: 6px 8px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}
.bte-table th {
  background: var(--bg-secondary);
  font-weight: 500;
  color: var(--text-muted);
  font-size: 10px;
  letter-spacing: 0.4px;
  text-transform: uppercase;
}
.bte-row {
  transition: background 0.15s;
}
.bte-row-error {
  background: rgba(184, 69, 44, 0.05);
}
.bte-cell {
  width: 100%;
  border: 1px solid transparent;
  background: transparent;
  font-family: var(--font-body);
  font-size: 12px;
  color: var(--text-primary);
  padding: 3px 5px;
  border-radius: 2px;
  outline: none;
}
.bte-cell:focus {
  background: var(--bg-elevated);
  border-color: var(--accent-primary);
}
.bte-time-pair {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-muted);
}
.bte-quadrant {
  display: flex;
  gap: 2px;
}
.bte-quad-btn {
  width: 22px;
  height: 18px;
  border: 1px solid var(--border-accent);
  background: var(--bg-elevated);
  font-size: 10px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
}
.bte-quad-btn.active {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
}
.bte-delete {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 16px;
}
.bte-delete:hover {
  color: var(--accent-primary);
}
.bte-add-cell {
  text-align: center;
  padding: 6px;
}
.bte-add {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 12px;
  padding: 4px 12px;
}
.bte-add:hover {
  color: var(--accent-primary);
}
.bte-summary {
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
}
.bte-label {
  display: block;
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.bte-summary-input {
  width: 100%;
  min-height: 50px;
  border: none;
  background: transparent;
  font-family: var(--font-body);
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-primary);
  resize: vertical;
  outline: none;
}
.bte-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}
.bte-btn {
  border: none;
  border-radius: var(--radius-sm);
  padding: 6px 16px;
  font-size: 12px;
  font-family: var(--font-body);
  cursor: pointer;
}
.bte-btn-primary {
  background: var(--accent-primary);
  color: white;
}
.bte-btn-primary:hover:not(:disabled) {
  background: var(--accent-secondary);
}
.bte-btn-primary:disabled {
  background: var(--text-muted);
  cursor: not-allowed;
}
.bte-btn-secondary {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-accent);
}
.bte-btn-secondary:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
</style>
