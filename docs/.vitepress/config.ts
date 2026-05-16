import { defineConfig } from 'vitepress'

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
  themeConfig: {
    search: {
      provider: 'local'
    },
    nav: [
      { text: 'Guide', link: '/' },
      { text: 'Operations', link: '/operations/java-profiling-runbook' },
      { text: 'Architecture', link: '/architecture/java-profiler-architecture' },
      { text: 'Research', link: '/research/coroot-node-agent-java-agent' }
    ],
    sidebar: [
      {
        text: 'Start Here',
        items: [
          { text: 'Overview', link: '/' },
          { text: 'Requirements', link: '/brainstorms/java-profiler-requirements' }
        ]
      },
      {
        text: 'Architecture',
        items: [
          { text: 'System Architecture', link: '/architecture/java-profiler-architecture' },
          { text: 'Ingestion Review', link: '/architecture/performance-ingestion-architecture-review' }
        ]
      },
      {
        text: 'Operations',
        items: [
          { text: 'Deployment Manual', link: '/operations/deployment-operations-admin-manual' },
          { text: 'E2E Automation Guide', link: '/operations/e2e-automation-test-guide' },
          { text: 'Profiling Runbook', link: '/operations/java-profiling-runbook' },
          { text: 'Performance Analysis Manual', link: '/operations/performance-analysis-user-manual' },
          { text: 'Real Profiling Acceptance', link: '/operations/real-profiling-acceptance-standard' }
        ]
      },
      {
        text: 'Research',
        items: [
          { text: 'Coroot Node Agent Java Agent', link: '/research/coroot-node-agent-java-agent' },
          { text: 'chDB Go', link: '/research/chdb-go' },
          { text: 'Pyroscope UI Study', link: '/research/pyroscope-profile-ui-study' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'Profiling Contracts', link: '/reference/profiling-contracts' }
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/koolay/java-profiler' }
    ],
    editLink: {
      pattern: 'https://github.com/koolay/java-profiler/edit/main/docs/:path',
      text: 'Edit this page on GitHub'
    },
    footer: {
      message: 'Java services on Kubernetes. HotSpot first. async-profiler first.',
      copyright: 'Released from the java-profiler repository.'
    }
  }
})
