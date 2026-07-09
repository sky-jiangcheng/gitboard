# GitBoard 接口文档

GitBoard 通过 **Wails Bind** 将 Go 后端方法自动暴露为 JavaScript 可调用函数。前端无需关心 HTTP 协议细节，直接调用 `window.go.main.App.MethodName(args)` 即可。独立前端开发时自动降级为 HTTP fetch。

## Wails Bind 方法（20 个）

### 健康检查

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `Health` | 无 | `{status, version}` | 检查数据库连接状态 |

### 项目管理

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `GetProjects` | `date string` | `[]ProjectResponse` | 获取所有项目概览（含统计） |
| `GetProjectDetail` | `id int64` | `*ProjectDetailResponse` | 获取项目详情（含仓库+统计） |
| `GetProjectStats` | `id int64, date string` | `[]DailyStat` | 获取项目每日统计 |
| `UpdateProjectLevel` | `id int64, direction string` | `*LevelUpdateResult` | 调整项目分组层级（up/down） |

### 扫描

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `TriggerScan` | 无 | `*ScanResult` | 触发全量扫描和重新分组 |

### 配置

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `GetConfig` | 无 | `*ConfigData` | 获取所有配置和扫描根目录 |
| `UpdateConfig` | `key, value string` | `error` | 更新配置项（daily_code_standard, scan_depth, git_author）|
| `UpdateScanRoots` | `scanRoots []string` | `error` | 替换扫描根目录列表 |

### 摘要

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `GetSummary` | `date string` | `*SummaryData` | 获取某日全局聚合统计 |

### Todo

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `ListTodos` | `projectId int64` | `[]Todo` | 获取项目 Todo 列表 |
| `CreateTodo` | `projectId int64, title string` | `*Todo` | 创建新 Todo（标题不能为空） |
| `ToggleTodo` | `todoId int64` | `error` | 切换完成状态 |
| `DeleteTodo` | `todoId int64` | `error` | 删除 Todo |
| `ReorderTodos` | `todoIds []int64` | `error` | 批量更新排序 |
| `GetTodoCounts` | 无 | `[]TodoCount` | 获取各项目 Todo 计数 |

### 笔记

| 方法 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `ListNotes` | `projectId int64` | `[]Note` | 获取项目笔记列表 |
| `CreateNote` | `projectId int64, content string` | `*Note` | 创建新笔记（内容不能为空） |
| `UpdateNote` | `noteId int64, content string` | `error` | 更新笔记内容 |
| `DeleteNote` | `noteId int64` | `error` | 删除笔记 |

## 数据类型

### ProjectResponse
```go
type ProjectResponse struct {
    db.Project
    RepoCount     int    `json:"repo_count"`
    TotalAdded    int    `json:"total_added"`
    TotalDeleted  int    `json:"total_deleted"`
    MyAdded       int    `json:"my_added"`
    MyDeleted     int    `json:"my_deleted"`
    MyFiles       int    `json:"my_files"`
    IsWorkday     bool   `json:"is_workday"`
    BelowStandard bool   `json:"below_standard"`
}
```

### ProjectDetailResponse
```go
type ProjectDetailResponse struct {
    *db.Project
    Repos []RepoInfo `json:"repos"`
}
```

### Todo
```go
type Todo struct {
    ID        int64  `json:"id"`
    ProjectID int64  `json:"project_id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
    Priority  int    `json:"priority"`
    SortOrder int    `json:"sort_order"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

### Note
```go
type Note struct {
    ID        int64  `json:"id"`
    ProjectID int64  `json:"project_id"`
    Content   string `json:"content"`
    SortOrder int    `json:"sort_order"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

### TodoCount
```go
type TodoCount struct {
    ProjectID int64 `json:"project_id"`
    Count     int   `json:"count"`
    Total     int   `json:"total"`
}
```

## 前端调用示例

```typescript
// 获取 Todo 列表
const todos = await listTodos(projectId)

// 双模自动检测：
// Wails 环境：window.go.main.App.ListTodos(projectId)
// 独立开发：GET /api/todos?project_id=xxx
```
