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
        <!-- 冲突清单:可展开,逐条覆盖 -->
        <details v-if="preview.modules[key]?.conflicts?.length" class="conflict-list">
          <summary>冲突清单 ({{ preview.modules[key].conflicts.length }})</summary>
          <div v-for="c in preview.modules[key].conflicts" :key="c.id" class="conflict-item">
            <div class="conflict-id">{{ c.id }}</div>
            <ul class="conflict-fields">
              <li v-for="f in c.fields" :key="f.field">
                {{ f.field }}: {{ displayValue('', '', f.current) }} → {{ displayValue('', '', f.imported) }}
              </li>
            </ul>
            <el-radio-group
              :model-value="overrides[key]?.[c.id] || 'policy'"
              @update:model-value="setOverride(key, c.id, $event)"
              size="small"
            >
              <el-radio value="policy" class="opt-policy">跟随策略</el-radio>
              <el-radio value="current" class="opt-current">当前</el-radio>
              <el-radio value="file" class="opt-file">文件</el-radio>
            </el-radio-group>
          </div>
        </details>
      </div>

      <!-- 设置字段级 diff -->
      <div v-if="settingsConflicts.length" class="settings-diff">
        <div class="section-label">设置冲突(逐字段)</div>
        <div v-for="c in settingsConflicts" :key="c.section + '.' + c.field" class="diff-row">
          <span class="diff-field">{{ c.section }}.{{ c.field }}</span>
          <el-radio-group :model-value="settingsChoice[c.section + '.' + c.field] || 'current'" @update:model-value="setSettingsChoice(c.section, c.field, $event)">
            <el-radio value="current" class="opt-current">当前:{{ displayValue(c.section, c.field, c.current) }}</el-radio>
            <el-radio value="file" class="opt-file">导入:{{ displayValue(c.section, c.field, c.imported) }}</el-radio>
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
// 逐条冲突覆盖:module → id → 'file' | 'current'。'policy' 表示不写入,跟随模块策略。
const overrides = reactive<Record<string, Record<string, 'file' | 'current'>>>({})

const settingsConflicts = computed(() => preview.value.modules['settings']?.settings_conflicts || [])

const applyPayload = computed<ApplyImportRequest>(() => {
  const modules: ApplyImportRequest['modules'] = {}
  for (const k of collectionKeys) {
    modules[k] = { policy: policies[k] || 'add_new_only', overrides: overrides[k] || {} }
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
    // 重置逐条覆盖(新文件不沿用旧选择)
    for (const k of Object.keys(overrides)) delete overrides[k]
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
// 设置冲突记录的 override;'policy' 删除 override(回退到模块策略),'file'/'current' 写入。
function setOverride(module: string, id: string, choice: 'file' | 'current' | 'policy') {
  if (!overrides[module]) overrides[module] = {}
  if (choice === 'policy') {
    delete overrides[module][id]
  } else {
    overrides[module][id] = choice
  }
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
  step, preview, applyPayload, displayValue, overrides,
  onFileSelected, setPolicy, setSettingsChoice, setOverride, clickApply
})
</script>

<style scoped>
/* native file input — dark themed (raw HTML, not covered by global overrides) */
input[type="file"] {
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: 13px;
}

input[type="file"]::file-selector-button {
  background: var(--accent-primary);
  color: var(--bg-primary);
  border: none;
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  font-family: var(--font-body);
  font-weight: 600;
  font-size: 13px;
  letter-spacing: -0.01em;
  cursor: pointer;
  margin-right: 12px;
  transition: background var(--transition-fast);
}

input[type="file"]::file-selector-button:hover {
  background: var(--accent-secondary);
}

.error {
  color: var(--accent-crimson);
  font-size: 13px;
  margin-top: 10px;
}

/* module rows: summary counts get semantic colour */
.module-row {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-color);
  gap: 10px;
}

.module-summary {
  display: flex;
  gap: 14px;
  align-items: center;
  flex: 1;
  flex-wrap: wrap;
}

.module-summary strong {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  min-width: 92px;
}

.module-summary span {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-muted);
}

.module-summary span:nth-child(2) { color: var(--accent-primary); }   /* 新增 */
.module-summary span:nth-child(3) { color: var(--text-secondary); }   /* 相同 */
.module-summary span:nth-child(4) { color: var(--accent-crimson); }   /* 冲突 */
.module-summary span:nth-child(5) { color: var(--text-muted); }       /* 仅当前 */

/* conflict list (per-record overrides) */
.conflict-list {
  flex-basis: 100%;
  width: 100%;
  margin-top: 10px;
  padding: 12px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
}

.conflict-list summary {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--accent-crimson);
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  margin-bottom: 6px;
  list-style: none;
}

.conflict-list summary::-webkit-details-marker {
  display: none;
}

.conflict-list summary::before {
  content: '▸';
  opacity: 0.7;
}

.conflict-list[open] summary::before {
  content: '▾';
}

.conflict-list[open] summary {
  margin-bottom: 12px;
}

.conflict-item {
  padding: 10px 0;
  border-top: 1px dashed var(--border-color);
}

.conflict-item:first-child {
  border-top: none;
}

.conflict-id {
  font-family: var(--font-mono);
  font-weight: 500;
  font-size: 12px;
  color: var(--text-primary);
  margin-bottom: 6px;
  letter-spacing: 0.02em;
}

.conflict-fields {
  margin: 0 0 8px 4px;
  padding: 0;
  font-size: 12px;
  color: var(--text-secondary);
  list-style: none;
}

.conflict-fields li {
  padding: 2px 0;
  font-family: var(--font-mono);
  letter-spacing: 0.02em;
}

.conflict-fields li::before {
  content: '→ ';
  color: var(--accent-crimson);
  opacity: 0.7;
}

.conflict-item :deep(.el-radio-group) {
  margin-top: 2px;
}

.conflict-item :deep(.el-radio__label) {
  font-size: 12px;
}

/* per-record override radio semantics:
   policy = skip (muted), current = keep (sage), file = overwrite (amber via global) */
:deep(.opt-policy .el-radio__input.is-checked .el-radio__inner) {
  background-color: var(--bg-elevated) !important;
  border-color: var(--border-strong) !important;
}

:deep(.opt-policy .el-radio__input.is-checked .el-radio__inner::after) {
  background-color: var(--text-muted) !important;
}

:deep(.opt-policy .el-radio__input.is-checked + .el-radio__label) {
  color: var(--text-muted) !important;
}

:deep(.opt-current .el-radio__input.is-checked .el-radio__inner) {
  background-color: var(--accent-sage) !important;
  border-color: var(--accent-sage) !important;
}

:deep(.opt-current .el-radio__input.is-checked + .el-radio__label) {
  color: var(--accent-sage) !important;
}

/* settings diff */
.settings-diff {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.diff-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  padding: 8px 0;
}

.diff-field {
  width: 180px;
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.section-label {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 15px;
  font-weight: 440;
  color: var(--text-primary);
  margin-bottom: 10px;
  letter-spacing: -0.01em;
}
</style>
