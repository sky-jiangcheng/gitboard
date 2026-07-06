# 需求实施计划

- [x] 1. 初始化 Go 项目结构和模块
  - 创建 `go.mod`，设置 module 名称为 `git-dashboard`
  - 创建 `main.go` 入口文件，定义程序启动骨架
  - 创建 `internal/` 目录下的子包目录结构：`server/`, `scanner/`, `stats/`, `grouper/`, `db/`, `platform/`
  - 创建 `scripts/` 目录
  - 对应需求: R1 (扫描范围配置), R8 (跨平台支持)

- [x] 2. 实现平台工具层 (internal/platform)
  - [x] 2.1 实现平台检测函数 `DetectOS()`
  - [x] 2.2 实现默认扫描根目录获取函数 `DefaultScanRoots()`
  - [x] 2.3 实现浏览器打开函数 `OpenBrowser(url string)`
  - [x] 2.4 实现 Git 命令可用性检测 `CheckGitInstalled() bool`
  - [ ]* 2.5 为平台工具层编写单元测试

- [x] 3. 实现数据库层 (internal/db)
  - [x] 3.1 实现 SQLite 数据库初始化 `InitDB(dbPath string) (*sql.DB, error)`
  - [x] 3.2 实现配置存取函数
  - [x] 3.3 实现 ScanRoots 数据访问函数
  - [x] 3.4 实现 Projects 数据访问函数
  - [x] 3.5 实现 Repositories 数据访问函数
  - [x] 3.6 实现 DailyStats 数据访问函数
  - [ ]* 3.7 为数据库层编写单元测试

- [x] 4. 检查点 — 数据库层编译通过，表结构正确创建

- [x] 5. 实现扫描引擎 (internal/scanner)
  - [x] 5.1 实现 `ScanRepositories(roots []string, maxDepth int) ([]RepoInfo, error)`
  - [ ]* 5.2 为扫描引擎编写单元测试

- [x] 6. 实现统计引擎 (internal/stats)
  - [x] 6.1 实现 `Result` 结构体和 `QueryStats(repoPath, date, author string) (*Result, error)`
  - [x] 6.2 实现 `QueryMultiBranch(repoPath, date string, branches []string) (*Result, error)`
  - [ ]* 6.3 为统计引擎编写单元测试

- [x] 7. 实现分组引擎 (internal/grouper)
  - [x] 7.1 实现 `GroupRepositories(repos []RepoInfo) []ProjectGroup`
  - [x] 7.2 实现 `AdjustProjectLevelUp` / `AdjustProjectLevelDown`
  - [ ]* 7.3 为分组引擎编写单元测试

- [x] 8. 检查点 — 后端核心引擎（扫描、统计、分组）编译通过

- [x] 9. 实现 HTTP API 层 (internal/server)
  - [x] 9.1 实现 HTTP 服务器初始化 `NewServer(db *sql.DB) *Server`
  - [x] 9.2 实现全部 API 处理函数
  - [x] 9.3 实现扫描时自动统计逻辑
  - [ ]* 9.4 为 API 处理函数编写 HTTP 测试

- [x] 10. 实现程序主入口 (main.go)
  - [x] 10.1 启动流程编排
  - [x] 10.2 实现 `//go:embed` 嵌入前端构建产物
  - [x] 10.3 实现优雅退出

- [x] 11. 初始化 React 前端项目 (web/)
  - [x] 11.1 使用 Vite 创建 React + TypeScript 项目
  - [x] 11.2 创建 API 客户端封装 (`web/src/api/client.ts`)

- [x] 12. 实现前端仪表盘页面 (Dashboard)
  - [x] 12.1 实现 `SummaryBar` 组件
  - [x] 12.2 实现 `DatePicker` 组件
  - [x] 12.3 实现 `ProjectCard` 组件
  - [x] 12.4 实现 `Dashboard` 页面

- [x] 13. 实现前端项目详情页面 (ProjectDetail)
  - [x] 13.1 实现 `TrendChart` 组件
  - [x] 13.2 实现 `ProjectDetail` 页面

- [x] 14. 实现前端设置页面 (Settings)
  - [x] 14.1 实现扫描根目录管理
  - [x] 14.2 实现代码量标准和扫描深度配置

- [x] 15. 实现前端路由和导航
  - [x] 15.1 设置 React Router 路由
  - [x] 15.2 实现顶部导航栏

- [x] 16. 实现跨平台构建脚本 (scripts/build.sh)
  - [x] 16.1 构建脚本
  - [ ]* 16.2 编写 Makefile 辅助构建

- [x] 17. 检查点 — 端到端验证
  - 启动应用，验证浏览器能打开面板
  - 扫描当前工作区 Git 仓库，验证统计结果正确
