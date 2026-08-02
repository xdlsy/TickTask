<template>
  <el-dropdown trigger="click" @command="onCommand">
    <button class="action-btn">+ 生成报告 ▼</button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="weekly">本周周报</el-dropdown-item>
        <el-dropdown-item command="monthly">本月月报</el-dropdown-item>
        <el-dropdown-item command="halfyear">本半年半年报</el-dropdown-item>
        <el-dropdown-item command="yearly">本年年报</el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import type { WorkReportType } from '@/types'

const emit = defineEmits<{ (e: 'generate', type: WorkReportType): void }>()

function onCommand(cmd: WorkReportType) {
  emit('generate', cmd)
}
</script>

<style scoped>
.action-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
}
.action-btn:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
</style>
