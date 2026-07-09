# 需求文档

## 概述

为 GitBoard 的每个项目添加综合 Todo 与笔记功能，用户可在项目详情页管理待办事项和记录笔记，提升日常开发工作的可玩性和实用性。

## 术语表

- **项目（Project）**：GitBoard 中的项目分组节点，对应 `projects` 表中的一行记录
- **Todo**：与项目关联的待办事项，支持完成状态标记和优先级
- **笔记（Note）**：与项目关联的自由文本记录，支持 Markdown 格式
- **系统（System）**：GitBoard 应用的 Go 后端 + React 前端

## 需求

### 需求 1：数据库存储

**用户故事：** AS 开发者，I want Todo 和笔记数据能持久化存储，so that 关闭应用后数据不会丢失。

#### 验收标准

1. 系统 SHALL 在 SQLite 数据库中创建 `project_todos` 表，包含 id、project_id、title、completed、priority、sort_order、created_at、updated_at 字段
2. 系统 SHALL 在 SQLite 数据库中创建 `project_notes` 表，包含 id、project_id、content、sort_order、created_at、updated_at 字段
3. `project_id` SHALL 引用 `projects` 表的 `id` 字段，并设置外键约束
4. 数据库初始化时 SHALL 自动创建上述两张新表

### 需求 2：Todo 展示

**用户故事：** AS 开发者，I want 在项目详情页看到该项目的 Todo 列表，so that 快速了解当前项目的待办事项。

#### UI 决策

**项目详情页布局**：左右分栏。左侧显示原有内容（趋势图、仓库列表、层级调整），右侧面板显示 Todo 和笔记。

**验收标准**

1. 系统 SHALL 在项目详情页右侧面板展示 Todo 列表
2. WHEN Todo 列表为空，系统 SHALL 显示引导提示"暂无待办，点击添加"
3. 每个 Todo 项 SHALL 显示标题、完成状态（复选框）、创建时间
4. 系统 SHALL 按 sort_order 升序排列 Todo 项
5. 已完成的 Todo SHALL 以删除线样式展示

### 需求 3：Todo 增删改

**用户故事：** AS 开发者，I want 能添加、完成、删除 Todo，so that 管理项目中的待办事项。

#### 验收标准

1. 系统 SHALL 在 Todo 列表顶部提供添加输入框和确认按钮
2. WHEN 用户输入标题并点击添加，系统 SHALL 创建新的 Todo 并刷新列表
3. 系统 SHALL 在 Todo 标题为空时禁用添加按钮
4. WHEN 用户勾选复选框，系统 SHALL 将 Todo 标记为已完成
5. WHEN 用户点击删除按钮，系统 SHALL 移除该 Todo
6. 系统 SHALL 为每个 Todo 提供拖拽排序能力

### 需求 4：笔记展示与编辑

**用户故事：** AS 开发者，I want 在项目详情页撰写和查看笔记，so that 记录项目相关的思路、决策和备忘。

#### UI 决策

**笔记编辑器**：Markdown 编辑器，提供编辑区 + 实时预览切换。

**验收标准**

1. 系统 SHALL 在项目详情页右侧面板展示笔记列表
2. WHEN 用户创建或编辑笔记，系统 SHALL 提供 Markdown 文本编辑区域
3. 系统 SHALL 以渲染后的 Markdown 格式展示已有笔记
4. 笔记 SHALL 显示创建时间和最后更新时间
5. WHEN 用户点击编辑按钮，系统 SHALL 切换为 Markdown 编辑模式
6. WHEN 用户点击删除按钮，系统 SHALL 在确认后移除该笔记

### 需求 5：与项目关联

**用户故事：** AS 开发者，I want Todo 和笔记数据与具体项目绑定，so that 不同项目的数据互不干扰。

#### 验收标准

1. 系统 SHALL 通过 project_id 将 Todo 和笔记关联到具体项目
2. WHEN 用户在项目 A 详情页添加 Todo 或笔记，系统 SHALL 仅在该项目下可见
3. IF 项目被删除，系统 SHALL 级联删除关联的 Todo 和笔记

### 需求 6：仪表盘集成

**用户故事：** AS 开发者，I want 在仪表盘上能看到各项目的待办统计，so that 从概览页面即可感知哪些项目有待处理事项。

#### 验收标准

1. 系统 SHALL 在项目卡片上显示未完成 Todo 数量徽标
2. WHEN 项目有未完成 Todo，系统 SHALL 在卡片上显示橙色数字徽标
3. 系统 SHALL 在 SummaryBar 中显示全局未完成 Todo 总数

### 需求 7：后端 API（Wails Bind）

**用户故事：** AS 前端开发者，I want 通过标准化的 API 调用操作 Todo 和笔记，so that 前端无需关心底层数据库实现。

#### 验收标准

1. 系统 SHALL 提供 `ListTodos(projectId int64) []Todo` 方法获取项目 Todo 列表
2. 系统 SHALL 提供 `CreateTodo(projectId int64, title string) (*Todo, error)` 方法创建 Todo
3. 系统 SHALL 提供 `ToggleTodo(todoId int64) error` 方法切换完成状态
4. 系统 SHALL 提供 `DeleteTodo(todoId int64) error` 方法删除 Todo
5. 系统 SHALL 提供 `ReorderTodos(todoIds []int64) error` 方法更新排序
6. 系统 SHALL 提供 `ListNotes(projectId int64) []Note` 方法获取项目笔记列表
7. 系统 SHALL 提供 `CreateNote(projectId int64, content string) (*Note, error)` 方法创建笔记
8. 系统 SHALL 提供 `UpdateNote(noteId int64, content string) error` 方法更新笔记
9. 系统 SHALL 提供 `DeleteNote(noteId int64) error` 方法删除笔记
10. 系统 SHALL 提供 `GetTodoCounts() []TodoCount` 方法获取各项目 Todo 计数
