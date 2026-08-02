<template>
  <div class="brain-dump-input">
    <div class="bd-header">
      <span class="bd-title">脑暴输入</span>
      <button
        class="bd-btn"
        :disabled="loading || !text.trim()"
        @click="onStructure"
      >
        {{ loading ? 'AI 拆条中…' : 'AI 拆条 →' }}
      </button>
    </div>
    <textarea
      class="bd-textarea"
      v-model="text"
      :disabled="loading"
      placeholder="今天做了什么、解决了什么、产出了什么…一段话丢进来，AI 帮你拆成结构化条目。"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{ loading?: boolean; modelValue?: string }>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'structure', text: string): void
}>()

const text = ref(props.modelValue || '')

watch(() => props.modelValue, (v) => {
  if (v !== undefined && v !== text.value) text.value = v
})

watch(text, (v) => emit('update:modelValue', v))

function onStructure() {
  if (!text.value.trim()) return
  emit('structure', text.value)
}
</script>

<style scoped>
.brain-dump-input {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 16px;
}
.bd-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.bd-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.bd-btn {
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--transition-fast);
}
.bd-btn:hover:not(:disabled) {
  background: var(--accent-secondary);
}
.bd-btn:disabled {
  background: var(--text-muted);
  cursor: not-allowed;
}
.bd-textarea {
  width: 100%;
  min-height: 120px;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
}
.bd-textarea:focus {
  outline: none;
  border-color: var(--accent-primary);
}
</style>
