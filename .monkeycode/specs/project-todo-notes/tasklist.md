# 需求实施计划

- [ ] 1. 数据库层：新增 Todo 和笔记表
  - [ ] 1.1 在 `internal/db/db.go` 的 createTables 中添加 `project_todos` 和 `project_notes` 的 CREATE TABLE DDL
    - 参考需求 1：包含 id、project_id、title/content、completed、priority、sort_order、created_at、updated_at 字段
    - 外键关联 projects(id) ON DELETE CASCADE

  - [ ] 1.2 在 `internal/db/queries.go` 中添加 Todo CRUD 函数
    - `ListTodos(projectId int64) ([]Todo, error)` — 按 sort_order 排序查询
    - `CreateTodo(projectId int64, title string) (*Todo, error)` — 插入并返回新记录
    - `ToggleTodo(todoId int64) error` — 原子翻转 completed 字段并更新 updated_at
    - `DeleteTodo(todoId int64) error` — 按 ID 删除
    - `ReorderTodos(todoIds []int64) error` — 事务内批量更新 sort_order
    - 参考需求 7.1-7.5、正确性约束 3-5

  - [ ] 1.3 在 `internal/db/queries.go` 中添加 Note CRUD 函数
    - `ListNotes(projectId int64) ([]Note, error)` — 按 sort_order 排序查询
    - `CreateNote(projectId int64, content string) (*Note, error)` — 插入并返回
    - `UpdateNote(noteId int64, content string) error` — 更新 content 和 updated_at
    - `DeleteNote(noteId int64) error` — 按 ID 删除
    - 参考需求 7.6-7.9

  - [ ] 1.4 在 `internal/db/queries.go` 中添加 `GetTodoCounts() ([]TodoCount, error)` 查询函数
    - 按 project_id 分组统计未完成 Todo 数和总数
    - 参考需求 7.10

  - [ ] 1.5 为数据库查询编写单元测试
    - 测试 Todo CRUD 边界情况（空标题、不存在项目、并发排序）
    - 测试 Note CRUD 正确性（空内容拒绝、更新后时间戳变更）
    - 测试 ON DELETE CASCADE 级联删除

- [ ] 2. 后端：新增 Wails Bind 方法
  - [ ] 2.1 在 `app.go` 中添加 Todo 相关的 5 个 Bind 方法
    - `ListTodos`、`CreateTodo`、`ToggleTodo`、`DeleteTodo`、`ReorderTodos`
    - 每个方法调用 db 层对应函数，返回 JSON 可序列化的响应
    - 参考需求 7.1-7.5、错误处理表

  - [ ] 2.2 在 `app.go` 中添加 Note 相关的 4 个 Bind 方法
    - `ListNotes`、`CreateNote`、`UpdateNote`、`DeleteNote`
    - 参考需求 7.6-7.9、错误处理表

  - [ ] 2.3 在 `app.go` 中添加 `GetTodoCounts` Bind 方法
    - 返回所有项目的 TodoCount 切片
    - 参考需求 7.10

  - [ ] 2.4 为 Bind 方法编写集成测试
    - 测试 CreateTodo 后 ListTodos 能查询到
    - 测试 ToggleTodo 状态翻转
    - 测试 ReorderTodos 后排序正确

- [ ] 3. 检查点：Go 编译和现有测试
  - 确保 `go build ./...` 通过
  - 确保 `go test ./...` 现有测试通过
  - 如有疑问请询问用户

- [ ] 4. 前端 API 客户端：添加 Todo 和笔记接口
  - [ ] 4.1 在 `web/src/api/client.ts` 中添加 Todo、Note、TodoCount 类型定义
  - [ ] 4.2 添加 10 个 API 方法（双模：Wails Bind / HTTP）
    - `listTodos`、`createTodo`、`toggleTodo`、`deleteTodo`、`reorderTodos`
    - `listNotes`、`createNote`、`updateNote`、`deleteNote`
    - `getTodoCounts`

- [ ] 5. 前端组件：Markdown 依赖和 Todo 组件
  - [ ] 5.1 安装 `marked` 依赖
    - `cd web && npm install marked`

  - [ ] 5.2 创建 `web/src/components/TodoSection.tsx`
    - 添加输入框 + 确认按钮（标题为空时禁用）
    - Todo 列表渲染（复选框切换完成状态、删除线样式、创建时间）
    - 上下排序箭头按钮
    - 删除按钮
    - 加载骨架屏 + 空状态引导提示
    - 参考需求 2、3

  - [ ] 5.3 创建 `web/src/components/NoteSection.tsx`
    - 新建笔记按钮
    - Markdown 编辑区（textarea）和预览切换
    - 使用 `marked` 渲染 Markdown 为 HTML
    - 笔记显示创建时间和更新时间
    - 编辑/删除按钮
    - 参考需求 4

- [ ] 6. 前端组件：面板容器和页面集成
  - [ ] 6.1 创建 `web/src/components/ProjectPanel.tsx`
    - 作为右侧面板容器，嵌入 TodoSection + NoteSection
    - 接收 projectId prop

  - [ ] 6.2 更新 `web/src/pages/ProjectDetail.tsx` 为左右分栏布局
    - CSS Grid/Flex 实现 60% / 40% 分栏
    - 左侧保持原有内容，右侧嵌入 ProjectPanel
    - 响应式：小屏幕时上下堆叠

  - [ ] 6.3 更新 `web/src/components/ProjectCard.tsx`
    - 接收可选 `todoCount` prop
    - 有未完成 Todo 时显示橙色数字徽标
    - 参考需求 6.1-6.2

  - [ ] 6.4 更新 `web/src/components/SummaryBar.tsx`
    - 接收可选 `globalTodoCount` prop
    - 显示全局未完成 Todo 统计项
    - 参考需求 6.3

  - [ ] 6.5 更新 `web/src/pages/Dashboard.tsx`
    - 加载时调用 `getTodoCounts()` 获取各项目 Todo 计数
    - 将计数传递给 ProjectCard 和 SummaryBar

- [ ] 7. 全局样式：面板和编辑区域样式
  - [ ] 7.1 在 `web/src/styles/global.css` 中添加右侧面板样式
    - `.project-layout` Grid 分栏
    - `.side-panel` 面板容器
    - `.todo-list` / `.todo-item` 样式
    - `.note-section` / `.note-editor` 样式
    - Markdown 预览区域样式
    - 上下排序按钮样式
    - 响应式断点适配

- [ ] 8. 检查点：前端构建和最终验证
  - 确保 `cd web && npm run build` 通过
  - 确保 Go 编译通过
  - 打开预览验证 Todo 添加/完成/删除/排序交互
  - 验证笔记创建/编辑/预览/删除交互
  - 验证仪表盘 Todo 计数徽标显示
  - 如有疑问请询问用户
