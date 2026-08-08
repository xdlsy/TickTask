<template>
  <div class="settings-page">
    <div class="page-header">
      <span class="eyebrow">Settings</span>
      <h1>设置</h1>
      <p class="page-subtitle">自定义你的工作流程</p>
    </div>

    <!-- 番茄时钟设置 -->
    <div class="settings-card">
      <div class="card-header">
        <div class="card-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="card-icon">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
          <span>番茄时钟</span>
        </div>
      </div>

      <div class="card-content" v-loading="loading">
        <div class="settings-grid">
          <div class="setting-item">
            <label>工作时长</label>
            <div class="setting-control">
              <el-input-number
                v-model="pomodoroSettings.work_duration"
                :min="1"
                :max="60"
                size="large"
              />
              <span class="unit">分钟</span>
            </div>
          </div>

          <div class="setting-item">
            <label>短休息时长</label>
            <div class="setting-control">
              <el-input-number
                v-model="pomodoroSettings.short_break_duration"
                :min="1"
                :max="30"
                size="large"
              />
              <span class="unit">分钟</span>
            </div>
          </div>

          <div class="setting-item">
            <label>长休息时长</label>
            <div class="setting-control">
              <el-input-number
                v-model="pomodoroSettings.long_break_duration"
                :min="5"
                :max="60"
                size="large"
              />
              <span class="unit">分钟</span>
            </div>
          </div>

          <div class="setting-item">
            <label>长休息间隔</label>
            <div class="setting-control">
              <el-input-number
                v-model="pomodoroSettings.long_break_after"
                :min="1"
                :max="10"
                size="large"
              />
              <span class="unit">个番茄</span>
            </div>
          </div>
        </div>

        <div class="settings-toggles">
          <div class="toggle-item">
            <div class="toggle-info">
              <span class="toggle-label">自动开始休息</span>
              <span class="toggle-desc">工作结束后自动开始休息计时</span>
            </div>
            <el-switch v-model="pomodoroSettings.auto_start_break" size="large" />
          </div>

          <div class="toggle-item">
            <div class="toggle-info">
              <span class="toggle-label">自动开始工作</span>
              <span class="toggle-desc">休息结束后自动开始工作计时</span>
            </div>
            <el-switch v-model="pomodoroSettings.auto_start_work" size="large" />
          </div>

          <div class="toggle-item">
            <div class="toggle-info">
              <span class="toggle-label">启用提示音</span>
              <span class="toggle-desc">计时结束时播放提示音</span>
            </div>
            <el-switch v-model="pomodoroSettings.enable_sound" size="large" />
          </div>
        </div>

        <!-- AI 排程偏好 -->
        <div class="ai-preference-section">
          <div class="section-label">AI 排程偏好</div>

          <div class="setting-item full-width">
            <label>打断缓冲比例</label>
            <div class="buffer-control">
              <el-slider
                v-model="pomodoroSettings.buffer_ratio"
                :step="10"
                :min="10"
                :max="30"
                :marks="{ 10: '10%', 20: '20%', 30: '30%' }"
                show-stops
              />
              <span class="buffer-hint">约每天预留 {{ Math.round(pomodoroSettings.work_duration * pomodoroSettings.buffer_ratio / 100) }} 分钟应对打断</span>
            </div>
          </div>

          <div class="setting-item full-width">
            <label>任务类型时段偏好</label>
            <div class="time-preference-grid">
              <div class="pref-row">
                <span class="pref-label">管理类任务</span>
                <el-select v-model="taskTimePrefs.management" size="default" style="width: 160px">
                  <el-option label="上午" value="morning" />
                  <el-option label="下午" value="afternoon" />
                  <el-option label="无所谓" value="any" />
                </el-select>
              </div>
              <div class="pref-row">
                <span class="pref-label">开发类任务</span>
                <el-select v-model="taskTimePrefs.dev" size="default" style="width: 160px">
                  <el-option label="上午" value="morning" />
                  <el-option label="下午" value="afternoon" />
                  <el-option label="无所谓" value="any" />
                </el-select>
              </div>
            </div>
            <div class="form-tip">
              AI 会根据你的历史执行数据不断优化排程，越用越精准
            </div>
          </div>
        </div>

        <div class="card-actions">
          <el-button type="primary" size="large" @click="savePomodoroSettings" :loading="saving">
            保存设置
          </el-button>
        </div>
      </div>
    </div>

    <!-- AI 设置 -->
    <div class="settings-card">
      <div class="card-header">
        <div class="card-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="card-icon">
            <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z"/>
            <circle cx="8" cy="14" r="1.5"/>
            <circle cx="16" cy="14" r="1.5"/>
          </svg>
          <span>AI 智能助手</span>
        </div>
        <el-tag v-if="agentStore.status.configured" type="success" size="large" effect="dark">已配置</el-tag>
        <el-tag v-else type="info" size="large">未配置</el-tag>
      </div>

      <div class="card-content" v-loading="loading">
        <div class="form-item">
          <label>服务商</label>
          <el-select v-model="aiSettings.provider" @change="handleProviderChange" size="large">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Anthropic" value="anthropic" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </div>

        <div class="form-item">
          <label>API Key <span class="required">*</span></label>
          <el-input
            v-model="aiSettings.api_key"
            type="password"
            placeholder="请输入 API Key"
            show-password
            size="large"
          />
        </div>

        <div class="form-item" v-if="aiSettings.provider === 'custom' || aiSettings.provider === 'openai'">
          <label>API 地址</label>
          <el-input
            v-model="aiSettings.base_url"
            placeholder="例如: https://api.openai.com/v1"
            size="large"
          />
          <div class="form-tip">
            {{ aiSettings.provider === 'openai' ? '默认使用 OpenAI 官方地址，如需代理可修改' : '请输入兼容 OpenAI API 的地址' }}
          </div>
        </div>

        <div class="form-item">
          <label>模型</label>
          <el-select
            v-model="aiSettings.model"
            :placeholder="modelPlaceholder"
            allow-create
            filterable
            size="large"
          >
            <el-option
              v-for="model in availableModels"
              :key="model"
              :label="model"
              :value="model"
            />
          </el-select>
          <div class="form-tip">
            可选择预设模型或手动输入
          </div>
        </div>

        <div class="card-actions">
          <el-button type="primary" size="large" @click="saveAISettings" :loading="saving">
            保存设置
          </el-button>
          <el-button size="large" @click="testAIConnection" :loading="testing">
            测试连接
          </el-button>
        </div>
      </div>
    </div>

    <!-- 数据管理 -->
    <div class="settings-card">
      <div class="card-header">
        <div class="card-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="card-icon">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <span>数据管理</span>
        </div>
      </div>
      <div class="card-content">
        <p class="form-tip">导出全部数据为 JSON 备份;或从备份文件导入(支持冲突人工解决)。</p>
        <div class="card-actions">
          <el-button type="primary" size="large" data-test="export-btn" @click="exportData" :loading="exporting">导出全部数据</el-button>
          <el-button size="large" data-test="import-btn" @click="importVisible = true">导入数据</el-button>
        </div>
        <ImportWizard v-model="importVisible" @applied="onImported" />

        <div class="clear-zone">
          <div class="clear-text">
            <span class="clear-title">清空全部数据</span>
            <span class="clear-desc">删除所有任务、番茄记录、日程与工作日志(配置与 AI Key 保留)。操作不可恢复。</span>
          </div>
          <el-button type="danger" size="large" data-test="clear-btn" :loading="clearing" @click="clearAllData">清空全部数据</el-button>
        </div>
      </div>
    </div>

    <!-- 关于 -->
    <div class="settings-card about-card">
      <div class="card-header">
        <div class="card-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="card-icon">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="16" x2="12" y2="12"/>
            <line x1="12" y1="8" x2="12.01" y2="8"/>
          </svg>
          <span>关于</span>
        </div>
      </div>

      <div class="card-content">
        <div class="about-info">
          <div class="about-brand">
            <div class="brand-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M12 6v6l4 2"/>
              </svg>
            </div>
            <div class="brand-info">
              <h3>TickTask</h3>
              <p>个人时间管理工具</p>
            </div>
          </div>
          <p class="about-desc">集成番茄工作法、四象限法则和 AI 智能推荐，帮助你高效管理时间</p>
          <p class="version">版本 1.0.0</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api/client'
import { useAgentStore } from '@/stores/agent'
import ImportWizard from '@/components/settings/ImportWizard.vue'
import type { PomodoroSettings, AISettings, ClearResult } from '@/types'

const agentStore = useAgentStore()

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)

const pomodoroSettings = ref<PomodoroSettings>({
  work_duration: 25,
  short_break_duration: 5,
  long_break_duration: 15,
  long_break_after: 4,
  auto_start_break: false,
  auto_start_work: false,
  enable_sound: true,
  buffer_ratio: 20,
  task_time_preferences: '{"management":"any","dev":"any"}'
})

const aiSettings = ref<AISettings>({
  provider: 'openai',
  api_key: '',
  base_url: 'https://api.openai.com/v1',
  model: 'gpt-4o-mini'
})

const openaiModels = ['gpt-4o-mini', 'gpt-4o', 'gpt-4-turbo', 'gpt-3.5-turbo']
const anthropicModels = ['claude-3-5-sonnet-latest', 'claude-3-5-haiku-latest', 'claude-3-opus-latest']

const availableModels = computed(() => {
  if (aiSettings.value.provider === 'openai') return openaiModels
  if (aiSettings.value.provider === 'anthropic') return anthropicModels
  return []
})

const modelPlaceholder = computed(() => {
  if (aiSettings.value.provider === 'openai') return '选择或输入 OpenAI 模型'
  if (aiSettings.value.provider === 'anthropic') return '选择或输入 Anthropic 模型'
  return '输入模型名称'
})

const taskTimePrefs = computed({
  get: () => {
    try {
      return JSON.parse(pomodoroSettings.value.task_time_preferences)
    } catch {
      return { management: 'any', dev: 'any' }
    }
  },
  set: (val) => {
    pomodoroSettings.value.task_time_preferences = JSON.stringify(val)
  }
})

function handleProviderChange() {
  if (aiSettings.value.provider === 'openai') {
    aiSettings.value.base_url = 'https://api.openai.com/v1'
    aiSettings.value.model = 'gpt-4o-mini'
  } else if (aiSettings.value.provider === 'anthropic') {
    aiSettings.value.base_url = 'https://api.anthropic.com/v1'
    aiSettings.value.model = 'claude-3-5-sonnet-latest'
  } else {
    aiSettings.value.base_url = ''
    aiSettings.value.model = ''
  }
}

async function loadSettings() {
  loading.value = true
  try {
    const res = await api.getSettings()
    if (res.data.pomodoro) {
      pomodoroSettings.value = {
        work_duration: Math.floor(res.data.pomodoro.work_duration / 60),
        short_break_duration: Math.floor(res.data.pomodoro.short_break_duration / 60),
        long_break_duration: Math.floor(res.data.pomodoro.long_break_duration / 60),
        long_break_after: res.data.pomodoro.long_break_after,
        auto_start_break: res.data.pomodoro.auto_start_break,
        auto_start_work: res.data.pomodoro.auto_start_work,
        enable_sound: res.data.pomodoro.enable_sound,
        buffer_ratio: res.data.pomodoro.buffer_ratio || 20,
        task_time_preferences: res.data.pomodoro.task_time_preferences || '{"management":"any","dev":"any"}'
      }
    }
    if (res.data.ai) {
      aiSettings.value = res.data.ai
    }
  } catch (error) {
    ElMessage.error('加载设置失败')
  } finally {
    loading.value = false
  }
}

async function savePomodoroSettings() {
  saving.value = true
  try {
    const data = {
      work_duration: pomodoroSettings.value.work_duration * 60,
      short_break_duration: pomodoroSettings.value.short_break_duration * 60,
      long_break_duration: pomodoroSettings.value.long_break_duration * 60,
      long_break_after: pomodoroSettings.value.long_break_after,
      auto_start_break: pomodoroSettings.value.auto_start_break,
      auto_start_work: pomodoroSettings.value.auto_start_work,
      enable_sound: pomodoroSettings.value.enable_sound,
      buffer_ratio: pomodoroSettings.value.buffer_ratio,
      task_time_preferences: pomodoroSettings.value.task_time_preferences
    }
    await api.updatePomodoroSettings(data)
    ElMessage.success('番茄时钟设置已保存')
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function saveAISettings() {
  saving.value = true
  try {
    await api.updateAISettings(aiSettings.value)
    await agentStore.checkStatus()
    ElMessage.success('AI 设置已保存')
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function testAIConnection() {
  if (!aiSettings.value.api_key && aiSettings.value.provider !== 'claude' && aiSettings.value.provider !== 'cli') {
    ElMessage.warning('请先输入 API Key')
    return
  }

  testing.value = true
  try {
    // 先保存当前编辑中的设置,后端才能用最新 key/model 去发测试请求
    await api.updateAISettings(aiSettings.value)
    await agentStore.checkStatus()

    const r = await api.agent.test()
    const result = r.data
    if (result.ok) {
      const latency = result.latency_ms ? ` · ${result.latency_ms}ms` : ''
      const model = result.model ? ` · ${result.model}` : ''
      ElMessage.success(`${result.provider}${model} 连接成功${latency}`)
    } else {
      const errMsg = result.error ? `: ${result.error}` : ' — 请检查 API Key、BaseURL、Model 与网络'
      ElMessage({
        type: 'error',
        message: `${result.provider} 连接失败${errMsg}`,
        duration: 8000,
      })
    }
  } catch (error: any) {
    const detail = error?.response?.data?.error || error?.message || '未知错误'
    ElMessage({
      type: 'error',
      message: `连接测试请求失败: ${detail}`,
      duration: 8000,
    })
  } finally {
    testing.value = false
  }
}

const exporting = ref(false)
const importVisible = ref(false)
const clearing = ref(false)

function exportData() {
  // 直接走真实端点 URL(而非 blob URL):服务端用 Content-Disposition 决定文件名
  // (ticktask-backup-<ts>.json),同源 <a> 也会保留后缀。blob URL 会丢弃
  // Content-Disposition,而部分浏览器(尤其是内嵌 webview)会忽略其 download 属性,
  // 于是回退成 blob 的 UUID、丢失扩展名。
  exporting.value = true
  const a = document.createElement('a')
  a.href = '/api/data/export?include_api_key=true'
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  ElMessage.success('导出成功')
  setTimeout(() => { exporting.value = false }, 800)
}

async function clearAllData() {
  // 1. 是否先备份:confirm = 先备份再清空;cancel = 直接清空;close(X/Esc) = 取消
  let backup = false
  try {
    await ElMessageBox.confirm(
      '此操作将清空所有任务、番茄记录、日程与工作日志,且不可恢复。配置与 AI Key 会保留。',
      '清空全部数据',
      {
        confirmButtonText: '先备份再清空',
        cancelButtonText: '直接清空',
        distinguishCancelAndClose: true,
        type: 'warning'
      }
    )
    backup = true
  } catch (action) {
    if (action === 'cancel') {
      backup = false
    } else {
      return // close → 用户放弃
    }
  }

  if (backup) {
    exportData() // 触发与「导出全部数据」相同的下载
  }

  // 2. 最终确认:输入「清空」
  try {
    await ElMessageBox.prompt('请输入「清空」以确认。', '最终确认', {
      confirmButtonText: '清空全部数据',
      cancelButtonText: '取消',
      inputPlaceholder: '清空',
      inputValidator: (v: string) => v === '清空' || '请输入「清空」以确认',
      type: 'error'
    })
  } catch {
    return
  }

  // 3. 执行
  clearing.value = true
  try {
    const { data }: { data: ClearResult } = await api.clearAll()
    const total = data.tasks + data.sessions + data.schedules + data.work_logs + data.work_reports + data.daily_stats
    ElMessage.success(`已清空 ${total} 条记录`)
    setTimeout(() => location.reload(), 600)
  } catch {
    ElMessage.error('清空失败,请重试')
  } finally {
    clearing.value = false
  }
}

async function onImported() {
  await loadSettings()
  await agentStore.checkStatus()
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-page {
  padding: 0;
  max-width: 900px;
  margin: 0 auto;
}

/* ── Page header ── */
.page-header {
  margin-bottom: 40px;
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
  margin-bottom: 14px;
}

.eyebrow::before {
  content: '';
  width: 26px;
  height: 1px;
  background: var(--accent-primary);
  opacity: 0.6;
}

.page-header h1 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 38px;
  font-weight: 380;
  color: var(--text-primary);
  margin: 0 0 10px 0;
  letter-spacing: -0.03em;
  line-height: 1.05;
}

.page-header h1 em {
  font-style: italic;
  font-weight: 360;
  color: var(--text-secondary);
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0;
  font-weight: 400;
}

/* ── Settings cards ── */
.settings-card {
  background: var(--gradient-card);
  border-radius: var(--radius-xl);
  margin-bottom: 24px;
  border: 1px solid var(--border-color);
  overflow: hidden;
  box-shadow: var(--shadow-card);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 22px 28px;
  border-bottom: 1px solid var(--border-color);
}

.card-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 19px;
  font-weight: 420;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

.card-icon {
  width: 18px;
  height: 18px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.card-content {
  padding: 28px;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24px;
  margin-bottom: 28px;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.setting-item label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
}

.setting-control {
  display: flex;
  align-items: center;
  gap: 14px;
}

.setting-control .el-input-number {
  width: 130px;
}

.unit {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
  letter-spacing: 0.06em;
}

.settings-toggles {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 24px 0;
  border-top: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 28px;
}

.toggle-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.toggle-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toggle-label {
  font-size: 14.5px;
  font-weight: 500;
  color: var(--text-primary);
}

.toggle-desc {
  font-size: 13px;
  color: var(--text-muted);
}

.card-actions {
  display: flex;
  gap: 14px;
}

.ai-preference-section {
  padding: 24px 0;
  border-top: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 28px;
}

.section-label {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 16px;
  font-weight: 440;
  color: var(--text-primary);
  margin-bottom: 20px;
  letter-spacing: -0.01em;
}

.setting-item.full-width {
  margin-bottom: 20px;
}

.buffer-control {
  margin-top: 4px;
}

.buffer-hint {
  display: block;
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 12px;
}

.time-preference-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 4px;
}

.pref-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 400px;
}

.pref-label {
  font-size: 14px;
  color: var(--text-primary);
}

.card-actions .el-button {
  min-width: 130px;
  border-radius: var(--radius-md);
}

.clear-zone {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin-top: 22px;
  padding-top: 20px;
  border-top: 1px solid var(--border-color);
}

.clear-text {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.clear-title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 40;
  font-size: 15px;
  font-weight: 440;
  color: var(--accent-crimson);
  letter-spacing: -0.01em;
}

.clear-desc {
  font-size: 12.5px;
  color: var(--text-muted);
  line-height: 1.5;
}

.clear-zone .el-button {
  flex-shrink: 0;
}

/* el-slider — dark (not covered by global App.vue overrides) */
:deep(.el-slider__runway) {
  background-color: rgba(239, 231, 215, 0.08);
  height: 4px;
  margin: 14px 0;
}

:deep(.el-slider__bar) {
  background-color: var(--accent-primary);
  height: 4px;
}

:deep(.el-slider__button) {
  width: 14px;
  height: 14px;
  border: 2px solid var(--accent-primary);
  background-color: var(--bg-secondary);
  box-shadow: 0 0 0 4px rgba(230, 162, 60, 0.10);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

:deep(.el-slider__button:hover),
:deep(.el-slider__button.hover),
:deep(.el-slider__button.dragging) {
  border-color: var(--accent-secondary);
  box-shadow: 0 0 0 6px rgba(230, 162, 60, 0.14);
}

:deep(.el-slider__stop) {
  background-color: var(--text-muted);
  opacity: 0.5;
}

:deep(.el-slider__marks-text) {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-muted);
  letter-spacing: 0.08em;
  margin-top: 10px;
}

/* ── AI settings form ── */
.form-item {
  margin-bottom: 24px;
}

.form-item:last-of-type {
  margin-bottom: 28px;
}

.form-item label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 10px;
}

.form-item .required {
  color: var(--accent-crimson);
  margin-left: 2px;
}

.form-item .el-select,
.form-item .el-input {
  width: 100%;
}

.form-tip {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 8px;
}

/* ── About card ── */
.about-card .about-info {
  text-align: center;
  padding: 20px 0;
}

.about-brand {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18px;
  margin-bottom: 20px;
}

.brand-icon {
  width: 48px;
  height: 48px;
  background: var(--accent-fill);
  border: 1px solid rgba(230, 162, 60, 0.25);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent-primary);
}

.brand-icon svg {
  width: 24px;
  height: 24px;
}

.brand-info h3 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 22px;
  font-weight: 440;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.brand-info p {
  font-size: 13px;
  color: var(--text-muted);
  margin: 4px 0 0 0;
}

.about-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 20px 0;
  max-width: 450px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.6;
}

.version {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
  margin: 0;
  letter-spacing: 0.12em;
}

@media (max-width: 768px) {
  .settings-page {
    padding: 0;
  }

  .settings-grid {
    grid-template-columns: 1fr;
  }

  .card-actions {
    flex-direction: column;
  }

  .card-actions .el-button {
    width: 100%;
  }
}
</style>
