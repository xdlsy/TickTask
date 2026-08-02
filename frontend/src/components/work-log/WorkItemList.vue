<template>
  <div class="work-item-list">
    <WorkItemEditor
      v-for="(item, idx) in items"
      :key="idx"
      :model-value="item"
      @update:model-value="onUpdate(idx, $event)"
      @remove="onRemove(idx)"
    />
    <button class="wil-add" @click="onAdd">+ 加一条</button>

    <div class="wil-summary">
      <label class="wil-label">今日小结</label>
      <textarea
        class="wil-summary-input"
        :value="summary"
        @input="$emit('update:summary', ($event.target as HTMLTextAreaElement).value)"
        placeholder="2-3 句概括今日工作"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import WorkItemEditor from './WorkItemEditor.vue'

interface ItemInput {
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

const props = defineProps<{
  items: ItemInput[]
  summary: string
}>()

const emit = defineEmits<{
  (e: 'update:items', items: ItemInput[]): void
  (e: 'update:summary', summary: string): void
}>()

function onUpdate(idx: number, val: ItemInput) {
  const next = [...props.items]
  next[idx] = val
  emit('update:items', next)
}

function onRemove(idx: number) {
  const next = props.items.filter((_, i) => i !== idx)
  emit('update:items', next)
}

function onAdd() {
  emit('update:items', [
    ...props.items,
    { title: '', content: '', problem_solved: '', result: '', impact: '' },
  ])
}
</script>

<style scoped>
.work-item-list {
  margin-bottom: 16px;
}
.wil-add {
  background: transparent;
  border: 1px dashed var(--border-accent);
  color: var(--text-secondary);
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  cursor: pointer;
  width: 100%;
  transition: all var(--transition-fast);
}
.wil-add:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
.wil-summary {
  margin-top: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
}
.wil-label {
  display: block;
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.wil-summary-input {
  width: 100%;
  min-height: 60px;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
}
.wil-summary-input:focus {
  outline: none;
}
</style>
