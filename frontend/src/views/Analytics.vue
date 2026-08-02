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

        <!-- 番茄钟排行榜 -->
        <div class="card">
          <div class="card-header">
            <h3>番茄钟排行榜</h3>
            <div class="pomodoro-period-select">
              <button v-for="p in pomodoroPeriods" :key="p.value" :class="['period-btn', { active: pomodoroPeriod === p.value }]" @click="pomodoroPeriod = p.value; fetchPomodoroStats()">{{ p.label }}</button>
            </div>
          </div>
          <div v-if="pomodoroByTask.length === 0" class="empty-state compact">
            <p>暂无番茄钟数据</p>
            <p class="hint">完成任务后这里会显示统计</p>
          </div>
          <div v-else class="ranking-list">
            <div v-for="(item, index) in pomodoroByTask" :key="item.task_id" class="ranking-item">
              <span class="ranking-index">{{ index + 1 }}</span>
              <span class="ranking-title">{{ item.task_title }}</span>
              <div class="ranking-bar-wrapper">
                <div class="ranking-bar" :style="{ width: getRankingBarWidth(item.completed_pomodoros) }"></div>
              </div>
              <span class="ranking-count">{{ item.completed_pomodoros }} 番茄</span>
            </div>
          </div>
        </div>

        <!-- 计划 vs 实际趋势 -->
        <div class="card">
          <div class="card-header">
            <h3>计划 vs 实际</h3>
            <div class="pomodoro-period-select">
              <button v-for="p in pomodoroPeriods" :key="p.value" :class="['period-btn', { active: pomodoroTrendPeriod === p.value }]" @click="pomodoroTrendPeriod = p.value; fetchPomodoroTrends()">{{ p.label }}</button>
            </div>
          </div>
          <div v-if="pomodoroTrends.length === 0" class="empty-state compact">
            <p>暂无趋势数据</p>
          </div>
          <div v-else class="trend-comparison">
            <div class="trend-legend">
              <span class="legend-item"><span class="legend-dot planned"></span>计划</span>
              <span class="legend-item"><span class="legend-dot actual"></span>实际</span>
            </div>
            <div class="trend-bars">
              <div v-for="day in pomodoroTrends" :key="day.date" class="trend-day">
                <div class="trend-bar-group">
                  <div class="trend-bar planned" :style="{ height: getTrendBarHeight(day.planned) }" :title="`计划: ${day.planned}`"></div>
                  <div class="trend-bar actual" :style="{ height: getTrendBarHeight(day.actual) }" :title="`实际: ${day.actual}`"></div>
                </div>
                <span class="trend-label">{{ formatTrendLabel(day.date) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 番茄钟完成率 -->
        <div class="card">
          <div class="card-header">
            <h3>番茄钟完成率</h3>
            <div class="pomodoro-period-select">
              <button v-for="p in pomodoroPeriods" :key="p.value" :class="['period-btn', { active: pomodoroCompletionPeriod === p.value }]" @click="pomodoroCompletionPeriod = p.value; fetchPomodoroStats()">{{ p.label }}</button>
            </div>
          </div>
          <div v-if="pomodoroByTask.length === 0" class="empty-state compact">
            <p>暂无数据</p>
          </div>
          <div v-else class="completion-rings">
            <div class="ring-item">
              <div class="ring-container">
                <svg viewBox="0 0 36 36" class="ring-svg">
                  <path class="ring-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path class="ring-fill on-time" :stroke-dasharray="`${onTimeRate}, 100`" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                </svg>
                <span class="ring-value">{{ onTimeRate }}%</span>
              </div>
              <span class="ring-label">按时完成</span>
            </div>
            <div class="ring-item">
              <div class="ring-container">
                <svg viewBox="0 0 36 36" class="ring-svg">
                  <path class="ring-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path class="ring-fill exceeded" :stroke-dasharray="`${exceededRate}, 100`" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                </svg>
                <span class="ring-value">{{ exceededRate }}%</span>
              </div>
              <span class="ring-label">超时完成</span>
            </div>
            <div class="ring-item">
              <div class="ring-container">
                <svg viewBox="0 0 36 36" class="ring-svg">
                  <path class="ring-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path class="ring-fill incomplete" :stroke-dasharray="`${incompleteRate}, 100`" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                </svg>
                <span class="ring-value">{{ incompleteRate }}%</span>
              </div>
              <span class="ring-label">未完成</span>
            </div>
          </div>
        </div>

        <!-- AI 每日洞察 -->
        <div class="card ai-insights-card" v-if="aiStore.configured">
          <div class="card-header">
            <h3>AI 每日洞察</h3>
            <button class="insight-btn" @click="fetchAIInsights" :disabled="insightsLoading">
              {{ insightsLoading ? '分析中...' : aiInsights ? '刷新' : '获取洞察' }}
            </button>
          </div>
          <div v-if="aiInsights" class="insights-content">
            <div class="score-row">
              <span class="score-label">生产力评分</span>
              <span class="score-value" :class="getScoreClass(aiInsights.productivity_score)">
                {{ aiInsights.productivity_score }}
              </span>
            </div>
            <div class="insight-block">
              <div class="insight-label">高产时段</div>
              <div class="insight-text">{{ aiInsights.peak_hours }}</div>
            </div>
            <div class="insight-block">
              <div class="insight-label">亮点</div>
              <ul class="insight-list">
                <li v-for="(item, i) in aiInsights.achievements" :key="'a'+i">{{ item }}</li>
              </ul>
            </div>
            <div class="insight-block">
              <div class="insight-label">建议</div>
              <ul class="insight-list">
                <li v-for="(item, i) in aiInsights.suggestions" :key="'s'+i">{{ item }}</li>
              </ul>
            </div>
            <div class="insight-motivation">{{ aiInsights.motivation }}</div>
          </div>
          <div v-else class="insights-empty">
            <p>点击"获取洞察"，AI 将分析今日数据并给出建议</p>
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
import { useAIStore } from '@/stores/ai'
import type { TaskTimeStats, DailySummary, TrendDataPoint, DistributionStats, DailyInsights, PomodoroByTaskItem, PomodoroTrendDay } from '@/types'

const aiStore = useAIStore()
const quadrantInfo = QUADRANT_INFO

const aiInsights = ref<DailyInsights | null>(null)
const insightsLoading = ref(false)

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

// Pomodoro statistics state
const pomodoroByTask = ref<PomodoroByTaskItem[]>([])
const pomodoroTrends = ref<PomodoroTrendDay[]>([])
const pomodoroPeriod = ref<'week' | 'month'>('week')
const pomodoroTrendPeriod = ref<'week' | 'month'>('week')
const pomodoroCompletionPeriod = ref<'week' | 'month'>('week')

const pomodoroPeriods = [
  { label: '周', value: 'week' as const },
  { label: '月', value: 'month' as const }
]

// Completion rate computed
const onTimeRate = computed(() => {
  const tasks = pomodoroByTask.value
  if (tasks.length === 0) return 0
  const onTime = tasks.filter(t => t.status === 'completed').length
  return Math.round((onTime / tasks.length) * 100)
})

const exceededRate = computed(() => {
  const tasks = pomodoroByTask.value
  if (tasks.length === 0) return 0
  const exceeded = tasks.filter(t => t.status === 'exceeded').length
  return Math.round((exceeded / tasks.length) * 100)
})

const incompleteRate = computed(() => {
  const tasks = pomodoroByTask.value
  if (tasks.length === 0) return 0
  const incomplete = tasks.filter(t => t.status === 'not_started' || t.status === 'in_progress').length
  return Math.round((incomplete / tasks.length) * 100)
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

async function fetchPomodoroStats() {
  try {
    const res = await api.getPomodoroByTask(pomodoroPeriod.value)
    pomodoroByTask.value = res.data.tasks || []
  } catch (error) {
    console.error('Failed to fetch pomodoro by task:', error)
  }
}

async function fetchPomodoroTrends() {
  try {
    const res = await api.getPomodoroTrends(pomodoroTrendPeriod.value)
    pomodoroTrends.value = res.data.days || []
  } catch (error) {
    console.error('Failed to fetch pomodoro trends:', error)
  }
}

function getRankingBarWidth(count: number): string {
  const maxCount = Math.max(...pomodoroByTask.value.map(t => t.completed_pomodoros), 1)
  return `${(count / maxCount) * 100}%`
}

const maxTrendValue = computed(() => {
  return Math.max(...pomodoroTrends.value.flatMap(d => [d.planned, d.actual]), 1)
})

function getTrendBarHeight(value: number): string {
  return `${Math.max((value / maxTrendValue.value) * 100, 2)}%`
}

function formatTrendLabel(dateStr: string): string {
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

function getScoreClass(score: number): string {
  if (score >= 80) return 'score-high'
  if (score >= 60) return 'score-mid'
  return 'score-low'
}

function changeFilter(filter: 'today' | 'week' | 'month') {
  currentFilter.value = filter
}

async function fetchAIInsights() {
  if (!aiStore.configured) return
  insightsLoading.value = true
  try {
    const distSummary = Object.entries(distribution.value.quadrant_stats)
      .map(([q, stats]) => `${quadrantInfo[Number(q) as 1|2|3|4]?.name}: ${stats.total}个`)
      .join(', ')
    const result = await aiStore.getDailyInsights({
      date: formatDate(getDateRange().start),
      completed_pomodoros: summary.value.completed_pomodoros,
      total_focus_minutes: Math.floor(summary.value.total_focus_time / 60),
      completed_tasks: summary.value.completed_tasks,
      total_interruptions: 0,
      task_distribution: distSummary
    })
    if (result) {
      aiInsights.value = result
    }
  } catch (error) {
    console.error('Failed to fetch AI insights:', error)
  } finally {
    insightsLoading.value = false
  }
}

watch(currentFilter, () => {
  fetchData()
})

onMounted(() => {
  fetchData()
  fetchPomodoroStats()
  fetchPomodoroTrends()
})
</script>

<style scoped>
.analytics-page {
  padding: 0;
  max-width: 1200px;
  margin: 0 auto;
  position: relative;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 36px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
}

.page-header::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(196, 103, 61, 0.15), transparent);
}

.header-left h1 {
  font-size: 30px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 6px 0;
  letter-spacing: -0.5px;
}

.page-subtitle {
  font-size: 15px;
  color: var(--text-secondary);
  margin: 0;
}

.time-filter {
  display: flex;
  gap: 8px;
  background: var(--bg-elevated);
  padding: 6px;
  border-radius: 14px;
  border: 1px solid var(--border-color);
}

.filter-btn {
  padding: 10px 22px;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all var(--transition-normal);
}

.filter-btn:hover {
  color: var(--text-primary);
}

.filter-btn.active {
  background: var(--gradient-primary);
  color: #fff;
  box-shadow: 0 4px 16px rgba(196, 103, 61, 0.2);
}

/* 概览卡片 */
.overview-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
  margin-bottom: 36px;
}

.overview-card {
  background: var(--bg-card);
  border-radius: 20px;
  padding: 28px;
  display: flex;
  align-items: center;
  gap: 20px;
  border: 1px solid var(--border-color);
  transition: all var(--transition-normal);
}

.overview-card:hover {
  transform: translateY(-6px);
  border-color: rgba(196, 103, 61, 0.2);
  box-shadow: 0 16px 48px rgba(60, 30, 10, 0.1);
}

.card-icon {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  position: relative;
}

.card-icon::after {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: 14px;
  background: inherit;
  filter: blur(12px);
  opacity: 0.4;
  z-index: -1;
}

.card-icon svg {
  width: 24px;
  height: 24px;
  position: relative;
  z-index: 1;
}

.card-icon.pomodoro {
  background: linear-gradient(135deg, #C4554D, #D4786D);
  color: #fff;
  box-shadow: 0 0 24px rgba(196, 85, 77, 0.25);
}

.card-icon.time {
  background: linear-gradient(135deg, #C4973D, #D4AD5E);
  color: #fff;
  box-shadow: 0 0 24px rgba(196, 149, 61, 0.25);
}

.card-icon.completed {
  background: linear-gradient(135deg, #6B8B6F, #8BA88E);
  color: #fff;
  box-shadow: 0 0 24px rgba(107, 139, 111, 0.25);
}

.card-icon.created {
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary));
  color: #fff;
  box-shadow: 0 0 24px rgba(196, 103, 61, 0.2);
}

.card-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--text-primary);
  font-family: var(--font-mono);
}

.card-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-top: 4px;
}

/* 主内容区域 */
.analytics-content {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 28px;
}

.right-column {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.card {
  background: var(--bg-card);
  border-radius: 20px;
  padding: 28px;
  border: 1px solid var(--border-color);
  transition: all var(--transition-normal);
}

.card:hover {
  border-color: rgba(196, 103, 61, 0.12);
  box-shadow: 0 8px 32px rgba(60, 30, 10, 0.06);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.card-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.trend-summary {
  font-size: 13px;
  color: var(--text-secondary);
  background: rgba(196, 103, 61, 0.05);
  padding: 6px 12px;
  border-radius: 12px;
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
  background: var(--gradient-primary);
  border-radius: 6px 6px 0 0;
  transition: all 0.3s ease;
  position: relative;
  cursor: pointer;
  min-height: 4px;
}

.chart-bar:hover {
  filter: brightness(1.2);
  box-shadow: 0 0 16px rgba(196, 103, 61, 0.25);
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
  background: var(--bg-elevated);
  color: var(--text-primary);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  opacity: 0;
  visibility: hidden;
  transition: all 0.2s ease;
  border: 1px solid var(--border-color);
  box-shadow: 0 4px 12px rgba(60, 30, 10, 0.1);
}

.bar-tooltip::after {
  content: '';
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: var(--bg-elevated);
}

.chart-label {
  font-size: 12px;
  color: var(--text-secondary);
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

.quadrant-indicator.quadrant-1 { background: linear-gradient(135deg, #C4554D, #D4786D); box-shadow: 0 0 8px rgba(196, 85, 77, 0.25); }
.quadrant-indicator.quadrant-2 { background: linear-gradient(135deg, #C4973D, #D4AD5E); box-shadow: 0 0 8px rgba(196, 149, 61, 0.25); }
.quadrant-indicator.quadrant-3 { background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary)); box-shadow: 0 0 8px rgba(196, 103, 61, 0.2); }
.quadrant-indicator.quadrant-4 { background: linear-gradient(135deg, var(--text-muted), var(--text-secondary)); }

.quadrant-name {
  font-size: 13px;
  color: var(--text-secondary);
  width: 80px;
  flex-shrink: 0;
}

.quadrant-bar-wrapper {
  flex: 1;
  height: 8px;
  background: var(--bg-elevated);
  border-radius: 4px;
  overflow: hidden;
}

.quadrant-bar {
  height: 100%;
  background: var(--gradient-primary);
  border-radius: 4px;
  transition: width 0.3s ease;
  box-shadow: 0 0 8px rgba(196, 103, 61, 0.2);
}

.quadrant-stat .count {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary);
  width: 28px;
  text-align: right;
}

.task-stats-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
  font-size: 13px;
  color: var(--text-secondary);
}

.completion-rate {
  font-weight: 600;
  color: var(--accent-sage);
  font-size: 15px;
  text-shadow: 0 0 8px rgba(107, 139, 111, 0.25);
}

/* 任务投入列表 */
.empty-state {
  text-align: center;
  padding: 32px 24px;
  background: rgba(196, 103, 61, 0.02);
  border-radius: var(--radius-md);
  border: 1px dashed var(--border-color);
}

.empty-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  color: var(--text-muted);
}

.empty-icon svg {
  width: 100%;
  height: 100%;
}

.empty-state p {
  margin: 0 0 4px 0;
  color: var(--text-secondary);
}

.empty-state .hint {
  font-size: 13px;
  color: var(--text-muted);
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
  border-bottom: 1px solid var(--border-color);
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
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}

.task-stat-count {
  font-size: 12px;
  color: var(--text-secondary);
  background: rgba(196, 103, 61, 0.05);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.task-stat-time {
  height: 6px;
  background: var(--bg-elevated);
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}

.time-bar {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, #C4554D, #C4673D);
  border-radius: 3px;
  transition: width 0.3s ease;
  box-shadow: 0 0 8px rgba(196, 85, 77, 0.25);
}

.task-stat-duration {
  font-size: 13px;
  color: var(--text-secondary);
  text-align: right;
  font-weight: 500;
}

/* AI 洞察 */
.ai-insights-card {
  border-color: rgba(196, 103, 61, 0.15);
}

.insight-btn {
  padding: 6px 16px;
  border: 1px solid rgba(196, 103, 61, 0.3);
  background: rgba(196, 103, 61, 0.06);
  color: var(--accent-primary);
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  font-family: var(--font-body);
  transition: all var(--transition-normal);
}

.insight-btn:hover:not(:disabled) {
  background: rgba(196, 103, 61, 0.12);
  border-color: rgba(196, 103, 61, 0.5);
}

.insight-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.insights-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.score-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: rgba(196, 103, 61, 0.04);
  border-radius: var(--radius-md);
}

.score-label {
  font-size: 14px;
  color: var(--text-secondary);
}

.score-value {
  font-size: 28px;
  font-weight: 700;
  font-family: var(--font-mono);
}

.score-high { color: var(--accent-sage); text-shadow: 0 0 12px rgba(107, 139, 111, 0.3); }
.score-mid { color: var(--accent-gold); text-shadow: 0 0 12px rgba(196, 149, 61, 0.3); }
.score-low { color: var(--accent-crimson); text-shadow: 0 0 12px rgba(196, 85, 77, 0.3); }

.insight-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.insight-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.insight-text {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.insight-list {
  margin: 0;
  padding-left: 18px;
}

.insight-list li {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 2px;
}

.insight-motivation {
  font-size: 14px;
  font-weight: 500;
  color: var(--accent-primary);
  text-align: center;
  padding: 12px;
  background: rgba(196, 103, 61, 0.04);
  border-radius: var(--radius-md);
  font-style: italic;
}

.insights-empty {
  text-align: center;
  padding: 24px 16px;
}

.insights-empty p {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.5;
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

/* Pomodoro statistics */
.pomodoro-period-select {
  display: flex;
  gap: 4px;
  background: rgba(0, 0, 0, 0.03);
  padding: 3px;
  border-radius: 8px;
}

.period-btn {
  padding: 4px 12px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all 0.15s;
}

.period-btn.active {
  background: var(--accent-primary);
  color: #fff;
}

.ranking-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ranking-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ranking-index {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-muted);
  width: 20px;
  text-align: center;
  flex-shrink: 0;
}

.ranking-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
}

.ranking-bar-wrapper {
  flex: 1;
  height: 8px;
  background: #e8e4df;
  border-radius: 4px;
  overflow: hidden;
}

.ranking-bar {
  height: 100%;
  background: #B8452C;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.ranking-count {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  width: 60px;
  text-align: right;
  flex-shrink: 0;
}

.empty-state.compact {
  padding: 20px 16px;
}

.empty-state.compact p {
  font-size: 13px;
}

/* Trend comparison */
.trend-legend {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
  justify-content: center;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
}

.legend-dot.planned {
  background: #e8e4df;
}

.legend-dot.actual {
  background: #B8452C;
}

.trend-bars {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  height: 160px;
}

.trend-day {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}

.trend-bar-group {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 3px;
  width: 100%;
}

.trend-bar {
  flex: 1;
  border-radius: 3px 3px 0 0;
  min-height: 2px;
  transition: height 0.3s ease;
  cursor: pointer;
}

.trend-bar.planned {
  background: #e8e4df;
}

.trend-bar.actual {
  background: #B8452C;
}

.trend-label {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 6px;
  white-space: nowrap;
}

/* Completion rings */
.completion-rings {
  display: flex;
  justify-content: space-around;
  padding: 16px 0;
}

.ring-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.ring-container {
  position: relative;
  width: 80px;
  height: 80px;
}

.ring-svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.ring-bg {
  fill: none;
  stroke: #e8e4df;
  stroke-width: 3;
}

.ring-fill {
  fill: none;
  stroke-width: 3;
  stroke-linecap: round;
  transition: stroke-dasharray 0.5s ease;
}

.ring-fill.on-time {
  stroke: #6B8B6F;
}

.ring-fill.exceeded {
  stroke: #C4973D;
}

.ring-fill.incomplete {
  stroke: #B8452C;
}

.ring-value {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  font-family: var(--font-mono);
}

.ring-label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}
</style>