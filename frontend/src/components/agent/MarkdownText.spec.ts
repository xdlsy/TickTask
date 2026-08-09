import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import MarkdownText from './MarkdownText.vue'

describe('MarkdownText', () => {
  beforeEach(() => {
    // jsdom has no clipboard by default
    Object.defineProperty(navigator, 'clipboard', {
      writable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) }
    })
  })

  it('renders bold + list as real HTML', () => {
    const w = mount(MarkdownText, { props: { content: '**hi**\n- a\n- b' } })
    const html = w.html()
    expect(html).toContain('<strong>hi</strong>')
    expect(html).toContain('<ul>')
  })

  it('strips an XSS payload', () => {
    const w = mount(MarkdownText, { props: { content: '<img src=x onerror="alert(1)">' } })
    expect(w.html()).not.toContain('onerror')
    expect(w.html()).not.toContain('alert')
  })

  it('renders a code block with a copy button + language label', async () => {
    const w = mount(MarkdownText, { props: { content: '```bash\necho hi\n```' } })
    await w.vm.$nextTick()
    await w.vm.$nextTick() // Double tick for safety
    const btn = w.find('button.md-copy')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('bash')
  })

  it('copy button writes code text to clipboard', async () => {
    const w = mount(MarkdownText, { props: { content: '```js\nconst a = 1\n```' } })
    await w.vm.$nextTick()
    const btn = w.find('button.md-copy')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('js')
    await btn.trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('const a = 1\n')
  })
})
