<template>
  <div ref="root" class="md" data-testid="md" v-html="html" />
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps<{ content: string }>()
const root = ref<HTMLElement | null>(null)

const html = computed(() => {
  const raw = marked.parse(props.content ?? '', {
    gfm: true, breaks: true,
  }) as string
  return DOMPurify.sanitize(raw, { USE_PROFILES: { html: true } })
})

// Inject a copy button + language label into each <pre><code> after every render.
function wireCodeBlocks() {
  const el = root.value
  if (!el) return

  // Handle both <pre><code> blocks and standalone <code class="language-*"> blocks
  const codeBlocks = el.querySelectorAll('pre code, code[class*="language-"]')

  codeBlocks.forEach((code) => {
    // Skip if this code block already has a button
    const parent = code.parentElement
    if (!parent) return
    if (parent.tagName === 'PRE' && parent.querySelector('button.md-copy')) return
    if (parent.querySelector('.md-copy-wrapper')) return

    const lang = (code.className.match(/language-([\w-]+)/) || [])[1] || ''
    const btn = document.createElement('button')
    btn.type = 'button'
    btn.className = 'md-copy'
    btn.textContent = lang ? `${lang} ⧉` : '⧉'
    btn.addEventListener('click', () => {
      navigator.clipboard?.writeText(code.textContent || '')
    })

    // If wrapped in pre, append to pre; otherwise wrap code in a container
    if (parent.tagName === 'PRE') {
      parent.appendChild(btn)
    } else {
      const wrapper = document.createElement('div')
      wrapper.className = 'md-copy-wrapper'
      wrapper.style.position = 'relative'
      wrapper.style.display = 'inline-block'
      wrapper.style.width = '100%'
      code.parentNode?.insertBefore(wrapper, code)
      wrapper.appendChild(code)
      wrapper.appendChild(btn)
    }
  })
}

watch(html, () => nextTick(wireCodeBlocks), { immediate: true })
</script>

<style scoped>
.md {
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.65;
  color: var(--text-primary);
  word-wrap: break-word;
}
.md :deep(h1),
.md :deep(h2),
.md :deep(h3) {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-weight: 500;
  letter-spacing: -0.01em;
  color: var(--text-primary);
  margin: 12px 0 6px;
  line-height: 1.3;
}
.md :deep(h1) { font-size: 18px; }
.md :deep(h2) { font-size: 16px; }
.md :deep(h3) { font-size: 14px; }
.md :deep(p) { margin: 0 0 8px; }
.md :deep(p:last-child) { margin-bottom: 0; }
.md :deep(strong) { color: var(--accent-secondary); font-weight: 600; }
.md :deep(em) { color: var(--text-secondary); }
.md :deep(ul),
.md :deep(ol) { margin: 0 0 8px; padding-left: 18px; }
.md :deep(li) { margin: 3px 0; }
.md :deep(ol li::marker) { font-family: var(--font-mono); color: var(--text-muted); font-size: 11px; }
.md :deep(a) { color: var(--accent-primary); text-decoration: underline; text-underline-offset: 2px; }
.md :deep(code) {
  font-family: var(--font-mono);
  font-size: 11px;
  background: rgba(230, 162, 60, 0.12);
  color: var(--accent-secondary);
  padding: 1px 5px;
  border-radius: 4px;
}
.md :deep(pre) {
  position: relative;
  background: var(--bg-elevated);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 9px 11px;
  margin: 0 0 8px;
  overflow-x: auto;
}
.md :deep(pre > code) {
  display: block;
  background: transparent;
  color: var(--text-secondary);
  padding: 0;
  font-size: 11px;
  white-space: pre;
}
.md :deep(blockquote) {
  margin: 0 0 8px;
  padding: 4px 0 4px 11px;
  border-left: 2px solid var(--accent-primary);
  color: var(--text-secondary);
  font-family: var(--font-display);
  font-style: italic;
  font-size: 12.5px;
}
.md :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0 0 8px;
  font-size: 11.5px;
}
.md :deep(th),
.md :deep(td) {
  text-align: left;
  padding: 4px 7px;
  border-bottom: 1px solid var(--border-color);
}
.md :deep(th) {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
  font-weight: 400;
}
.md :deep(td) { color: var(--text-primary); }
.md :deep(hr) { border: 0; border-top: 1px solid var(--border-color); margin: 10px 0; }
:deep(.md-copy) {
  position: absolute;
  top: 4px;
  right: 4px;
  font-family: var(--font-mono);
  font-size: 9px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 1px 6px;
  cursor: pointer;
}
:deep(.md-copy:hover) { color: var(--text-primary); }
</style>
