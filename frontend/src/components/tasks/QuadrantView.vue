<template>
  <div class="quadrant-view">
    <div class="quadrants-grid">
      <div v-for="q in ([1,2,3,4] as Quadrant[])" :key="q" class="quadrant" @dragover.prevent @drop="onDrop($event, q)">
        <div class="quadrant-header">
          <div class="quadrant-info">
            <span class="quadrant-name">{{ quadrantInfo[q].name }}</span>
            <span class="quadrant-count">{{ tasksByQuadrant[q].length }}</span>
          </div>
          <span class="quadrant-desc">{{ quadrantInfo[q].description }}</span>
        </div>
        <div class="quadrant-tasks">
          <TaskCard v-for="task in tasksByQuadrant[q]" :key="task.id" :task="task" mode="row" @drag-start="onDragStart" @edit="onEditTask" @complete="onCompleteTask" @delete="onDeleteTask" @start-pomodoro="onStartPomodoro" @show-detail="onShowDetail" />
        </div>
      </div>
    </div>
    <TaskForm v-if="showForm" :visible="showForm" :task="editingTask" @close="showForm = false" @save="onSaveTask" />
    <TaskPomodoroDetail ref="pomodoroDetailRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import TaskCard from './TaskCard.vue'
import TaskForm from './TaskForm.vue'
import { useTaskStore } from '@/stores/task'
import { QUADRANT_INFO } from '@/types'
import type { TaskResponse, Quadrant } from '@/types'
import { useTimerStore } from '@/stores/timer'
import TaskPomodoroDetail from './TaskPomodoroDetail.vue'

const taskStore = useTaskStore()
const timerStore = useTimerStore()
const showForm = ref(false)
const editingTask = ref<TaskResponse | null>(null)
const draggedTask = ref<TaskResponse | null>(null)
const pomodoroDetailRef = ref()
const tasksByQuadrant = computed(() => taskStore.tasksByQuadrant)
const quadrantInfo = QUADRANT_INFO

function onDragStart(e: DragEvent, t: TaskResponse) { draggedTask.value = t; if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move' }
function onDrop(_e: DragEvent, q: Quadrant) { if (draggedTask.value && draggedTask.value.quadrant !== q) taskStore.moveTask(draggedTask.value.id, q); draggedTask.value = null }
function onEditTask(t: TaskResponse) { editingTask.value = t; showForm.value = true }
function onShowDetail(t: TaskResponse) { pomodoroDetailRef.value?.open(t) }
function onAddTask() { editingTask.value = null; showForm.value = true }
async function onSaveTask(d: any) { if (editingTask.value) await taskStore.updateTask(editingTask.value.id, d); else await taskStore.createTask(d); showForm.value = false; editingTask.value = null }
async function onCompleteTask(id: string) { await taskStore.markCompleted(id) }
async function onDeleteTask(id: string) { await taskStore.deleteTask(id) }
async function onStartPomodoro(taskId: string) {
  try {
    await timerStore.createSession(taskId, 'work')
  } catch (error) {
    console.error('Failed to start pomodoro:', error)
  }
}
defineExpose({ onAddTask })
</script>

<style scoped>
.quadrant-view { padding: 0; height: 100%; overflow: hidden }

.quadrants-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.quadrant {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-color);
  transition: border-color var(--transition-normal);
  min-height: 0;
  overflow: hidden;
}

.quadrant:hover {
  border-color: var(--border-accent);
}

.quadrant-header {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

.quadrant-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.quadrant-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary);
  letter-spacing: -0.2px;
}

.quadrant-count {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
  font-family: var(--font-mono);
}

.quadrant-desc {
  font-size: 12px;
  color: var(--text-muted);
}

.quadrant-tasks {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
</style>
