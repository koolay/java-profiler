import { defineConfig } from 'vitepress'

const enNav = [
  { text: 'Get Started', link: '/getting-started/quickstart' },
  { text: 'Users', link: '/operations/performance-analysis-user-manual' },
  { text: 'Operators', link: '/operations/deployment-operations-admin-manual' },
  { text: 'Contributors', link: '/contributing/development' },
  { text: 'Architecture', link: '/architecture/java-profiler-architecture' },
  { text: 'Reference', link: '/reference/profiling-contracts' }
]

const zhNav = [
  { text: '快速开始', link: '/zh/getting-started/quickstart' },
  { text: '用户', link: '/zh/operations/performance-analysis-user-manual' },
  { text: '运维', link: '/operations/deployment-operations-admin-manual' },
  { text: '贡献者', link: '/zh/contributing/development' },
  { text: '架构', link: '/architecture/java-profiler-architecture' },
  { text: '参考', link: '/zh/reference/profiling-contracts' }
]

const enSidebar = [
  {
    text: 'Start Here',
    items: [
      { text: 'Overview', link: '/' },
      { text: 'Quickstart', link: '/getting-started/quickstart' }
    ]
  },
  {
    text: 'Users',
    items: [
      { text: 'Analyze Performance', link: '/operations/performance-analysis-user-manual' },
      { text: 'Enable Profiling', link: '/operations/java-profiling-runbook' }
    ]
  },
  {
    text: 'Operators',
    items: [
      { text: 'Deployment Manual', link: '/operations/deployment-operations-admin-manual' },
      { text: 'Real Profiling Acceptance', link: '/operations/real-profiling-acceptance-standard' },
      { text: 'E2E Automation Guide', link: '/operations/e2e-automation-test-guide' }
    ]
  },
      {
        text: 'Contributors',
        items: [
          { text: 'Development Setup', link: '/contributing/development' },
          { text: 'Localization', link: '/contributing/localization' },
          { text: 'System Architecture', link: '/architecture/java-profiler-architecture' },
          { text: 'Ingestion Architecture', link: '/architecture/performance-ingestion-architecture-review' }
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
    text: '开始',
    items: [
      { text: '概览', link: '/zh/' },
      { text: '快速开始', link: '/zh/getting-started/quickstart' }
    ]
  },
  {
    text: '用户',
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
        text: '贡献者',
        items: [
          { text: '开发设置', link: '/zh/contributing/development' },
          { text: '本地化策略', link: '/zh/contributing/localization' },
          { text: '系统架构', link: '/architecture/java-profiler-architecture' },
          { text: 'Ingestion 架构', link: '/architecture/performance-ingestion-architecture-review' }
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
  description: 'Java performance profiling for Kubernetes with async-profiler and ClickHouse',
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
  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      title: 'Java Profiler',
      description: 'Java performance profiling for Kubernetes with async-profiler and ClickHouse',
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
      description: '面向 Kubernetes Java 服务的真实性能 Profiling 文档',
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
