#!/bin/bash

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 配置文件路径（优先使用脚本同目录下的配置文件）
CONFIG_FILE="$SCRIPT_DIR/.statistics.conf"

# 加载配置文件（如果存在）
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi

# 默认配置（可通过配置文件覆盖）
SCAN_DIR="${SCAN_DIR:-.}"                    # 扫描目录，默认当前目录
SCAN_DEPTH="${SCAN_DEPTH:-3}"                 # 扫描深度，默认3层
DAILY_CODE_STANDARD="${DAILY_CODE_STANDARD:-500}"  # 代码量标准，默认500行/天

# 统计模式：yesterday、today 或指定日期（格式：YYYY-MM-DD）
STATS_MODE="${1:-yesterday}"

if [ "$STATS_MODE" = "yesterday" ]; then
    # 获取昨天的日期（格式：YYYY-MM-DD）
    STATS_DATE=$(date -v-1d +"%Y-%m-%d" 2>/dev/null || date -d "yesterday" +"%Y-%m-%d" 2>/dev/null)
    DATE_DESC="昨天"
elif [ "$STATS_MODE" = "today" ]; then
    # 获取今天的日期（格式：YYYY-MM-DD）
    STATS_DATE=$(date +"%Y-%m-%d")
    DATE_DESC="今天"
else
    # 尝试作为指定日期处理
    # 验证日期格式是否正确（YYYY-MM-DD）
    if date -j -f "%Y-%m-%d" "$STATS_MODE" +"%Y-%m-%d" >/dev/null 2>&1 || \
       date -d "$STATS_MODE" +"%Y-%m-%d" >/dev/null 2>&1; then
        STATS_DATE="$STATS_MODE"
        DATE_DESC="指定日期 ($STATS_MODE)"
    else
        echo "错误: 无效的统计模式，请使用 'yesterday'、'today' 或指定日期格式 YYYY-MM-DD"
        echo "Error: Invalid statistics mode. Please use 'yesterday', 'today', or date format YYYY-MM-DD"
        echo "用法: $0 [yesterday|today|YYYY-MM-DD]"
        echo "Usage: $0 [yesterday|today|YYYY-MM-DD]"
        echo "示例: $0 2026-01-19"
        echo "Example: $0 2026-01-19"
        exit 1
    fi
fi

# 获取当前 git 用户名
MY_NAME=$(git config user.name 2>/dev/null || echo "")

# 判断是否为工作日（周一到周五）
if [ "$STATS_MODE" = "yesterday" ]; then
    DAY_OF_WEEK=$(date -v-1d +"%u" 2>/dev/null || date -d "yesterday" +"%u" 2>/dev/null)
elif [ "$STATS_MODE" = "today" ]; then
    DAY_OF_WEEK=$(date +"%u" 2>/dev/null || date +"%u" 2>/dev/null)
else
    # 对于指定日期，计算该日期是周几
    DAY_OF_WEEK=$(date -j -f "%Y-%m-%d" "$STATS_DATE" +"%u" 2>/dev/null || date -d "$STATS_DATE" +"%u" 2>/dev/null)
fi

IS_WORKDAY=1
if [ "$DAY_OF_WEEK" -ge 6 ]; then
    IS_WORKDAY=0
fi

echo "统计日期: $STATS_DATE ($DATE_DESC) | Statistics Date: $STATS_DATE ($DATE_DESC)"
echo "统计用户: $MY_NAME | User: $MY_NAME"
echo "扫描目录: $SCAN_DIR | Scan Directory: $SCAN_DIR"
echo "扫描深度: $SCAN_DEPTH 层 | Scan Depth: $SCAN_DEPTH"
if [ $IS_WORKDAY -eq 1 ]; then
    echo "日期类型: 工作日（代码量标准: $DAILY_CODE_STANDARD 行/天） | Type: Workday (Standard: $DAILY_CODE_STANDARD lines/day)"
else
    echo "日期类型: 周末（无代码量要求） | Type: Weekend (No requirement)"
fi
echo ""

total_additions=0
total_deletions=0
total_files=0
repo_count=0

my_additions=0
my_deletions=0
my_files=0

# 创建临时文件存储统计结果
tmp_file=$(mktemp)
my_tmp_file=$(mktemp)

# 检查扫描目录是否存在
if [ ! -d "$SCAN_DIR" ]; then
    echo "错误: 扫描目录不存在: $SCAN_DIR | Error: Scan directory does not exist: $SCAN_DIR"
    echo "请检查配置文件或使用绝对路径 | Please check config file or use absolute path"
    exit 1
fi

# 递归查找所有 git 仓库
find "$SCAN_DIR" -maxdepth "$SCAN_DEPTH" -name ".git" -type d 2>/dev/null | while read git_dir; do
    # 获取仓库的父目录
    repo_dir=$(dirname "$git_dir")
    repo_name=$(basename "$repo_dir")
    repo_path=$(cd "$repo_dir" && pwd)

    # 统计所有用户的提交信息
    stats=$(cd "$repo_path" && git log \
        --since="$STATS_DATE 00:00:00" \
        --until="$STATS_DATE 23:59:59" \
        --pretty=tformat: \
        --shortstat \
        2>/dev/null | awk '
        {
            if ($0 ~ /files? changed/) {
                files += $1;
                added += $4;
                # 检查是否有删除行数
                for (i=5; i<=NF; i++) {
                    if ($(i-1) ~ /deletion/) {
                        deleted += $i;
                    }
                }
            }
        } END {
            print files " " added " " deleted
        }')

    # 统计当前用户的提交信息
    my_stats=$(cd "$repo_path" && git log \
        --since="$STATS_DATE 00:00:00" \
        --until="$STATS_DATE 23:59:59" \
        --author="$MY_NAME" \
        --pretty=tformat: \
        --shortstat \
        2>/dev/null | awk '
        {
            if ($0 ~ /files? changed/) {
                files += $1;
                added += $4;
                # 检查是否有删除行数
                for (i=5; i<=NF; i++) {
                    if ($(i-1) ~ /deletion/) {
                        deleted += $i;
                    }
                }
            }
        } END {
            print files " " added " " deleted
        }')

    # 解析统计结果
    if [ -n "$stats" ]; then
        files=$(echo "$stats" | awk '{print $1}')
        add=$(echo "$stats" | awk '{print $2}')
        del=$(echo "$stats" | awk '{print $3}')
        # 确保变量有默认值
        files=${files:-0}
        add=${add:-0}
        del=${del:-0}
    else
        files=0
        add=0
        del=0
    fi

    # 解析当前用户统计结果
    if [ -n "$my_stats" ]; then
        my_files_local=$(echo "$my_stats" | awk '{print $1}')
        my_add_local=$(echo "$my_stats" | awk '{print $2}')
        my_del_local=$(echo "$my_stats" | awk '{print $3}')
        # 确保变量有默认值
        my_files_local=${my_files_local:-0}
        my_add_local=${my_add_local:-0}
        my_del_local=${my_del_local:-0}
    else
        my_files_local=0
        my_add_local=0
        my_del_local=0
    fi

    # 只显示有变更的仓库（所有人或自己有提交）
    if [ "${add:-0}" -gt 0 ] || [ "${del:-0}" -gt 0 ] || [ "${my_add_local:-0}" -gt 0 ] || [ "${my_del_local:-0}" -gt 0 ]; then
        echo "=== 正在统计: $repo_name | Statistics: $repo_name ==="
        echo "  文件变更: $files (我的: $my_files_local) | Files Changed: $files (Mine: $my_files_local)"
        echo "  新增行数: $add (我的: $my_add_local) | Lines Added: $add (Mine: $my_add_local)"
        echo "  删除行数: $del (我的: $my_del_local) | Lines Deleted: $del (Mine: $my_del_local)"
        echo "  净增行数: $((add - del)) (我的: $((my_add_local - my_del_local))) | Net Added: $((add - del)) (Mine: $((my_add_local - my_del_local)))"
        echo ""
    fi

    # 写入临时文件（所有仓库都写入，用于汇总统计）
    echo "$files $add $del" >> "$tmp_file"
    echo "$my_files_local $my_add_local $my_del_local" >> "$my_tmp_file"
done

# 从临时文件累加统计结果
while read files add del; do
    repo_count=$((repo_count + 1))
    total_files=$((total_files + files))
    total_additions=$((total_additions + add))
    total_deletions=$((total_deletions + del))
done < "$tmp_file"

# 累加当前用户的统计结果
while read files add del; do
    my_files=$((my_files + files))
    my_additions=$((my_additions + add))
    my_deletions=$((my_deletions + del))
done < "$my_tmp_file"

# 删除临时文件
rm -f "$tmp_file" "$my_tmp_file"

# 计算占比
if [ $total_additions -gt 0 ]; then
    percentage=$(awk "BEGIN {printf \"%.1f\", $my_additions * 100 / $total_additions}")
else
    percentage="N/A"
fi

echo "=========================================="
echo "统计汇总 ($STATS_DATE) | Summary ($STATS_DATE)"
echo "=========================================="
echo "仓库数量: $repo_count | Repositories: $repo_count"
echo "------------------------------------------"
echo "所有人汇总: | All Users:"
echo "  文件变更总数: $total_files | Total Files Changed: $total_files"
echo "  新增行数总计: $total_additions | Total Lines Added: $total_additions"
echo "  删除行数总计: $total_deletions | Total Lines Deleted: $total_deletions"
echo "  净增行数总计: $((total_additions - total_deletions)) | Net Lines Added: $((total_additions - total_deletions))"
echo "------------------------------------------"
echo "$MY_NAME 的贡献: | $MY_NAME's Contribution:"
echo "  文件变更数: $my_files | Files Changed: $my_files"
echo "  新增行数: $my_additions | Lines Added: $my_additions"
echo "  删除行数: $my_deletions | Lines Deleted: $my_deletions"
echo "  净增行数: $((my_additions - my_deletions)) | Net Lines Added: $((my_additions - my_deletions))"
if [ "$percentage" != "N/A" ]; then
    echo "  占比: ${percentage}% | Percentage: ${percentage}%"
else
    echo "  占比: N/A | Percentage: N/A"
fi
echo "=========================================="

# 工作日代码量告警
if [ $IS_WORKDAY -eq 1 ]; then
    echo ""
    echo "=========================================="
    echo "工作日代码量检查 ($DATE_DESC) | Workday Code Check ($DATE_DESC)"
    echo "=========================================="
    echo "标准要求: 每天 $DAILY_CODE_STANDARD 行代码 | Standard: $DAILY_CODE_STANDARD lines/day"
    echo "实际提交: $my_additions 行 | Actual: $my_additions lines"
    progress=$(awk "BEGIN {printf \"%.1f\", $my_additions * 100.0 / $DAILY_CODE_STANDARD}")
    echo "完成进度: ${progress}% | Progress: ${progress}%"
    echo ""

    if [ $my_additions -lt $DAILY_CODE_STANDARD ]; then
        GAP=$((DAILY_CODE_STANDARD - my_additions))
        echo "⚠️  警告: $DATE_DESC未达到工作日代码量标准！"
        echo "⚠️  Warning: Workday code standard not met on $DATE_DESC!"
        echo "⚠️  还需要补充: $GAP 行代码 | Need: $GAP more lines"
        echo "⚠️  请努力赶上进度！ | Please catch up!"
        exit 1
    else
        echo "✅ $DATE_DESC已完成工作日代码量标准！继续保持！"
        echo "✅ Workday code standard met! Keep it up!"
        exit 0
    fi
else
    echo ""
    echo "=========================================="
    echo "周末代码量检查 | Weekend Code Check"
    echo "=========================================="
    echo "周末无强制代码量要求 | No mandatory requirement on weekends"
    echo "实际提交: $my_additions 行 | Actual: $my_additions lines"
    if [ $my_additions -gt 0 ]; then
        echo "👍 周末也这么努力，太棒了！"
        echo "👍 Working hard on weekend too! Awesome!"
    fi
    exit 0
fi
