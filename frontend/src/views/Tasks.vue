<template>
  <div class="tasks-page">
    <div class="tasks-header">
      <div class="header-left">
        <div class="page-title">
          <span class="eyebrow">Tasks</span>
          <h1>任务管理</h1>
          <p class="page-subtitle">先做对的事 — 重要不紧急,才是长期价值的所在</p>
        </div>
        <div class="view-switch">
          <button
            class="view-btn"
            :class="{ active: viewMode === 'quadrant' }"
            @click="viewMode = 'quadrant'"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
            <span class="view-text">四象限</span>
          </button>
          <button
            class="view-btn"
            :class="{ active: viewMode === 'list' }"
            @click="viewMode = 'list'"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="8" y1="6" x2="21" y2="6"/>
              <line x1="8" y1="12" x2="21" y2="12"/>
              <line x1="8" y1="18" x2="21" y2="18"/>
              <line x1="3" y1="6" x2="3.01" y2="6"/>
              <line x1="3" y1="12" x2="3.01" y2="12"/>
              <line x1="3" y1="18" x2="3.01" y2="18"/>
            </svg>
            <span class="view-text">列表</span>
          </button>
        </div>
      </div>
      <el-button type="primary" size="large" @click="onAddTask" class="add-btn">
        <el-icon class="el-icon--left"><Plus /></el-icon>
        添加任务
      </el-button>
    </div>

    <div class="tasks-content">
      <QuadrantView v-if="viewMode === 'quadrant'" ref="quadrantViewRef" />
      <ListView v-else ref="listViewRef" />
    </div>

    <TaskForm
      v-if="showForm"
      :visible="showForm"
      :task="editingTask"
      @close="showForm = false"
      @save="onSaveTask"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Plus } from '@element-plus/icons-vue'
import QuadrantView from '@/components/tasks/QuadrantView.vue'
import ListView from '@/components/tasks/ListView.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { useTaskStore } from '@/stores/task'
import type { Task } from '@/types'

const router = useRouter()
const route = useRoute()
const taskStore = useTaskStore()

const viewMode = ref<'quadrant' | 'list'>('quadrant')
const showForm = ref(false)
const editingTask = ref<Task | null>(null)
const quadrantViewRef = ref()
const listViewRef = ref()

onMounted(async () => {
  await taskStore.fetchTasks()
  await taskStore.fetchTasksByQuadrant()

  if (route.query.add === 'true') {
    onAddTask()
    router.replace({ query: {} })
  }
})

function onAddTask() {
  if (viewMode.value === 'quadrant' && quadrantViewRef.value) {
    quadrantViewRef.value.onAddTask()
  } else if (viewMode.value === 'list' && listViewRef.value) {
    listViewRef.value.onAddTask()
  } else {
    editingTask.value = null
    showForm.value = true
  }
}

function onSaveTask(data: any) {
  taskStore.createTask(data)
  showForm.value = false
}
</script>

<style scoped>
.tasks-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  max-width: 1200px;
  margin: 0 auto;
}

.tasks-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  padding: 0 0 28px 0;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 28px;
}

.header-left {
  display: flex;
  align-items: flex-end;
  gap: 36px;
}

.eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 11px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  color: var(--accent-primary);
  margin-bottom: 12px;
}

.eyebrow::before {
  content: '';
  width: 26px;
  height: 1px;
  background: var(--accent-primary);
  opacity: 0.6;
}

.page-title h1 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 36px;
  font-weight: 380;
  color: var(--text-primary);
  margin: 0 0 8px 0;
  letter-spacing: -0.03em;
  line-height: 1.05;
}

.page-title h1 em {
  font-style: italic;
  font-weight: 360;
  color: var(--text-secondary);
}

.page-subtitle {
  font-size: 13.5px;
  color: var(--text-muted);
  margin: 0;
  font-weight: 400;
}

.view-switch {
  display: flex;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: 4px;
  border: 1px solid var(--border-color);
}

.view-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
}

.view-btn svg {
  width: 16px;
  height: 16px;
}

.view-btn:hover {
  color: var(--text-primary);
  background: rgba(239, 231, 215, 0.04);
}

.view-btn.active {
  background: var(--accent-primary);
  color: var(--bg-primary);
  font-weight: 600;
}

.add-btn {
  height: 44px;
  padding: 0 22px;
  font-size: 14px;
  font-weight: 600;
  border-radius: var(--radius-md);
}

.tasks-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

@media (max-width: 768px) {
  .tasks-header {
    flex-direction: column;
    gap: 20px;
    align-items: flex-start;
    padding: 0 0 24px 0;
  }

  .header-left {
    width: 100%;
    flex-direction: column;
    align-items: flex-start;
    gap: 20px;
  }

  .view-text {
    display: none;
  }

  .add-btn {
    width: 100%;
  }
}
</style>
