<template>
  <div class="quadrant-view">
    <div class="quadrants-grid">
      <div
        v-for="quadrant in [1, 2, 3, 4] as Quadrant[]"
        :key="quadrant"
        class="quadrant"
        :class="`quadrant-${quadrant}`"
        @dragover.prevent
        @drop="onDrop($event, quadrant)"
      >
        <div class="quadrant-header">
          <div class="quadrant-info">
            <span class="quadrant-name">{{ quadrantInfo[quadrant].name }}</span>
            <span class="quadrant-count">{{ tasksByQuadrant[quadrant].length }}</span>
          </div>
          <span class="quadrant-desc">{{ quadrantInfo[quadrant].description }}</span>
        </div>

        <div class="quadrant-tasks">
          <TaskCard
            v-for="task in tasksByQuadrant[quadrant]"
            :key="task.id"
            :task="task"
            @drag-start="onDragStart"
            @edit="onEditTask"
            @complete="onCompleteTask"
            @delete="onDeleteTask"
          />
        </div>
      </div>
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
import { ref, computed } from 'vue'
import TaskCard from './TaskCard.vue'
import TaskForm from './TaskForm.vue'
import { useTaskStore } from '@/stores/task'
import { QUADRANT_INFO } from '@/types'
import type { Task, Quadrant } from '@/types'

const taskStore = useTaskStore()

const showForm = ref(false)
const editingTask = ref<Task | null>(null)
const draggedTask = ref<Task | null>(null)

const tasksByQuadrant = computed(() => taskStore.tasksByQuadrant)
const quadrantInfo = QUADRANT_INFO

function onDragStart(event: DragEvent, task: Task) {
  draggedTask.value = task
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

function onDrop(_event: DragEvent, quadrant: Quadrant) {
  if (draggedTask.value && draggedTask.value.quadrant !== quadrant) {
    taskStore.moveTask(draggedTask.value.id, quadrant)
  }
  draggedTask.value = null
}

function onEditTask(task: Task) {
  editingTask.value = task
  showForm.value = true
}

function onAddTask() {
  editingTask.value = null
  showForm.value = true
}

async function onSaveTask(data: any) {
  if (editingTask.value) {
    await taskStore.updateTask(editingTask.value.id, data)
  } else {
    await taskStore.createTask(data)
  }
  showForm.value = false
  editingTask.value = null
}

async function onCompleteTask(id: string) {
  await taskStore.markCompleted(id)
}

async function onDeleteTask(id: string) {
  await taskStore.deleteTask(id)
}

// 暴露添加任务方法
defineExpose({
  onAddTask
})
</script>

<style scoped>
.quadrant-view {
  padding: 24px;
  height: 100%;
}

.quadrants-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 16px;
  height: 100%;
}

.quadrant {
  background: #f9fafb;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  border: 2px solid transparent;
  transition: border-color 0.2s;
}

.quadrant.quadrant-1 { border-color: #fecaca; }
.quadrant.quadrant-2 { border-color: #fde68a; }
.quadrant.quadrant-3 { border-color: #bfdbfe; }
.quadrant.quadrant-4 { border-color: #d1d5db; }

.quadrant-header {
  margin-bottom: 12px;
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
}

.quadrant-count {
  background: #e5e7eb;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.quadrant-desc {
  font-size: 12px;
  color: #6b7280;
}

.quadrant-tasks {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
