# 本地化策略

`java-profiler` 文档以英文为 source language。中文文档覆盖最重要的用户路径和贡献者路径，不追求把所有内部材料都翻译一遍。

## 必须双语的页面

这些页面需要同时提供英文和中文：

| English | 中文 |
| --- | --- |
| `/` | `/zh/` |
| `/getting-started/quickstart` | `/zh/getting-started/quickstart` |
| `/operations/performance-analysis-user-manual` | `/zh/operations/performance-analysis-user-manual` |
| `/contributing/development` | `/zh/contributing/development` |
| `/reference/profiling-contracts` | `/zh/reference/profiling-contracts` |

## 只保留英文的页面

实现细节重、访问频率低、或偏开发过程的材料默认只保留英文：

- 架构细节。
- Ingestion 架构 review。
- E2E 自动化细节。
- 真实 profiling 验收标准。
- Research notes。
- Brainstorms 和项目历史材料。

Research、brainstorms、plans 不进入公开导航。

## 翻译流程

修改必须双语的页面时：

1. 先改英文页面。
2. 同一个变更里同步中文页面。
3. 中文路径保持 `/zh/` 下的同构 route。
4. 截图默认复用，除非图片里的语言会让用户误解。
5. 发布前构建文档站。

```bash
cd docs
npm run docs:build
```

## 风格

- 使用清楚的技术中文，不做逐句硬翻。
- 产品名、API key、annotation、profile type、文件路径保持原样。
- UI 里显示的术语保留英文，例如 `Top Table`、`Flame Graph`、`Self CPU`、`Total CPU`。
- 不要只在中文页添加新的产品承诺。如果这个说法重要，先加到英文页。
