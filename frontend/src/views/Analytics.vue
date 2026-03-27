<template>
  <div class="analytics-page">
    <div class="page-header">
      <div class="header-left">
        <h1>数据分析</h1>
        <p class="page-subtitle">追踪你的专注轨迹</p>
      </div>
      <div class="time-filter">
        <button
          v-for="filter in timeFilters"
          :key="filter.value"
          :class="['filter-btn', { active: currentFilter === filter.value }]"
          @click="changeFilter(filter.value)"
        >
          {{ filter.label }}
        </button>
      </div>
    </div>

    <!-- 概览卡片 -->
    <div class="overview-cards">
      <div class="overview-card">
        <div class="card-icon pomodoro">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
        </div>
        <div class="card-content">
          <div class="card-value">{{ summary.completed_pomodoros }}</div>
          <div class="card-label">番茄数</div>
        </div>
      </div>
      <div class="overview-card">
        <div class="card-icon time">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
          </svg>
        </div>
        <div class="card-content">
          <div class="card-value">{{ formatDuration(summary.total_focus_time) }}</div>
          <div class="card-label">专注时长</div>
        </div>
      </div>
      <div class="overview-card">
        <div class="card-icon completed">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <div class="card-content">
          <div class="card-value">{{ summary.completed_tasks }}</div>
          <div class="card-label">完成任务</div>
        </div>
      </div>
      <div class="overview-card">
        <div class="card-icon created">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </div>
        <div class="card-content">
          <div class="card-value">{{ summary.created_tasks }}</div>
          <div class="card-label">创建任务</div>
        </div>
      </div>
    </div>

    <div class="analytics-content">
      <!-- 趋势图表 -->
      <div class="card trend-card">
        <div class="card-header">
          <h3>专注趋势</h3>
          <span class="trend-summary">最近{{ trendData.length }}天</span>
        </div>
        <div class="chart-container">
          <div class="chart-bars">
            <div
              v-for="(point, index) in trendData"
              :key="index"
              class="chart-bar-wrapper"
            >
              <div
                class="chart-bar"
                :style="{ height: getBarHeight(point.focus_time) }"
              >
                <span class="bar-tooltip">{{ formatDuration(point.focus_time) }}</span>
              </div>
              <span class="chart-label">{{ formatChartLabel(point.date, index) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="right-column">
        <!-- 任务分布 -->
        <div class="card">
          <div class="card-header">
            <h3>任务分布</h3>
          </div>
          <div class="quadrant-stats">
            <div class="quadrant-stat" v-for="q in ([1, 2, 3, 4] as const)" :key="q">
              <div class="quadrant-indicator" :class="`quadrant-${q}`"></div>
              <span class="quadrant-name">{{ quadrantInfo[q].name }}</span>
              <div class="quadrant-bar-wrapper">
                <div
                  class="quadrant-bar"
                  :style="{ width: getQuadrantBarWidth(q) }"
                ></div>
              </div>
              <span class="count">{{ distribution.quadrant_stats[q]?.total || 0 }}</span>
            </div>
          </div>
          <div class="task-stats-summary">
            <span>完成率</span>
            <span class="completion-rate">{{ Math.round((distribution.task_stats?.completion_rate || 0) * 100) }}%</span>
          </div>
        </div>

        <!-- 今日任务投入 -->
        <div class="card">
          <div class="card-header">
            <h3>{{ currentFilter === 'today' ? '今日' : '期间' }}任务投入</h3>
          </div>
          <div v-if="todayStats.length === 0" class="empty-state">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 6v6l4 2"/>
            </svg>
            <p>还没有完成任何番茄钟</p>
            <p class="hint">从任务列表开始一个番茄吧！</p>
          </div>
          <div v-else class="task-stats-list">
            <div
              v-for="stat in todayStats"
              :key="stat.task_id || 'no-task'"
              class="task-stat-item"
            >
              <div class="task-stat-header">
                <span class="task-stat-title">{{ stat.task_title || '未关联任务' }}</span>
                <span class="task-stat-count">{{ stat.session_count }} 个番茄</span>
              </div>
              <div class="task-stat-time">
                <span class="time-bar" :style="{ width: getTimeBarWidth(stat.total_time) }"></span>
              </div>
              <div class="task-stat-duration">{{ formatDuration(stat.total_time) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '@/api/client'
import { QUADRANT_INFO } from '@/types'
import type { TaskTimeStats, DailySummary, TrendDataPoint, DistributionStats } from '@/types'

const quadrantInfo = QUADRANT_INFO

const timeFilters: Array<{ label: string; value: 'today' | 'week' | 'month' }> = [
  { label: '今日', value: 'today' },
  { label: '本周', value: 'week' },
  { label: '本月', value: 'month' }
]

const currentFilter = ref<'today' | 'week' | 'month'>('today')
const todayStats = ref<TaskTimeStats[]>([])
const summary = ref<DailySummary>({
  completed_pomodoros: 0,
  total_focus_time: 0,
  completed_tasks: 0,
  created_tasks: 0
})
const trendData = ref<TrendDataPoint[]>([])
const distribution = ref<DistributionStats>({
  quadrant_stats: { 1: { total: 0, completed: 0 }, 2: { total: 0, completed: 0 }, 3: { total: 0, completed: 0 }, 4: { total: 0, completed: 0 } },
  task_stats: { total: 0, completed: 0, completion_rate: 0 }
})

const maxFocusTime = computed(() => {
  return Math.max(...trendData.value.map(p => p.focus_time), 1)
})

function getDateRange() {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())

  if (currentFilter.value === 'today') {
    return { start: today, end: today }
  } else if (currentFilter.value === 'week') {
    const dayOfWeek = today.getDay()
    const monday = new Date(today)
    monday.setDate(today.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1))
    return { start: monday, end: today }
  } else {
    const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)
    return { start: firstDayOfMonth, end: today }
  }
}

function formatDate(date: Date): string {
  return date.toISOString().split('T')[0]
}

function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}秒`
  }
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remainingSeconds = seconds % 60

  if (hours > 0) {
    if (remainingSeconds > 0) {
      return `${hours}h ${minutes}m ${remainingSeconds}s`
    }
    if (minutes > 0) {
      return `${hours}h ${minutes}m`
    }
    return `${hours}h`
  }
  if (remainingSeconds > 0) {
    return `${minutes}m ${remainingSeconds}s`
  }
  return `${minutes}m`
}

function formatChartLabel(dateStr: string, index: number): string {
  const date = new Date(dateStr)
  const weekdays = ['日', '一', '二', '三', '四', '五', '六']

  if (currentFilter.value === 'today') {
    return date.getDate() + '日'
  } else if (currentFilter.value === 'week') {
    return '周' + weekdays[date.getDay()]
  } else {
    if (index % 5 === 0 || index === trendData.value.length - 1) {
      return date.getDate() + '日'
    }
    return ''
  }
}

function getBarHeight(focusTime: number): string {
  const percentage = (focusTime / maxFocusTime.value) * 100
  return `${Math.max(percentage, 2)}%`
}

function getQuadrantBarWidth(q: number): string {
  const total = distribution.value.task_stats.total || 1
  const count = distribution.value.quadrant_stats[q]?.total || 0
  return `${(count / total) * 100}%`
}

function getTimeBarWidth(seconds: number): string {
  const maxTime = Math.max(...todayStats.value.map(s => s.total_time), 1)
  return `${(seconds / maxTime) * 100}%`
}

async function fetchData() {
  const { start, end } = getDateRange()
  const startDate = formatDate(start)
  const endDate = formatDate(end)

  try {
    const summaryRes = await api.getAnalyticsSummary(startDate)
    summary.value = summaryRes.data

    const days = currentFilter.value === 'today' ? 1 : currentFilter.value === 'week' ? 7 : 30
    const trendRes = await api.getAnalyticsTrend(days)
    trendData.value = trendRes.data.data

    const distRes = await api.getAnalyticsDistribution(startDate, endDate)
    distribution.value = distRes.data

    const statsRes = await api.getTodayTaskStats()
    todayStats.value = statsRes.data
  } catch (error) {
    console.error('Failed to fetch analytics data:', error)
  }
}

function changeFilter(filter: 'today' | 'week' | 'month') {
  currentFilter.value = filter
}

watch(currentFilter, () => {
  fetchData()
})

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.analytics-page {
  padding: 32px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}

.header-left h1 {
  font-size: 28px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-size: 15px;
  color: #64748b;
  margin: 0;
}

.time-filter {
  display: flex;
  gap: 8px;
  background: #fff;
  padding: 4px;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.filter-btn {
  padding: 8px 20px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
  transition: all 0.2s ease;
}

.filter-btn:hover {
  color: #334155;
}

.filter-btn.active {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.4);
}

/* 概览卡片 */
.overview-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}

.overview-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.overview-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.card-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.card-icon svg {
  width: 24px;
  height: 24px;
}

.card-icon.pomodoro {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: #fff;
}

.card-icon.time {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #fff;
}

.card-icon.completed {
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  color: #fff;
}

.card-icon.created {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
}

.card-value {
  font-size: 28px;
  font-weight: 700;
  color: #1e2937;
}

.card-label {
  font-size: 14px;
  color: #64748b;
  margin-top: 2px;
}

/* 主内容区域 */
.analytics-content {
  display: grid;
  grid-template-columns: 1fr 360px;
  gap: 24px;
}

.right-column {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.card-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.trend-summary {
  font-size: 12px;
  color: #94a3b8;
  background: #f1f5f9;
  padding: 4px 10px;
  border-radius: 8px;
}

/* 趋势图表 */
.trend-card {
  min-height: 320px;
}

.chart-container {
  height: 240px;
  padding: 20px 0;
}

.chart-bars {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  height: 100%;
  gap: 8px;
}

.chart-bar-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}

.chart-bar {
  width: 100%;
  max-width: 32px;
  background: linear-gradient(180deg, #60a5fa 0%, #3b82f6 100%);
  border-radius: 6px 6px 0 0;
  transition: all 0.3s ease;
  position: relative;
  cursor: pointer;
  min-height: 4px;
}

.chart-bar:hover {
  background: linear-gradient(180deg, #93c5fd 0%, #60a5fa 100%);
}

.chart-bar:hover .bar-tooltip {
  opacity: 1;
  visibility: visible;
}

.bar-tooltip {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  background: #1e293b;
  color: #fff;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  opacity: 0;
  visibility: hidden;
  transition: all 0.2s ease;
}

.bar-tooltip::after {
  content: '';
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: #1e293b;
}

.chart-label {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 10px;
  height: 16px;
}

/* 象限分布 */
.quadrant-stats {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.quadrant-stat {
  display: flex;
  align-items: center;
  gap: 12px;
}

.quadrant-indicator {
  width: 12px;
  height: 12px;
  border-radius: 4px;
  flex-shrink: 0;
}

.quadrant-indicator.quadrant-1 { background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%); }
.quadrant-indicator.quadrant-2 { background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%); }
.quadrant-indicator.quadrant-3 { background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%); }
.quadrant-indicator.quadrant-4 { background: linear-gradient(135deg, #94a3b8 0%, #64748b 100%); }

.quadrant-name {
  font-size: 13px;
  color: #475569;
  width: 80px;
  flex-shrink: 0;
}

.quadrant-bar-wrapper {
  flex: 1;
  height: 8px;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}

.quadrant-bar {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.quadrant-stat .count {
  font-weight: 600;
  font-size: 14px;
  color: #334155;
  width: 28px;
  text-align: right;
}

.task-stats-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #f1f5f9;
  font-size: 13px;
  color: #64748b;
}

.completion-rate {
  font-weight: 600;
  color: #22c55e;
  font-size: 15px;
}

/* 任务投入列表 */
.empty-state {
  text-align: center;
  padding: 32px 24px;
}

.empty-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  color: #cbd5e1;
}

.empty-icon svg {
  width: 100%;
  height: 100%;
}

.empty-state p {
  margin: 0 0 4px 0;
  color: #64748b;
}

.empty-state .hint {
  font-size: 13px;
  color: #94a3b8;
}

.task-stats-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-height: 300px;
  overflow-y: auto;
}

.task-stat-item {
  padding: 14px 0;
  border-bottom: 1px solid #f1f5f9;
}

.task-stat-item:last-child {
  border-bottom: none;
}

.task-stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.task-stat-title {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}

.task-stat-count {
  font-size: 12px;
  color: #94a3b8;
  background: #f1f5f9;
  padding: 2px 8px;
  border-radius: 6px;
}

.task-stat-time {
  height: 6px;
  background: #f1f5f9;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}

.time-bar {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, #ef4444 0%, #f97316 100%);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.task-stat-duration {
  font-size: 13px;
  color: #64748b;
  text-align: right;
  font-weight: 500;
}

/* 响应式 */
@media (max-width: 1200px) {
  .overview-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .analytics-content {
    grid-template-columns: 1fr;
  }

  .right-column {
    flex-direction: row;
  }

  .right-column .card {
    flex: 1;
  }
}

@media (max-width: 768px) {
  .analytics-page {
    padding: 20px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .overview-cards {
    grid-template-columns: 1fr;
  }

  .right-column {
    flex-direction: column;
  }
}
</style>