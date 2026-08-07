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
  background: var(--gradient-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 18px 20px;
  margin-bottom: 16px;
  box-shadow: var(--shadow-card);
}
.bd-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.bd-title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 15px;
  font-weight: 420;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}
.bd-btn {
  background: var(--accent-primary);
  color: var(--bg-primary);
  border: none;
  border-radius: var(--radius-sm);
  padding: 7px 14px;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-fast), box-shadow var(--transition-fast);
}
.bd-btn:hover:not(:disabled) {
  background: var(--accent-secondary);
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.24);
}
.bd-btn:disabled {
  background: rgba(239, 231, 215, 0.08);
  color: var(--text-muted);
  cursor: not-allowed;
}
.bd-textarea {
  width: 100%;
  min-height: 120px;
  padding: 12px 14px;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-sm);
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}
.bd-textarea::placeholder {
  color: var(--text-muted);
}
.bd-textarea:focus {
  outline: none;
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px rgba(230, 162, 60, 0.12);
}
.bd-textarea:disabled {
  opacity: 0.55;
  cursor: progress;
}
</style>
