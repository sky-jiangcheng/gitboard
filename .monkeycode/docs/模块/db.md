# 数据库模块 (internal/db)

SQLite 数据库管理、建表 DDL、配置管理和所有 CRUD 查询操作。

## 结构

```
internal/db/
├── db.go              # 初始化、建表（7 张表）、默认配置
├── queries.go         # 所有 CRUD 查询 + 数据模型
└── queries_test.go    # 14 个单元测试
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `db.go` | `InitDB()` 初始化 SQLite、启用 WAL + FK、建表、插入默认配置 |
| `queries.go` | Config、ScanRoots、Projects、Repositories、DailyStats、Todos、Notes 的 CRUD |
| `queries_test.go` | Todo/Note CRUD 边界测试 + CASCADE 级联删除测试 |

## 数据库表

| 表 | 说明 |
|----|------|
| `scan_roots` | 扫描根目录 |
| `projects` | 项目分组 |
| `repositories` | Git 仓库 |
| `daily_stats` | 每日统计 |
| `app_config` | 配置键值对 |
| `project_todos` | 项目待办 |
| `project_notes` | 项目笔记 |

## 依赖

**本模块依赖**: `modernc.org/sqlite`（纯 Go SQLite 驱动）

**依赖本模块的**: `app.go` 中所有 Bind 方法

## 规范

### 函数签名
```go
func GetXxx(db *sql.DB, ...) (...) // 查询类
func UpsertXxx(db *sql.DB, ...) error // 写入类
func DeleteXxx(db *sql.DB, ...) error // 删除类
```

### 错误处理
- 查询结果为空返回空切片，不返回错误
- 不存在的记录返回 `sql.ErrNoRows`
- 数据库错误使用 `fmt.Errorf` 包装
