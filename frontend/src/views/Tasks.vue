<template>
  <div class="tasks-page">
    <div class="tasks-header">
      <div class="header-left">
        <div class="page-title">
          <h1>任务管理</h1>
          <p class="page-subtitle">管理你的四象限任务</p>
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
import { Plus } from '@element-plus/icons-vue'
import QuadrantView from '@/components/tasks/QuadrantView.vue'
import ListView from '@/components/tasks/ListView.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { useTaskStore } from '@/stores/task'
import type { Task } from '@/types'

const taskStore = useTaskStore()

const viewMode = ref<'quadrant' | 'list'>('quadrant')
const showForm = ref(false)
const editingTask = ref<Task | null>(null)
const quadrantViewRef = ref()
const listViewRef = ref()

onMounted(async () => {
  await taskStore.fetchTasks()
  await taskStore.fetchTasksByQuadrant()
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
  background: #f0f2f5;
}

.tasks-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px 32px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 32px;
}

.page-title h1 {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 2px 0;
}

.page-subtitle {
  font-size: 14px;
  color: #64748b;
  margin: 0;
}

/* 视图切换 */
.view-switch {
  display: flex;
  background: #f1f5f9;
  border-radius: 12px;
  padding: 4px;
}

.view-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
  transition: all 0.2s ease;
}

.view-btn svg {
  width: 18px;
  height: 18px;
}

.view-btn:hover {
  color: #334155;
}

.view-btn.active {
  background: #fff;
  color: #1e293b;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.add-btn {
  height: 44px;
  padding: 0 24px;
  font-size: 15px;
  font-weight: 600;
  border-radius: 10px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border: none;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.4);
}

.add-btn:hover {
  background: linear-gradient(135deg, #60a5fa 0%, #3b82f6 100%);
}

.tasks-content {
  flex: 1;
  overflow: hidden;
}

/* 响应式 */
@media (max-width: 768px) {
  .tasks-header {
    flex-direction: column;
    gap: 16px;
    align-items: flex-start;
    padding: 20px;
  }

  .header-left {
    width: 100%;
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .view-text {
    display: none;
  }

  .add-btn {
    width: 100%;
  }
}
</style>