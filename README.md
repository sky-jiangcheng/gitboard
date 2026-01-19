# CodeStat - Git Commit Statistics Tool / Git 代码提交统计工具

An efficient Git commit statistics tool that supports batch statistics of multiple repositories, helping developers track work progress and code contributions.

一个高效的 Git 代码提交统计工具，支持批量统计多个仓库的提交记录，帮助开发者跟踪工作进度和代码贡献。

## Features / 功能特性

- 📊 **Batch Statistics**: Automatically scan and statistics multiple Git repositories
  **批量统计**：自动扫描并统计多个 Git 仓库的提交记录
- 👤 **Personal Contribution Tracking**: Distinguish between current user and other users' commits
  **个人贡献追踪**：区分统计当前用户与其他用户的提交
- 📅 **Flexible Date Range**: Support yesterday, today, or any specified date
  **灵活的时间范围**：支持昨天、今天或指定任意日期
- 🎯 **Workday Check**: Automatically identify workdays and weekends, set different code standards
  **工作日检查**：自动识别工作日与周末，设置不同的代码量标准
- ⚙️ **Configurable**: Support customization via configuration file for scan directories, depth, and code standards
  **可配置**：支持通过配置文件自定义扫描目录、深度和代码量标准
- 📈 **Visual Reports**: Provide clear statistical reports and completion progress
  **可视化报告**：提供清晰的统计报告和完成进度

## Installation / 安装使用

### Quick Start / 快速开始

1. Clone or download the script / 克隆或下载脚本：
```bash
git clone <repository_url>
cd CodeStat
```

2. Grant execution permissions / 赋予执行权限：
```bash
chmod +x statistics.sh
```

3. Run directly (statistics for yesterday)/ 直接运行（统计昨天的提交）：
```bash
./statistics.sh
```

### Usage Examples / 使用示例

```bash
# Statistics for yesterday (default) / 统计昨天的提交（默认）
./statistics.sh

# Statistics for today / 统计今天的提交
./statistics.sh today

# Statistics for a specified date / 统计指定日期的提交
./statistics.sh 2026-01-19
```

## Configuration / 配置说明

Create a `.statistics.conf` file in the same directory as the script for custom configuration:

在脚本同目录下创建 `.statistics.conf` 文件可自定义配置：

```bash
# Scan directory (absolute or relative path) / 扫描目录（绝对路径或相对路径）
SCAN_DIR="/path/to/projects"

# Scan depth (recursively find git repositories depth) / 扫描深度（递归查找 git 仓库的层数）
SCAN_DEPTH=3

# Workday code standard (lines/day) / 工作日代码量标准（行/天）
DAILY_CODE_STANDARD=500
```

### Configuration Options / 配置项说明

- **SCAN_DIR**: Specify the root directory to scan. The script will recursively find all Git repositories under this directory.
  **SCAN_DIR**：指定扫描的根目录，脚本会递归查找该目录下的所有 Git 仓库
- **SCAN_DEPTH**: Control the depth of recursive search to avoid scanning too deep.
  **SCAN_DEPTH**：控制递归查找的深度，避免扫描过深
- **DAILY_CODE_STANDARD**: Expected daily code standard for workdays, not enforced on weekends.
  **DAILY_CODE_STANDARD**：工作日每天期望的代码量标准，周末不强制要求

## Sample Output / 统计输出示例

```
Statistics Date: 2026-01-19 (Yesterday) / 统计日期: 2026-01-19 (昨天)
Statistics User: John Doe / 统计用户: John Doe
Scan Directory: /path/to/projects / 扫描目录: /path/to/projects
Scan Depth: 3 levels / 扫描深度: 3 层
Date Type: Workday (Code standard: 500 lines/day) / 日期类型: 工作日（代码量标准: 500 行/天）

=== Statistics: project-1 === / === 正在统计: project-1 ===
  File Changes: 12 (Mine: 8) / 文件变更: 12 (我的: 8)
  Lines Added: 256 (Mine: 180) / 新增行数: 256 (我的: 180)
  Lines Deleted: 45 (Mine: 30) / 删除行数: 45 (我的: 30)
  Net Lines: 211 (Mine: 150) / 净增行数: 211 (我的: 150)

==========================================
Statistics Summary (2026-01-19) / 统计汇总 (2026-01-19)
==========================================
Repository Count: 3 / 仓库数量: 3
------------------------------------------
All Users Summary / 所有人汇总:
  Total File Changes: 25 / 文件变更总数: 25
  Total Lines Added: 520 / 新增行数总计: 520
  Total Lines Deleted: 80 / 删除行数总计: 80
  Total Net Lines: 440 / 净增行数总计: 440
------------------------------------------
John Doe's Contribution / John Doe 的贡献:
  File Changes: 15 / 文件变更数: 15
  Lines Added: 320 / 新增行数: 320
  Lines Deleted: 50 / 删除行数: 50
  Net Lines: 270 / 净增行数: 270
  Percentage: 61.5% / 占比: 61.5%
==========================================

==========================================
Workday Code Check (Yesterday) / 工作日代码量检查 (昨天)
==========================================
Standard Requirement: 500 lines per day / 标准要求: 每天 500 行代码
Actual Commits: 320 lines / 实际提交: 320 行
Completion Progress: 64.0% / 完成进度: 64.0%

⚠️  Warning: Yesterday did not meet the workday code standard! / 警告: 昨天未达到工作日代码量标准！
⚠️  Need to add: 180 lines of code / 还需要补充: 180 行代码
⚠️  Please work hard to catch up! / 请努力赶上进度！
```

## How It Works / 工作原理

1. Scan all `.git` folders under the specified directory (recursion depth configurable)
   扫描指定目录下所有 `.git` 文件夹（递归深度可配置）
2. Use `git log` command to statistics commit records within the specified date range
   使用 `git log` 命令统计指定日期范围内的提交记录
3. Parse commit statistics (lines added, lines deleted, file changes)
   解析提交统计信息（新增行数、删除行数、文件变更数）
4. Separately statistics contributions from all users and current git user (via `git config user.name`)
   分别统计所有用户和当前 git 用户（通过 `git config user.name`）的贡献
5. Generate summary report and check progress according to workday standards
   汇总生成报告，并根据工作日标准进行进度检查

## System Requirements / 系统要求

- **Operating System / 操作系统**: Linux or macOS / Linux 或 macOS
- **Dependencies / 依赖工具**:
  - Bash shell
  - Git
  - Standard Unix tools like awk, date, etc. / awk、date 等标准 Unix 工具

## Notes / 注意事项

- The script identifies personal commits based on current git configuration (`git config user.name`)
  脚本会根据当前 git 配置（`git config user.name`）识别个人提交
- Configuration file priority: Configuration file > Default values
  配置文件优先级：配置文件 > 默认值
- No code quantity warnings on weekends (Saturday, Sunday)
  周末（周六、周日）不进行代码量告警
- Date format must be `YYYY-MM-DD`
  日期格式必须为 `YYYY-MM-DD`

## License / 许可证

MIT License

## Contributing / 贡献

Issues and Pull Requests are welcome! / 欢迎提交 Issue 和 Pull Request！
