# 需求实施计划

- [x] 1. 数据库层：新增 Todo 和笔记表
  - [x] 1.1 在 `internal/db/db.go` 的 createTables 中添加 `project_todos` 和 `project_notes` 的 CREATE TABLE DDL
  - [x] 1.2 在 `internal/db/queries.go` 中添加 Todo CRUD 函数
  - [x] 1.3 在 `internal/db/queries.go` 中添加 Note CRUD 函数
  - [x] 1.4 在 `internal/db/queries.go` 中添加 `GetTodoCounts() ([]TodoCount, error)` 查询函数
  - [x] 1.5 为数据库查询编写单元测试

- [x] 2. 后端：新增 Wails Bind 方法
  - [x] 2.1 在 `app.go` 中添加 Todo 相关的 5 个 Bind 方法
  - [x] 2.2 在 `app.go` 中添加 Note 相关的 4 个 Bind 方法
  - [x] 2.3 在 `app.go` 中添加 `GetTodoCounts` Bind 方法
  - [x] 2.4 为 Bind 方法编写集成测试

- [x] 3. 检查点：Go 编译和现有测试

- [x] 4. 前端 API 客户端：添加 Todo 和笔记接口
  - [x] 4.1 在 `web/src/api/client.ts` 中添加 Todo、Note、TodoCount 类型定义
  - [x] 4.2 添加 10 个 API 方法（双模：Wails Bind / HTTP）

- [x] 5. 前端组件：Markdown 依赖和 Todo 组件
  - [x] 5.1 安装 `marked` 依赖
  - [x] 5.2 创建 `web/src/components/TodoSection.tsx`
  - [x] 5.3 创建 `web/src/components/NoteSection.tsx`

- [x] 6. 前端组件：面板容器和页面集成
  - [x] 6.1 创建 `web/src/components/ProjectPanel.tsx`
  - [x] 6.2 更新 `web/src/pages/ProjectDetail.tsx` 为左右分栏布局
  - [x] 6.3 更新 `web/src/components/ProjectCard.tsx`
  - [x] 6.4 更新 `web/src/components/SummaryBar.tsx`
  - [x] 6.5 更新 `web/src/pages/Dashboard.tsx`

- [x] 7. 全局样式：面板和编辑区域样式
  - [x] 7.1 在 `web/src/styles/global.css` 中添加右侧面板样式

- [x] 8. 检查点：前端构建和最终验证