import { h, nextTick, watch } from 'vue'
import DefaultTheme from 'vitepress/theme'
import { useData, useRoute } from 'vitepress'
import { createMermaidRenderer } from 'vitepress-mermaid-renderer'
import './style.css'

let mermaidRenderer: ReturnType<typeof createMermaidRenderer> | null = null

export default {
  extends: DefaultTheme,
  Layout: () => {
    const { isDark } = useData()
    const route = useRoute()

    const initMermaid = () => {
      const theme = isDark.value ? 'dark' : 'forest'

      if (mermaidRenderer?.setConfig) {
        mermaidRenderer.setConfig({ theme })
        return
      }

      mermaidRenderer = createMermaidRenderer({
        theme,
        startOnLoad: false,
        flowchart: {
          useMaxWidth: true,
          htmlLabels: true
        },
        downloadFormat: 'svg',
        desktop: {
          zoomIn: 'enabled',
          zoomOut: 'enabled',
          resetView: 'enabled',
          copyCode: 'enabled',
          download: 'disabled',
          positions: { vertical: 'top', horizontal: 'right' }
        },
        mobile: {
          zoomIn: 'disabled',
          zoomOut: 'disabled',
          resetView: 'enabled',
          toggleFullscreen: 'enabled'
        }
      })
    }

    nextTick(initMermaid)
    watch(() => isDark.value, () => nextTick(initMermaid), { immediate: true })
    watch(() => route.path, () => nextTick(initMermaid))

    return h(DefaultTheme.Layout)
  }
}
