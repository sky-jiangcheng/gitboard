# GitBoard 开发者指南

## 项目目的

GitBoard 是一个本地 Git 代码量统计与项目看板工具。它帮助开发者在独立桌面应用中追踪个人和团队的每日代码产出、管理项目待办事项和笔记。

## 环境搭建

### 前置条件

- Go >= 1.24
- Node.js >= 18 + npm
- Git CLI（用于统计查询）
- Linux: `webkit2gtk-4.1-dev`（Wails WebView 运行时）
- macOS: Xcode Command Line Tools
- Windows: WebView2 运行时（Windows 11 自带）

### 安装

```bash
# 克隆仓库
git clone https://github.com/sky-jiangcheng/gitboard
cd gitboard

# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0

# 安装前端依赖
cd web && npm install

# 返回项目根
cd ..

# 开发模式运行
wails dev

# 或仅启动前端开发服务器（需要后端 HTTP fallback）
cd web && npm run dev
```

### 运行

```bash
# 开发（Wails 桌面应用，自动编译前后端）
wails dev

# 构建生产二进制
wails build

# 前端开发服务器（独立运行）
cd web && npm run dev

# 前端生产构建
cd web && npm run build

# 运行测试
go test ./...
```

## 开发工作流

### 代码质量工具

| 工具 | 命令 | 目的 |
|------|------|------|
| Go 编译 | `go build ./...` | 编译检查 |
| Go 测试 | `go test ./...` | 单元/集成测试 |
| TypeScript | `npx tsc --noEmit` | 类型检查 |
| Vite | `npm run build` | 前端构建 + PWA |

### 项目约定

- Go 代码使用 tab 缩进
- TypeScript 使用 2 空格缩进
- 注释使用简体中文
- 函数/类型首字母大写为公开，小写为私有
- Go 错误处理使用 `fmt.Errorf` 包装

### 分支策略

- `master` — 主分支，发布就绪代码
- 功能在 master 上直接开发，tag 标记发布版本

## 常见任务

### 添加新的 Bind 方法（Go → 前端 API）

**需修改的文件**:
1. `internal/db/queries.go` — 添加数据库查询函数
2. `app.go` — 添加公开方法（首字母大写）
3. `web/src/api/client.ts` — 添加双模（Wails/HTTP）API 方法
4. 可选: `app_test.go` — 集成测试

**步骤**:
1. 在 db 层实现 CRUD 函数
2. 在 app.go 中添加 Bind 方法，调用 db 层函数
3. 在 client.ts 中添加类型和方法
4. 编译验证: `go build ./... && cd web && npm run build`

### 添加新数据库表

**需修改的文件**:
1. `internal/db/db.go` — 在 `createTables()` 中添加 DDL
2. `internal/db/queries.go` — 添加 CRUD 函数
3. 可选: `internal/db/queries_test.go` — 单元测试

**DDL 示例**:
```sql
CREATE TABLE IF NOT EXISTS new_table (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 添加新前端页面

**需修改的文件**:
1. `web/src/pages/NewPage.tsx` — 新页面组件
2. `web/src/App.tsx` — 添加路由
3. `web/src/styles/global.css` — 样式

**步骤**:
1. 创建页面组件
2. 在 App.tsx 的 Routes 中添加 `<Route path="/new" element={<NewPage />} />`
3. 在导航栏添加链接
4. 构建验证: `cd web && npm run build`

## 编码规范

### 文件组织
- Go: 每个包一个职责（db、scanner、grouper、stats、platform）
- 前端: 页面放 `pages/`，可复用组件放 `components/`
- 测试: 与源码同目录，`_test.go` 或 `.test.tsx`

### 命名

| 类型 | Go | TypeScript |
|------|----|-----------|
| 文件名 | snake_case | PascalCase (组件) / camelCase |
| 类型/接口 | PascalCase | PascalCase |
| 函数/方法 | PascalCase (公开) / camelCase (私有) | camelCase |
| 测试文件 | *_test.go | *.test.tsx |

### API 客户端双模模式

```typescript
// 自动检测运行环境
const isWails = (): boolean =>
  typeof window !== 'undefined' && !!(window as any).go?.main?.App

// Wails 模式：直接调用 Go 方法
function wail<T>(method: string, ...args: any[]): Promise<T> {
  return (window as any).go.main.App[method](...args)
}

// HTTP 模式：fetch API（独立前端开发）
async function http<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch('/api' + url, options)
  return res.json()
}
```
