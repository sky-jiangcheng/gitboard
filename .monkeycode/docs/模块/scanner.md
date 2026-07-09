# 扫描器模块 (internal/scanner)

递归遍历文件系统，发现所有 Git 仓库。

## 结构

```
internal/scanner/
├── scanner.go         # 扫描逻辑
└── scanner_test.go    # 单元测试
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `scanner.go` | `ScanRepositories(roots, maxDepth)` 递归遍历目录，通过 `.git` 目录识别仓库 |

## 依赖

**本模块依赖**: 文件系统（`os`、`path/filepath`）

**依赖本模块的**: `app.go → TriggerScan()`

## 规范

- 扫描深度由 `maxDepth` 控制，避免全盘扫描
- 跳过隐藏目录（`.` 开头）
- 返回 `[]RepoInfo{Path, Depth}`
