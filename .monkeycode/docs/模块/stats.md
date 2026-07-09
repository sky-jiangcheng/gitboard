# 统计引擎 (internal/stats)

执行 `git log --shortstat` 查询每日代码变更量，并提供日期和工作日工具函数。

## 结构

```
internal/stats/
├── stats.go           # 统计查询 + 工具函数
└── stats_test.go      # 11 个单元测试
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `stats.go` | `QueryStats(path, date, author)` 执行 git log 并解析输出 |

## 依赖

**本模块依赖**: Git CLI

**依赖本模块的**: `app.go → refreshAllStats / refreshProjectStats`

## 规范

- `git log --shortstat --author=<author> --since=<date> --until=<date>` 查询统计
- 使用 `ParseShortStat` 解析 `X files changed, Y insertions(+), Z deletions(-)` 输出
- `IsWorkday` 判断是否为周一至周五
- `ValidateDate` 校验 YYYY-MM-DD 格式
