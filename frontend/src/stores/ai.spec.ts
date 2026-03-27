import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAIStore } from '@/stores/ai'

describe('AI Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes as not configured', () => {
    const store = useAIStore()
    expect(store.configured).toBe(false)
  })

  it('initializes with loading false', () => {
    const store = useAIStore()
    expect(store.loading).toBe(false)
  })

  it('initializes with no last classification', () => {
    const store = useAIStore()
    expect(store.lastClassification).toBeNull()
  })
})