# GitBoard 项目文档

GitBoard 是一个本地 Git 代码量统计与项目看板桌面应用。本文档涵盖系统架构、API 接口、开发指南和领域概念。

**快速链接**: [架构](./ARCHITECTURE.md) | [接口](./INTERFACES.md) | [开发者指南](./DEVELOPER_GUIDE.md)

---

## 核心文档

### [架构](./ARCHITECTURE.md)
系统设计、技术栈、项目结构、子系统和数据流。从这里开始了解系统如何运作。

### [接口](./INTERFACES.md)
20 个 Wails Bind 方法的完整参考，包含请求参数、返回类型和调用示例。

### [开发者指南](./DEVELOPER_GUIDE.md)
环境搭建、开发工作流、编码规范和常见任务（添加 Bind 方法、数据库表、前端页面）。

---

## 模块

| 模块 | 描述 | 文档 |
|------|------|--------|
| `internal/db/` | SQLite 数据库管理、建表、CRUD | [README](./模块/db.md) |
| `internal/scanner/` | 文件系统递归扫描 Git 仓库 | [README](./模块/scanner.md) |
| `internal/grouper/` | 仓库按目录层级自动分组 | [README](./模块/grouper.md) |
| `internal/stats/` | git log 统计查询 + 日期工具 | [README](./模块/stats.md) |
| `internal/platform/` | 跨平台路径/浏览器/用户工具 | 见源码注释 |
| `web/` | React 18 前端（页面/组件/样式） | [README](./模块/web.md) |

---

## 核心概念

| 概念 | 描述 |
|------|------|
| [Project](./专有概念/Project.md) | 多个 Git 仓库聚合而成的逻辑分组节点 |
| [Todo](./专有概念/Todo.md) | 关联到项目的待办事项，支持完成/排序 |
| [Note](./专有概念/Note.md) | 关联到项目的 Markdown 笔记 |

---

## 入门指南

### 项目新人？

按此路径学习：
1. **[架构](./ARCHITECTURE.md)** — 了解技术栈和系统结构
2. **[核心概念](#核心概念)** — 学习 Project/Todo/Note 等术语
3. **[开发者指南](./DEVELOPER_GUIDE.md)** — 搭建开发环境
4. **[接口](./INTERFACES.md)** — 探索所有 Bind 方法

## 快速参考

### 命令

```bash
wails dev          # 桌面应用开发模式
wails build        # 生产二进制构建
cd web && npm run dev  # 前端独立开发
go test ./...      # 运行所有 Go 测试
cd web && npm run build # 前端生产构建
```

### 测试统计

| 包 | 测试数 |
|----|--------|
| `internal/db` | 14 |
| `internal/grouper` | 7 |
| `internal/stats` | 11 |
| `internal/platform` | 10 |
| `app.go` (集成) | 10 |
| **总计** | **52** |
