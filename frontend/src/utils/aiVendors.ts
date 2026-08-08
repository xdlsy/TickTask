/**
 * AI vendor preset registry — the frontend source of truth for the
 * 「AI 智能助手」provider dropdown. Each preset carries the display label,
 * the backend protocol it routes through (see backend protocolFor), the
 * default base URL, and a set of preset model names. The model <el-select>
 * stays allow-create/filterable so users can type any model name.
 *
 * BaseURL convention (must match the backend clients in
 * backend/internal/ai/client.go):
 *   - OpenAI-protocol vendors: baseURL INCLUDES the version path; the backend
 *     appends `/chat/completions`.
 *   - Anthropic-protocol vendors: baseURL does NOT include `/v1`; the backend
 *     appends `/v1/messages`.
 *
 * NOTE: base URLs / model names for DeepSeek/Qwen/Zhipu/Moonshot are
 * best-effort; verify against each vendor's current docs before relying on
 * them. MiniMax values are confirmed from the official Anthropic-compat docs.
 */
export type AIProtocol = 'openai' | 'anthropic'

export interface VendorPreset {
  id: string
  label: string
  protocol: AIProtocol
  baseURL: string
  models: string[]
}

export const VENDOR_PRESETS: VendorPreset[] = [
  {
    id: 'openai',
    label: 'OpenAI',
    protocol: 'openai',
    baseURL: 'https://api.openai.com/v1',
    models: ['gpt-4o-mini', 'gpt-4o', 'gpt-4.1-mini', 'gpt-4.1'],
  },
  {
    id: 'anthropic',
    label: 'Anthropic',
    protocol: 'anthropic',
    baseURL: 'https://api.anthropic.com',
    models: ['claude-sonnet-4-6', 'claude-3-5-sonnet-latest', 'claude-3-5-haiku-latest'],
  },
  {
    id: 'deepseek',
    label: 'DeepSeek',
    protocol: 'openai',
    baseURL: 'https://api.deepseek.com/v1',
    models: ['deepseek-chat', 'deepseek-reasoner'],
  },
  {
    id: 'qwen',
    label: '通义千问',
    protocol: 'openai',
    baseURL: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    models: ['qwen-plus', 'qwen-max', 'qwen-turbo', 'qwen-long'],
  },
  {
    id: 'zhipu',
    label: '智谱 GLM',
    protocol: 'openai',
    baseURL: 'https://open.bigmodel.cn/api/paas/v4',
    models: ['glm-4-plus', 'glm-4-air', 'glm-4-flash', 'glm-4.5'],
  },
  {
    id: 'moonshot',
    label: '月之暗面 Kimi',
    protocol: 'openai',
    baseURL: 'https://api.moonshot.cn/v1',
    models: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k', 'kimi-k2-0905-preview'],
  },
  {
    id: 'minimax',
    label: 'MiniMax',
    protocol: 'anthropic',
    baseURL: 'https://api.minimaxi.com/anthropic',
    models: ['MiniMax-M3', 'MiniMax-M2.7', 'MiniMax-M2.7-highspeed', 'MiniMax-M2.5'],
  },
  {
    id: 'custom',
    label: '自定义',
    protocol: 'openai',
    baseURL: '',
    models: [],
  },
]

export const DEFAULT_VENDOR = 'openai'

/** Look up a preset by vendor id. Returns undefined for unknown ids. */
export function getVendorPreset(id: string): VendorPreset | undefined {
  return VENDOR_PRESETS.find((v) => v.id === id)
}
