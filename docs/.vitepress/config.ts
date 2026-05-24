import { defineConfig } from 'vitepress'

const enNav = [
  { text: 'Get Started', link: '/getting-started/quickstart' },
  { text: 'Workflows', link: '/operations/performance-analysis-user-manual' },
  { text: 'Operations', link: '/operations/deployment-operations-admin-manual' },
  { text: 'Contributing', link: '/contributing/development' },
  { text: 'Architecture', link: '/architecture/java-profiler-architecture' },
  { text: 'Reference', link: '/reference/profiling-contracts' }
]

const zhNav = [
  { text: '快速开始', link: '/zh/getting-started/quickstart' },
  { text: '使用场景', link: '/zh/operations/performance-analysis-user-manual' },
  { text: '运维', link: '/operations/deployment-operations-admin-manual' },
  { text: '贡献者', link: '/zh/contributing/development' },
  { text: '架构', link: '/architecture/java-profiler-architecture' },
  { text: '参考', link: '/zh/reference/profiling-contracts' }
]

const enSidebar = [
  {
    text: 'Getting Started',
    items: [
      { text: 'Overview', link: '/' },
      { text: 'Quickstart', link: '/getting-started/quickstart' }
    ]
  },
  {
    text: 'Workflows',
    items: [
      { text: 'Analyze Performance', link: '/operations/performance-analysis-user-manual' },
      { text: 'Enable Profiling', link: '/operations/java-profiling-runbook' }
    ]
  },
  {
    text: 'Operations',
    items: [
      { text: 'Deployment Manual', link: '/operations/deployment-operations-admin-manual' },
      { text: 'Real Profiling Acceptance', link: '/operations/real-profiling-acceptance-standard' },
      { text: 'E2E Automation Guide', link: '/operations/e2e-automation-test-guide' }
    ]
  },
  {
    text: 'Contributing',
    items: [
      { text: 'Development Setup', link: '/contributing/development' },
      { text: 'Localization', link: '/contributing/localization' },
      { text: 'System Architecture', link: '/architecture/java-profiler-architecture' },
      { text: 'Ingestion Architecture', link: '/architecture/performance-ingestion-architecture-review' },
      { text: 'Allocation Analysis Optimization', link: '/architecture/allocation-analysis-optimization-design' }
    ]
  },
  {
    text: 'Reference',
    items: [
      { text: 'Profiling Contracts', link: '/reference/profiling-contracts' }
    ]
  }
]

const zhSidebar = [
  {
    text: '入门',
    items: [
      { text: '概览', link: '/zh/' },
      { text: '快速开始', link: '/zh/getting-started/quickstart' }
    ]
  },
  {
    text: '使用场景',
    items: [
      { text: '性能分析', link: '/zh/operations/performance-analysis-user-manual' },
      { text: '启用 Profiling', link: '/operations/java-profiling-runbook' }
    ]
  },
  {
    text: '运维',
    items: [
      { text: '部署运维手册', link: '/operations/deployment-operations-admin-manual' },
      { text: '真实 Profiling 验收', link: '/operations/real-profiling-acceptance-standard' },
      { text: 'E2E 自动化指南', link: '/operations/e2e-automation-test-guide' }
    ]
  },
  {
    text: '贡献',
    items: [
      { text: '开发设置', link: '/zh/contributing/development' },
      { text: '本地化策略', link: '/zh/contributing/localization' },
      { text: '系统架构', link: '/architecture/java-profiler-architecture' },
      { text: 'Ingestion 架构', link: '/architecture/performance-ingestion-architecture-review' },
      { text: 'Allocation 分析优化', link: '/architecture/allocation-analysis-optimization-design' }
    ]
  },
  {
    text: '参考',
    items: [
      { text: 'Profiling 合同', link: '/zh/reference/profiling-contracts' }
    ]
  }
]

export default defineConfig({
  title: 'Java Profiler',
  description: 'Java performance profiling for Kubernetes incidents: HotSpot-first opt-in collection, async-profiler evidence, ClickHouse storage, and a service-focused UI.',
  lang: 'en-US',
  base: '/java-profiler/',
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: false,
  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  },
  head: [
    ['meta', { name: 'keywords', content: 'Java Profiler, Java performance profiling, Kubernetes, async-profiler, ClickHouse, flame graph, JVM diagnostics' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'Java Profiler' }],
    ['meta', { property: 'og:description', content: 'Java performance profiling for Kubernetes incidents: HotSpot-first opt-in collection, async-profiler evidence, ClickHouse storage, and a service-focused UI.' }]
  ],
  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      title: 'Java Profiler',
      description: 'Java performance profiling for Kubernetes incidents: HotSpot-first opt-in collection, async-profiler evidence, ClickHouse storage, and a service-focused UI.',
      themeConfig: {
        nav: enNav,
        sidebar: enSidebar,
        editLink: {
          pattern: 'https://github.com/koolay/java-profiler/edit/main/docs/:path',
          text: 'Edit this page on GitHub'
        },
        footer: {
          message: 'Java services on Kubernetes. HotSpot first. async-profiler first.',
          copyright: 'Released from the java-profiler repository.'
        }
      }
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      title: 'Java Profiler',
      description: '面向 Kubernetes Java 事故排障的 Profiling 文档：HotSpot 优先的 opt-in 采集、async-profiler 证据、ClickHouse 存储，以及面向服务的 UI。',
      link: '/zh/',
      themeConfig: {
        nav: zhNav,
        sidebar: zhSidebar,
        editLink: {
          pattern: 'https://github.com/koolay/java-profiler/edit/main/docs/:path',
          text: '在 GitHub 上编辑本页'
        },
        footer: {
          message: '面向 Kubernetes Java 服务。HotSpot 优先。async-profiler 优先。',
          copyright: '来自 java-profiler 仓库。'
        }
      }
    }
  },
  themeConfig: {
    search: {
      provider: 'local'
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/koolay/java-profiler' }
    ]
  }
})
