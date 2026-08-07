<template>
  <div class="today-panorama">
    <h3 class="panorama-title">今日全景 · {{ date }}</h3>
    <el-table v-if="items.length" :data="items" stripe size="small">
      <el-table-column label="时段" width="160">
        <template #default="{ row }">
          {{ row.start_time }} - {{ row.end_time }}
        </template>
      </el-table-column>
      <el-table-column label="活动" prop="activity" />
      <el-table-column label="象限" width="160">
        <template #default="{ row }">
          <el-tag
            v-if="row.quadrant"
            :color="quadrantColor(row.quadrant)"
            effect="dark"
            size="small"
          >
            Q{{ row.quadrant }} {{ quadrantName(row.quadrant) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button data-test="edit-btn" size="small" link @click="$emit('edit', row.id)">编辑</el-button>
          <el-popconfirm title="确定删除？" @confirm="onDelete(row.id)">
            <template #reference>
              <el-button data-test="delete-btn" size="small" link type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-else description="还没有记录，先在上面录入一条吧" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useWorkLogStore } from '@/stores/workLog'
import { QUADRANT_INFO } from '@/types'
import type { Quadrant } from '@/types'

const props = defineProps<{ date: string }>()
defineEmits<{ edit: [itemId: string] }>()

const store = useWorkLogStore()

const items = computed(() => store.todayManualItems)

function quadrantColor(q: number | null | undefined): string {
  if (!q) return '#6b7280'
  return QUADRANT_INFO[q as Quadrant].color
}

function quadrantName(q: number | null | undefined): string {
  if (!q) return ''
  return QUADRANT_INFO[q as Quadrant].name
}

async function onDelete(itemId: string) {
  await store.deleteQuickEntry(props.date, itemId)
}
</script>

<style scoped>
.today-panorama {
  background: var(--gradient-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 20px 22px;
  margin-bottom: 16px;
  box-shadow: var(--shadow-card);
}
.panorama-title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 18px;
  font-weight: 420;
  margin: 0 0 16px 0;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

/* 象限标签:彩色底 + 墨色字,保证暗色下的对比度 */
.today-panorama :deep(.el-tag) {
  color: var(--bg-primary) !important;
  border-color: transparent !important;
  font-weight: 600;
  letter-spacing: 0.04em;
}
</style>
