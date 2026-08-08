<template>
  <div class="agent-input">
    <el-input
      v-model="text"
      type="textarea"
      :rows="2"
      placeholder="问点什么都行..."
      data-testid="agent-input"
      @keydown.enter.exact.prevent="send"
    />
    <div class="send-row">
      <span class="shortcuts"><code>/clear</code><code>/history</code><code>/new</code></span>
      <el-button type="primary" :disabled="disabled || !text" @click="send">发送 ➤</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ disabled?: boolean }>()
const emit = defineEmits<{ send: [text: string] }>()

const text = ref('')
const send = () => {
  if (!text.value) return
  emit('send', text.value)
  text.value = ''
}
</script>

<style scoped>
.agent-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.send-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.shortcuts {
  display: flex;
  gap: 6px;
}
.shortcuts code {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-muted);
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 999px;
  padding: 2px 8px;
}
</style>
