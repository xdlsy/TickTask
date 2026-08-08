<!-- ToolConfirmDialog.vue stub (Task 19 implements) -->
<template>
  <div v-if="messageId" class="tool-confirm-stub" data-testid="tool-confirm-stub">
    <!-- TODO: Task 19 -->
  </div>
</template>

<script setup lang="ts">
defineProps<{
  messageId?: string
  toolName?: string
  args?: Record<string, unknown>
  preview?: unknown
}>()
</script>
