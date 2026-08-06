<template>
  <el-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)" title="导入数据" width="720px">
    <!-- Step 1: 选文件 -->
    <div v-if="step === 'select'">
      <input type="file" accept="application/json" @change="onFileInputChange" />
      <p v-if="previewError" class="error">{{ previewError }}</p>
    </div>

    <!-- Step 2: 预览 -->
    <div v-if="step === 'preview'">
      <el-alert v-if="preview.schema_warning" type="warning" :title="preview.schema_warning" :closable="false" />

      <!-- 集合模块总览 + 策略 -->
      <div v-for="key in collectionKeys" :key="key" class="module-row">
        <div class="module-summary">
          <strong>{{ moduleLabel[key] }}</strong>
          <span>新增 {{ preview.modules[key]?.new || 0 }}</span>
          <span>相同 {{ preview.modules[key]?.identical || 0 }}</span>
          <span>冲突 {{ preview.modules[key]?.conflict || 0 }}</span>
          <span>仅当前 {{ preview.modules[key]?.orphan || 0 }}</span>
        </div>
        <el-select :model-value="policies[key] || 'add_new_only'" @update:model-value="setPolicy(key, $event)" size="small" style="width: 160px">
          <el-option label="只加新的" value="add_new_only" />
          <el-option label="文件优先" value="merge_file" />
          <el-option label="当前优先" value="merge_current" />
          <el-option label="整模块覆盖" value="replace" />
        </el-select>
      </div>

      <!-- 设置字段级 diff -->
      <div v-if="settingsConflicts.length" class="settings-diff">
        <div class="section-label">设置冲突(逐字段)</div>
        <div v-for="c in settingsConflicts" :key="c.section + '.' + c.field" class="diff-row">
          <span class="diff-field">{{ c.section }}.{{ c.field }}</span>
          <el-radio-group :model-value="settingsChoice[c.section + '.' + c.field] || 'current'" @update:model-value="setSettingsChoice(c.section, c.field, $event)">
            <el-radio value="current">当前:{{ displayValue(c.section, c.field, c.current) }}</el-radio>
            <el-radio value="file">导入:{{ displayValue(c.section, c.field, c.imported) }}</el-radio>
          </el-radio-group>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button v-if="step === 'preview'" type="primary" :loading="applying" @click="clickApply">应用导入</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'
import type { ImportPreview, ImportPolicy, BackupData, ApplyImportRequest } from '@/types'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void; (e: 'applied'): void }>()

const step = ref<'select' | 'preview'>('select')
const preview = ref<ImportPreview>({ schema_version: 1, schema_warning: '', modules: {} })
const filePayload = ref<BackupData | null>(null)
const previewError = ref('')
const applying = ref(false)

const collectionKeys = ['tasks', 'sessions', 'schedules', 'work_logs', 'work_reports'] as const
const moduleLabel: Record<string, string> = {
  tasks: '任务', sessions: '番茄钟会话', schedules: '日程', work_logs: '工作日志', work_reports: '周期报告'
}

const policies = reactive<Record<string, ImportPolicy>>({})
const settingsChoice = reactive<Record<string, 'current' | 'file'>>({})

const settingsConflicts = computed(() => preview.value.modules['settings']?.settings_conflicts || [])

const applyPayload = computed<ApplyImportRequest>(() => {
  const modules: ApplyImportRequest['modules'] = {}
  for (const k of collectionKeys) {
    modules[k] = { policy: policies[k] || 'add_new_only', overrides: {} }
  }
  return { data: resolvedData(), modules }
})

// 合并设置冲突字段到最终 data.settings
function resolvedData(): BackupData {
  const base = JSON.parse(JSON.stringify(filePayload.value || { settings: { pomodoro: {}, ai: {} } })) as BackupData
  if (!base.settings) base.settings = { pomodoro: {} as any, ai: {} as any }
  for (const c of settingsConflicts.value) {
    const src = settingsChoice[c.section + '.' + c.field] === 'file' ? c.imported : c.current
    ;(base.settings as any)[c.section][c.field] = src
  }
  return base
}

// 只掩码 api_key,其余原样展示(数字/字符串)
function displayValue(section: string, field: string, v: unknown): string {
  if (section === 'ai' && field === 'api_key') return '••••'
  if (v === null || v === undefined) return ''
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

async function onFileSelected(file: File) {
  previewError.value = ''
  try {
    const text = await file.text()
    const env = JSON.parse(text)
    if (env.app !== 'ticktask') throw new Error('不是有效的 TickTask 备份文件')
    filePayload.value = env.data as BackupData
    const res = await api.previewImport(file)
    preview.value = res.data
    for (const c of preview.value.modules['settings']?.settings_conflicts || []) {
      settingsChoice[c.section + '.' + c.field] = 'current'
    }
    step.value = 'preview'
  } catch (e: any) {
    previewError.value = e?.message || '预览失败'
  }
}
function onFileInputChange(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) onFileSelected(f)
}

function setPolicy(key: string, p: ImportPolicy) {
  policies[key] = p
}
function setSettingsChoice(section: string, field: string, choice: 'current' | 'file') {
  settingsChoice[section + '.' + field] = choice
}

async function clickApply() {
  const replacing = collectionKeys.find(k => policies[k] === 'replace')
  if (replacing) {
    const ok = window.confirm(`「${moduleLabel[replacing]}」选择了整模块覆盖,将删除当前库中不在备份内的记录,确认?`)
    if (!ok) return
  }
  applying.value = true
  try {
    await api.applyImport(applyPayload.value)
    ElMessage.success('导入成功')
    emit('applied')
    emit('update:modelValue', false)
  } catch {
    ElMessage.error('导入失败,数据未改动')
  } finally {
    applying.value = false
  }
}

defineExpose({
  step, preview, applyPayload, displayValue,
  onFileSelected, setPolicy, setSettingsChoice, clickApply
})
</script>

<style scoped>
.module-row { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--border-color); }
.module-summary { display: flex; gap: 12px; align-items: center; }
.module-summary span { font-size: 13px; color: var(--text-muted); }
.settings-diff { margin-top: 16px; }
.diff-row { display: flex; gap: 12px; align-items: center; padding: 6px 0; }
.diff-field { width: 160px; font-weight: 500; }
.section-label { font-weight: 600; margin-bottom: 8px; }
.error { color: var(--accent-primary); }
</style>
