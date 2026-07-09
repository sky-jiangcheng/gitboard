# 前端模块 (web/)

React 18 + TypeScript 前端，负责 UI 渲染和用户交互。

## 结构

```
web/src/
├── api/client.ts              # 双模 API 客户端（Wails / HTTP）
├── App.tsx                    # 路由 + 导航栏
├── main.tsx                   # React 入口
├── pages/
│   ├── Dashboard.tsx          # 仪表盘（排序/日期/扫描/Todo 计数）
│   ├── ProjectDetail.tsx      # 项目详情（左右分栏）
│   └── Settings.tsx           # 设置（4 个 Tab 分区）
├── components/
│   ├── DatePicker.tsx         # 日期选择（昨天/今天/自定义）
│   ├── ProjectCard.tsx        # 项目卡片（含 Todo 徽标）
│   ├── ProjectPanel.tsx       # 右侧面板容器
│   ├── SummaryBar.tsx         # 6 指标摘要条
│   ├── TodoSection.tsx        # Todo 面板
│   ├── NoteSection.tsx        # 笔记面板（Markdown）
│   └── TrendChart.tsx         # 多线趋势图
└── styles/global.css          # 全局样式
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `api/client.ts` | 20 个 API 方法，自动双模切换 |
| `App.tsx` | HashRouter 路由 + active 导航高亮 |
| `Dashboard.tsx` | 核心页面，加载项目/Todo 计数，排序和日期过滤 |
| `ProjectDetail.tsx` | 左右分栏布局，左侧原有内容 + 右侧 Todo/笔记面板 |

## 依赖

**本模块依赖**: React 18, React Router 6, Chart.js 4, marked

**被依赖**: Wails 框架 embed `web/dist/` 目录

## 技术决策

- **无 UI 框架**: 纯 CSS 实现所有样式，约 900 行
- **双模 API**: `window.go.main.App` 存在时走 Wails Bind，否则 fetch HTTP
- **骨架屏**: CSS animation shimmer 替代传统 spinner
- **Markdown 渲染**: `marked` 库（~30KB minified），轻量无额外编辑器依赖
