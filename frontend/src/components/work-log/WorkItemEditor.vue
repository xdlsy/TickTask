<template>
  <div class="work-item-editor">
    <div class="wie-header">
      <input
        class="wie-title"
        :value="local.title"
        placeholder="标题（20 字以内）"
        @input="onInput('title', ($event.target as HTMLInputElement).value)"
      />
      <button class="wie-remove" @click="$emit('remove')" title="删除">×</button>
    </div>
    <div class="wie-grid">
      <label class="wie-field">
        <span class="wie-label">内容</span>
        <textarea
          class="wie-input"
          :value="local.content"
          @input="onInput('content', ($event.target as HTMLTextAreaElement).value)"
        />
      </label>
      <label class="wie-field">
        <span class="wie-label">解决了什么问题</span>
        <textarea
          class="wie-input"
          :value="local.problem_solved"
          @input="onInput('problem_solved', ($event.target as HTMLTextAreaElement).value)"
        />
      </label>
      <label class="wie-field">
        <span class="wie-label">已产生的结果</span>
        <textarea
          class="wie-input"
          :value="local.result"
          @input="onInput('result', ($event.target as HTMLTextAreaElement).value)"
        />
      </label>
      <label class="wie-field">
        <span class="wie-label">对后续的影响</span>
        <textarea
          class="wie-input"
          :value="local.impact"
          @input="onInput('impact', ($event.target as HTMLTextAreaElement).value)"
        />
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface ItemInput {
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

const props = defineProps<{
  modelValue: ItemInput
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: ItemInput): void
  (e: 'remove'): void
}>()

const local = reactive<ItemInput>({ ...props.modelValue })

watch(() => props.modelValue, (v) => {
  Object.assign(local, v)
})

function onInput<K extends keyof ItemInput>(key: K, value: ItemInput[K]) {
  local[key] = value
  emit('update:modelValue', { ...local })
}
</script>

<style scoped>
.work-item-editor {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 12px;
}
.wie-header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}
.wie-title {
  flex: 1;
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  background: transparent;
  border: none;
  border-bottom: 1px solid var(--border-color);
  padding: 4px 0;
}
.wie-title:focus {
  outline: none;
  border-bottom-color: var(--accent-primary);
}
.wie-remove {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
}
.wie-remove:hover {
  color: var(--accent-primary);
}
.wie-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 16px;
}
.wie-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.wie-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
}
.wie-input {
  width: 100%;
  min-height: 60px;
  padding: 6px 8px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-primary);
  resize: vertical;
}
.wie-input:focus {
  outline: none;
  border-color: var(--accent-primary);
}
</style>
