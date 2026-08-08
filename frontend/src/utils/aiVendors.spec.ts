import { describe, it, expect } from 'vitest'
import { VENDOR_PRESETS, getVendorPreset, DEFAULT_VENDOR } from './aiVendors'

describe('aiVendors registry', () => {
  it('exposes the 8 vendor presets in order', () => {
    expect(VENDOR_PRESETS.map((v) => v.id)).toEqual([
      'openai',
      'anthropic',
      'deepseek',
      'qwen',
      'zhipu',
      'moonshot',
      'minimax',
      'custom',
    ])
  })

  it('has unique ids', () => {
    const ids = VENDOR_PRESETS.map((v) => v.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('every preset except custom has a non-empty baseURL and at least one model', () => {
    for (const v of VENDOR_PRESETS) {
      if (v.id === 'custom') {
        expect(v.baseURL).toBe('')
        expect(v.models).toEqual([])
        continue
      }
      expect(v.baseURL.length).toBeGreaterThan(0)
      expect(v.models.length).toBeGreaterThan(0)
    }
  })

  it('routes MiniMax through the anthropic protocol with the documented base URL', () => {
    const mm = getVendorPreset('minimax')
    expect(mm?.protocol).toBe('anthropic')
    expect(mm?.baseURL).toBe('https://api.minimaxi.com/anthropic')
  })

  it('routes DeepSeek through the openai protocol', () => {
    expect(getVendorPreset('deepseek')?.protocol).toBe('openai')
  })

  it('returns undefined for unknown vendor ids', () => {
    expect(getVendorPreset('nope')).toBeUndefined()
  })

  it('defaults to openai', () => {
    expect(DEFAULT_VENDOR).toBe('openai')
  })
})
