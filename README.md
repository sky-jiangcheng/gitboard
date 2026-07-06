# Git Dashboard

自动发现本地所有 Git 仓库，以可视化 Web 面板独立展示每个项目的每日代码提交量。

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![PWA](https://img.shields.io/badge/PWA-Ready-5A0FC8?logo=pwa)](https://web.dev/progressive-web-apps/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

## 截屏预览

![仪表盘首页](screenshots/dashboard.png)

![项目详情](screenshots/project-detail.png)

## 功能特性

| 特性 | 说明 |
|------|------|
| 自动发现仓库 | 设置扫描根目录后递归发现所有 Git 仓库，平台自适应默认规则 |
| 可视化仪表盘 | 每个项目独立卡片展示新增/删除/净增行数，趋势折线图 |
| 智能项目分组 | 自动识别 Monorepo 与单仓库，支持手动调整目录级别 |
| 工作日检查 | 自定义每日代码量标准，未达标时面板告警提醒 |
| 跨平台单文件 | Go 编译为单个二进制，无运行时依赖，双击即用 |
| PWA 可安装 | 支持安装到桌面/主屏幕，获得原生应用体验 |

## 快速开始

### 下载安装

从 [Releases](https://github.com/sky-jiangcheng/CodeStat/releases) 下载对应平台的最新版本。

| 平台 | 一键安装 |
|------|---------|
| macOS / Linux | `curl -fsSL https://raw.githubusercontent.com/sky-jiangcheng/CodeStat/master/scripts/install.sh \| bash` |
| Windows | `iwr -useb https://raw.githubusercontent.com/sky-jiangcheng/CodeStat/master/scripts/install.ps1 \| iex` |

### 手动使用

```bash
# 下载后赋予执行权限
chmod +x git-dashboard

# 直接运行
./git-dashboard
```

启动后自动打开浏览器访问 `http://localhost:18731`，进入仪表盘。

### 配置

首次启动使用平台默认扫描规则：

| 平台 | 默认扫描范围 |
|------|-------------|
| Windows | 除系统盘(C:)外的所有磁盘根目录 |
| macOS | 当前用户 HOME 目录 |
| Linux | 当前用户 HOME 目录 |

在设置页面可修改扫描目录、代码量标准（默认 500 行/工作日）、扫描深度等。

## 从源码构建

```bash
# 安装依赖
cd web && npm install && cd ..

# 构建前端
cd web && npm run build && cd ..

# 编译 Go 二进制
go build -ldflags="-s -w" -o git-dashboard .

# 或使用构建脚本
bash scripts/build.sh
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go + net/http + SQLite (modernc.org/sqlite, 零 CGO) |
| 前端 | React 18 + TypeScript + Vite + Chart.js |
| PWA | vite-plugin-pwa + Workbox |
| 构建 | GitHub Actions 自动发布 Win/Mac/Linux 二进制 |

## 项目结构

```
├── main.go              # Go 程序入口
├── internal/
│   ├── platform/        # 平台检测、浏览器打开、默认扫描规则
│   ├── db/              # SQLite 初始化和数据访问层
│   ├── scanner/         # Git 仓库递归扫描引擎
│   ├── stats/           # git log 统计引擎
│   ├── grouper/         # 智能项目分组引擎
│   └── server/          # HTTP API 服务和 REST 端点
├── web/                 # React SPA 前端
│   └── src/
│       ├── pages/       # Dashboard / ProjectDetail / Settings
│       ├── components/  # ProjectCard / SummaryBar / DatePicker / TrendChart
│       └── api/         # API 客户端封装
├── docs/                # GitHub Pages 落地页
├── scripts/             # 构建和安装脚本
└── .github/workflows/   # CI/CD 自动构建发布
```

## License

MIT
