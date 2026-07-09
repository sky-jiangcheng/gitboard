# 分组器模块 (internal/grouper)

将扫描发现的 Git 仓库按目录层级自动分组为项目。

## 结构

```
internal/grouper/
├── grouper.go         # 分组逻辑
└── grouper_test.go    # 7 个单元测试
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `grouper.go` | `GroupRepositories(repos)` 按路径层级分组，返回 `[]ProjectGroup` |

## 依赖

**本模块依赖**: `internal/scanner`（RepoInfo 类型）

**依赖本模块的**: `app.go → TriggerScan()`

## 规范

- 同一父目录下多个仓库合并为一个项目
- 项目名称基于共同路径前缀生成
- 支持 `level_override` 手动调整层级
