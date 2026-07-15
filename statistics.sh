#!/bin/bash

# CodeStat - Git Code Statistics Tool
# Features:
# - Single day or date range statistics
# - Multiple branch support
# - Configurable with config file or command line
# - Bilingual output (Chinese + English)
# - Workday progress tracking

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Config file path (prefer same directory as script)
CONFIG_FILE="$SCRIPT_DIR/.statistics.conf"

# Default configuration (can be overridden by config file or command line)
SCAN_DIR="."
SCAN_DEPTH=3
DAILY_CODE_STANDARD=500
STATS_BRANCHES=""
EXCLUDE_DIRS=""
OUTPUT_FORMAT="text"
OUTPUT_FILE=""
QUIET=0
SHOW_HELP=0
SHOW_VERSION=0

# Statistics mode: single | range
MODE_TYPE="single"
STATS_MODE="yesterday"
STATS_DATE=""
STATS_SINCE=""
STATS_UNTIL=""
AUTHOR=""

# Safe load config file (if exists)
# Use safe config parsing to avoid security risks of source command
if [ -f "$CONFIG_FILE" ]; then
    # Safely read config items, only allow specific config formats
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip empty lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue

        # Only process allowed configuration variables
        if [[ "$line" =~ ^SCAN_DIR[[:space:]]*=[[:space:]]*(.+)$ ]]; then
            val="${BASH_REMATCH[1]}"
            val="${val#\"}" && val="${val%\"}" && val="${val#\'}" && val="${val%\'}"
            SCAN_DIR="$val"
        elif [[ "$line" =~ ^SCAN_DEPTH[[:space:]]*=[[:space:]]*([0-9]+)$ ]]; then
            SCAN_DEPTH="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ^DAILY_CODE_STANDARD[[:space:]]*=[[:space:]]*([0-9]+)$ ]]; then
            DAILY_CODE_STANDARD="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ^STATS_BRANCHES[[:space:]]*=[[:space:]]*(.+)$ ]]; then
            val="${BASH_REMATCH[1]}"
            val="${val#\"}" && val="${val%\"}" && val="${val#\'}" && val="${val%\'}"
            STATS_BRANCHES="$val"
        elif [[ "$line" =~ ^EXCLUDE_DIRS[[:space:]]*=[[:space:]]*(.+)$ ]]; then
            val="${BASH_REMATCH[1]}"
            val="${val#\"}" && val="${val%\"}" && val="${val#\'}" && val="${val%\'}"
            EXCLUDE_DIRS="$val"
        elif [[ "$line" =~ ^AUTHOR[[:space:]]*=[[:space:]]*(.+)$ ]]; then
            val="${BASH_REMATCH[1]}"
            val="${val#\"}" && val="${val%\"}" && val="${val#\'}" && val="${val%\'}"
            AUTHOR="$val"
        fi
    done < "$CONFIG_FILE"
fi

# Get current git username (default author)
if [ -z "$AUTHOR" ]; then
    AUTHOR=$(git config user.name 2>/dev/null || echo "")
fi

# Show help
show_help() {
    echo "CodeStat - Git Code Statistics Tool"
    echo ""
    echo "Usage: $0 [OPTIONS] [MODE]"
    echo ""
    echo "Mode (for single day):"
    echo "  yesterday        Statistics for yesterday (default)"
    echo "  today            Statistics for today"
    echo "  YYYY-MM-DD       Statistics for specified date"
    echo ""
    echo "Options:"
    echo "  -h, --help       Show this help message"
    echo "  -v, --version    Show version"
    echo "  --dir PATH       Override scan directory (default: from config or .)"
    echo "  --depth N        Override scan depth (default: from config or 3)"
    echo "  --branches LIST  Override branches to scan (comma-separated)"
    echo "  --standard N     Override daily code standard (lines/day)"
    echo "  --since DATE     Start date for range mode (YYYY-MM-DD)"
    echo "  --until DATE     End date for range mode (YYYY-MM-DD)"
    echo "  --exclude LIST   Exclude directories (space-separated, e.g. 'node_modules vendor')"
    echo "  --author NAME    Author name to track (defaults to git user.name)"
    echo "  --format FMT     Output format: text|json|csv (default: text)"
    echo "  --output FILE    Write output to file"
    echo "  --quiet          Quiet mode, only output summary"
    echo ""
    echo "Examples:"
    echo "  $0 yesterday                         # Statistics yesterday (use config)"
    echo "  $0 2026-01-19                         # Statistics specific day"
    echo "  $0 --since 2026-01-01 --until 2026-01-31  # January statistics"
    echo "  $0 --dir ~/projects --depth 4 --format json  # Custom directory and JSON output"
    echo ""
}

# Show version
show_version() {
    echo "CodeStat v2.0"
    echo "Bilingual Git Code Statistics Tool"
}

# Validate branch name, prevent command injection
# Only allow letters, numbers, underscore, hyphen, slash, and wildcards *?
validate_branch_name() {
    local branch="$1"
    # Check if branch name contains dangerous characters
    if [[ ! "$branch" =~ ^[a-zA-Z0-9_/\*?-]+$ ]]; then
        echo "警告: 分支名包含非法字符，已跳过: $branch | Warning: Invalid branch name skipped: $branch" >&2
        return 1
    fi
    # Check for path traversal
    if [[ "$branch" =~ \.\. ]] || [[ "$branch" =~ ^/ ]] || [[ "$branch" =~ /$ ]]; then
        echo "警告: 分支名包含非法路径，已跳过: $branch | Warning: Invalid path in branch name skipped: $branch" >&2
        return 1
    fi
    return 0
}

# Validate date format YYYY-MM-DD
validate_date() {
    local date="$1"
    if [[ ! "$date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        return 1
    fi
    if date -j -f "%Y-%m-%d" "$date" +"%Y-%m-%d" >/dev/null 2>&1 || \
       date -d "$date" +"%Y-%m-%d" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Check if directory should be excluded
should_exclude() {
    local dir="$1"
    local dirname=$(basename "$dir")
    for exclude in $EXCLUDE_DIRS; do
        if [ "$dirname" = "$exclude" ]; then
            return 0
        fi
    done
    return 1
}

# Cleanup temporary files
cleanup() {
    if [ -n "$tmp_file" ] && [ -f "$tmp_file" ]; then
        rm -f "$tmp_file"
    fi
    if [ -n "$my_tmp_file" ] && [ -f "$my_tmp_file" ]; then
        rm -f "$my_tmp_file"
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Parse command line arguments
while [ $# -gt 0 ]; do
    case "$1" in
        -h|--help)
            SHOW_HELP=1
            shift
            ;;
        -v|--version)
            SHOW_VERSION=1
            shift
            ;;
        --dir)
            SCAN_DIR="$2"
            shift 2
            ;;
        --depth)
            SCAN_DEPTH="$2"
            if ! [[ "$SCAN_DEPTH" =~ ^[0-9]+$ ]]; then
                echo "错误: 深度必须是数字 | Error: depth must be a number" >&2
                exit 1
            fi
            shift 2
            ;;
        --branches)
            STATS_BRANCHES="$2"
            shift 2
            ;;
        --standard)
            DAILY_CODE_STANDARD="$2"
            if ! [[ "$DAILY_CODE_STANDARD" =~ ^[0-9]+$ ]]; then
                echo "错误: 标准必须是数字 | Error: standard must be a number" >&2
                exit 1
            fi
            shift 2
            ;;
        --since)
            STATS_SINCE="$2"
            if ! validate_date "$STATS_SINCE"; then
                echo "错误: 无效日期格式，需要 YYYY-MM-DD | Error: Invalid date format, need YYYY-MM-DD" >&2
                exit 1
            fi
            MODE_TYPE="range"
            shift 2
            ;;
        --until)
            STATS_UNTIL="$2"
            if ! validate_date "$STATS_UNTIL"; then
                echo "错误: 无效日期格式，需要 YYYY-MM-DD | Error: Invalid date format, need YYYY-MM-DD" >&2
                exit 1
            fi
            MODE_TYPE="range"
            shift 2
            ;;
        --exclude)
            EXCLUDE_DIRS="$2"
            shift 2
            ;;
        --author)
            AUTHOR="$2"
            shift 2
            ;;
        --format)
            OUTPUT_FORMAT="$2"
            if [ "$OUTPUT_FORMAT" != "text" ] && [ "$OUTPUT_FORMAT" != "json" ] && [ "$OUTPUT_FORMAT" != "csv" ]; then
                echo "错误: 格式必须是 text|json|csv | Error: format must be text|json|csv" >&2
                exit 1
            fi
            shift 2
            ;;
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --quiet)
            QUIET=1
            shift
            ;;
        *)
            # Positional argument - mode or single date
            STATS_MODE="$1"
            shift
            ;;
    esac
done

# Show help if requested
if [ $SHOW_HELP -eq 1 ]; then
    show_help
    exit 0
fi

# Show version if requested
if [ $SHOW_VERSION -eq 1 ]; then
    show_version
    exit 0
fi

# Process date range
if [ "$MODE_TYPE" = "range" ]; then
    # If since is set but until not, use today as until
    if [ -n "$STATS_SINCE" ] && [ -z "$STATS_UNTIL" ]; then
        STATS_UNTIL=$(date +"%Y-%m-%d")
    fi
    # If until is set but since not, error
    if [ -z "$STATS_SINCE" ] || [ -z "$STATS_UNTIL" ]; then
        echo "错误: 范围模式需要同时指定 --since 和 --until | Error: Range mode requires both --since and --until" >&2
        exit 1
    fi
    DATE_RANGE_DESC="$STATS_SINCE 至 $STATS_UNTIL | $STATS_SINCE to $STATS_UNTIL"
else
    # Process single day mode
    if [ "$STATS_MODE" = "yesterday" ]; then
        # Get yesterday date (format: YYYY-MM-DD)
        STATS_DATE=$(date -v-1d +"%Y-%m-%d" 2>/dev/null || date -d "yesterday" +"%Y-%m-%d" 2>/dev/null)
        DATE_DESC="昨天 | Yesterday"
    elif [ "$STATS_MODE" = "today" ]; then
        # Get today date
        STATS_DATE=$(date +"%Y-%m-%d")
        DATE_DESC="今天 | Today"
    else
        # Try to parse as specific date
        if validate_date "$STATS_MODE"; then
            STATS_DATE="$STATS_MODE"
            DATE_DESC="指定日期 ($STATS_DATE) | Specified date ($STATS_DATE)"
        else
            echo "错误: 无效的统计模式，请使用 'yesterday'、'today' 或指定日期格式 YYYY-MM-DD" >&2
            echo "Error: Invalid statistics mode. Please use 'yesterday', 'today', or date format YYYY-MM-DD" >&2
            echo "用法: $0 [yesterday|today|YYYY-MM-DD]" >&2
            echo "Usage: $0 [yesterday|today|YYYY-MM-DD]" >&2
            echo "示例: $0 2026-01-19" >&2
            exit 1
        fi
    fi
fi

# Determine if it's workday (Monday to Friday)
if [ "$MODE_TYPE" = "single" ]; then
    if [ "$STATS_MODE" = "yesterday" ]; then
        DAY_OF_WEEK=$(date -v-1d +"%u" 2>/dev/null || date -d "yesterday" +"%u" 2>/dev/null)
    elif [ "$STATS_MODE" = "today" ]; then
        DAY_OF_WEEK=$(date +"%u" 2>/dev/null || date +"%u" 2>/dev/null)
    else
        DAY_OF_WEEK=$(date -j -f "%Y-%m-%d" "$STATS_DATE" +"%u" 2>/dev/null || date -d "$STATS_DATE" +"%u" 2>/dev/null)
    fi
    IS_WORKDAY=1
    if [ "$DAY_OF_WEEK" -ge 6 ]; then
        IS_WORKDAY=0
    fi
else
    # For range mode, don't do workday check
    IS_WORKDAY=-1
fi

# Check if git is available
if ! command -v git >/dev/null 2>&1; then
    echo "错误: git 命令未找到，请先安装 git | Error: git command not found, please install git first" >&2
    exit 1
fi

# Check scan directory exists
if [ ! -d "$SCAN_DIR" ]; then
    echo "错误: 扫描目录不存在: $SCAN_DIR | Error: Scan directory does not exist: $SCAN_DIR" >&2
    echo "请检查配置文件或使用绝对路径 | Please check config file or use absolute path" >&2
    exit 1
fi

# Initialize statistics
total_additions=0
total_deletions=0
total_files=0
total_commits=0
repo_count=0

my_additions=0
my_deletions=0
my_files=0
my_commits=0

# Create temporary files for results
tmp_file=$(mktemp)
my_tmp_file=$(mktemp)

# Start header output only in text mode
if [ "$OUTPUT_FORMAT" = "text" ] && [ $QUIET -eq 0 ]; then
    echo "=========================================="
    echo "CodeStat 代码统计工具 / Code Statistics Tool"
    echo "=========================================="
    if [ "$MODE_TYPE" = "single" ]; then
        echo "统计日期: $STATS_DATE ($DATE_DESC) | Statistics Date: $STATS_DATE ($DATE_DESC)"
    else
        echo "统计范围: $DATE_RANGE_DESC | Date Range: $DATE_RANGE_DESC"
    fi
    echo "统计用户: $AUTHOR | User: $AUTHOR"
    echo "扫描目录: $SCAN_DIR | Scan Directory: $SCAN_DIR"
    echo "扫描深度: $SCAN_DEPTH 层 | Scan Depth: $SCAN_DEPTH"
    if [ -n "$STATS_BRANCHES" ]; then
        echo "统计分支: $STATS_BRANCHES | Branches: $STATS_BRANCHES"
    else
        echo "统计分支: 当前分支 | Branches: Current branch"
    fi
    if [ -n "$EXCLUDE_DIRS" ]; then
        echo "排除目录: $EXCLUDE_DIRS | Excluded: $EXCLUDE_DIRS"
    fi
    if [ "$MODE_TYPE" = "single" ]; then
        if [ $IS_WORKDAY -eq 1 ]; then
            echo "日期类型: 工作日（代码量标准: $DAILY_CODE_STANDARD 行/天） | Type: Workday (Standard: $DAILY_CODE_STANDARD lines/day)"
        else
            echo "日期类型: 周末（无代码量要求） | Type: Weekend (No requirement)"
        fi
    fi
    echo ""
fi

# Process each git repository
# Build find exclusion arguments
find_exclude_args=()
if [ -n "$EXCLUDE_DIRS" ]; then
    for exc in $EXCLUDE_DIRS; do
        find_exclude_args+=(-not -path "*/$exc/*")
    done
fi

# Process each git repository
find "$SCAN_DIR" -maxdepth "$SCAN_DEPTH" -name ".git" -type d "${find_exclude_args[@]}" 2>/dev/null | while read -r git_dir; do
    # Get repository parent directory
    repo_dir=$(dirname "$git_dir")

    # Check if should exclude this directory
    if should_exclude "$repo_dir"; then
        continue
    fi

    repo_name=$(basename "$repo_dir")
    repo_path=$(cd "$repo_dir" && pwd)

    # Build branch parameters
    branch_params=""
    if [ -n "$STATS_BRANCHES" ]; then
        # Support comma or space separated branch list
        branch_list=$(echo "$STATS_BRANCHES" | tr ',' ' ')
        valid_branches=""

        for branch in $branch_list; do
            # Safety validation for branch name
            if ! validate_branch_name "$branch"; then
                continue
            fi

            # Check for wildcards
            if [[ "$branch" == *"*"* ]] || [[ "$branch" == *"?"* ]]; then
                # Use wildcard to match branches
                matched_branches=$(cd "$repo_path" && git branch --list "$branch" 2>/dev/null | sed 's/^[* ]*//')
                for matched_branch in $matched_branches; do
                    if validate_branch_name "$matched_branch"; then
                        if [ -z "$valid_branches" ]; then
                            valid_branches="$matched_branch"
                        else
                            valid_branches="$valid_branches $matched_branch"
                        fi
                    fi
                done
            else
                # Check if branch exists in this repository
                if (cd "$repo_path" && git show-ref --verify --quiet "refs/heads/$branch" 2>/dev/null); then
                    if [ -z "$valid_branches" ]; then
                        valid_branches="$branch"
                    else
                        valid_branches="$valid_branches $branch"
                    fi
                fi
            fi
        done

        if [ -n "$valid_branches" ]; then
            branch_params="$valid_branches"
        else
            # No valid branches found in this repo, fallback to default (HEAD)
            branch_params=""
        fi
    else
        # No branches configured, use default (HEAD)
        branch_params=""
    fi

    # Build safe git log argument array
    git_log_args=()
    if [ "$MODE_TYPE" = "single" ]; then
        git_log_args+=(
            "--since=$STATS_DATE 00:00:00"
            "--until=$STATS_DATE 23:59:59"
        )
    else
        git_log_args+=(
            "--since=$STATS_SINCE 00:00:00"
            "--until=$STATS_UNTIL 23:59:59"
        )
    fi

    # Add --first-parent when using default branch, it's a safe default
    # When we have specific branches, don't add it to allow traversing all specified branches
    if [ -z "$branch_params" ]; then
        git_log_args+=("--first-parent")
    else
        for branch in $branch_params; do
            git_log_args+=("$branch")
        done
    fi

    git_log_args+=(
        "--format=%H"
        "--shortstat"
    )

    # Get all users statistics and commit count
    output=$(cd "$repo_path" && git log "${git_log_args[@]}" 2>/dev/null)
    stats=$(echo "$output" | awk '
        /^[0-9a-f]{40}$/ { commits++; next }
        /files? changed/ {
            files += $1
            match($0, /([0-9]+) insertion/)
            if (RSTART > 0) {
                num = substr($0, RSTART, RLENGTH)
                gsub(/[^0-9]/, "", num)
                added += num
            }
            match($0, /([0-9]+) deletion/)
            if (RSTART > 0) {
                num = substr($0, RSTART, RLENGTH)
                gsub(/[^0-9]/, "", num)
                deleted += num
            }
        } END {
            print files " " added " " deleted " " commits
        }')

    # Build author search - support multiple authors with OR
    author_args=()
    if [ -n "$AUTHOR" ]; then
        IFS=',' read -ra authors <<< "$AUTHOR"
        for auth in "${authors[@]}"; do
            auth=$(echo "$auth" | xargs)
            if [ -n "$auth" ]; then
                author_args+=("--author=$auth")
            fi
        done
    fi

    # Get current user statistics and commit count
    my_output=$(cd "$repo_path" && git log "${git_log_args[@]}" "${author_args[@]}" 2>/dev/null)
    my_stats=$(echo "$my_output" | awk '
        /^[0-9a-f]{40}$/ { commits++; next }
        /files? changed/ {
            files += $1
            match($0, /([0-9]+) insertion/)
            if (RSTART > 0) {
                num = substr($0, RSTART, RLENGTH)
                gsub(/[^0-9]/, "", num)
                added += num
            }
            match($0, /([0-9]+) deletion/)
            if (RSTART > 0) {
                num = substr($0, RSTART, RLENGTH)
                gsub(/[^0-9]/, "", num)
                deleted += num
            }
        } END {
            print files " " added " " deleted " " commits
        }')

    # Parse statistics result
    if [ -n "$stats" ]; then
        files=$(echo "$stats" | awk '{print $1}')
        add=$(echo "$stats" | awk '{print $2}')
        del=$(echo "$stats" | awk '{print $3}')
        commit_count=$(echo "$stats" | awk '{print $4}')
        files=${files:-0}
        add=${add:-0}
        del=${del:-0}
    else
        files=0
        add=0
        del=0
    fi

    # Parse my statistics
    if [ -n "$my_stats" ]; then
        my_files_local=$(echo "$my_stats" | awk '{print $1}')
        my_add_local=$(echo "$my_stats" | awk '{print $2}')
        my_del_local=$(echo "$my_stats" | awk '{print $3}')
        my_commit_count=$(echo "$my_stats" | awk '{print $4}')
        my_files_local=${my_files_local:-0}
        my_add_local=${my_add_local:-0}
        my_del_local=${my_del_local:-0}
    else
        my_files_local=0
        my_add_local=0
        my_del_local=0
    fi

    commit_count=${commit_count:-0}
    my_commit_count=${my_commit_count:-0}

    # Show repository if there are changes and not quiet
    if [ \( $files -gt 0 -o $add -gt 0 -o $del -gt 0 -o $my_add_local -gt 0 -o $my_del_local -gt 0 \) ] && [ "$OUTPUT_FORMAT" = "text" ] && [ $QUIET -eq 0 ]; then
        echo "=== $repo_name ==="
        echo "  文件变更: $files (我的: $my_files_local) | Files Changed: $files (Mine: $my_files_local)"
        echo "  新增行数: $add (我的: $my_add_local) | Lines Added: $add (Mine: $my_add_local)"
        echo "  删除行数: $del (我的: $my_del_local) | Lines Deleted: $del (Mine: $my_del_local)"
        echo "  净增行数: $((add - del)) (我的: $((my_add_local - my_del_local))) | Net Added: $((add - del)) (Mine: $((my_add_local - my_del_local)))"
        echo "  提交次数: $commit_count (我的: $my_commit_count) | Commits: $commit_count (Mine: $my_commit_count)"
        echo ""
    fi

    # Write to temp files for summary accumulation
    echo "$files $add $del $commit_count" >> "$tmp_file"
    echo "$my_files_local $my_add_local $my_del_local $my_commit_count" >> "$my_tmp_file"
done

# Accumulate totals from temp file
while read -r files add del commits; do
    repo_count=$((repo_count + 1))
    total_files=$((total_files + files))
    total_additions=$((total_additions + add))
    total_deletions=$((total_deletions + del))
    total_commits=$((total_commits + commits))
done < "$tmp_file"

# Accumulate my statistics
while read -r files add del commits; do
    my_files=$((my_files + files))
    my_additions=$((my_additions + add))
    my_deletions=$((my_deletions + del))
    my_commits=$((my_commits + commits))
done < "$my_tmp_file"

# Calculate percentage
if [ $total_additions -gt 0 ]; then
    percentage=$(awk "BEGIN {printf \"%.1f\", $my_additions * 100 / $total_additions}")
else
    percentage="N/A"
fi

total_net=$((total_additions - total_deletions))
my_net=$((my_additions - my_deletions))

# Output based on format
if [ "$OUTPUT_FORMAT" = "json" ]; then
    # JSON output (safe, no injection)
    json_output=$(jq -n \
      --arg dir "$SCAN_DIR" \
      --argjson depth "$SCAN_DEPTH" \
      --arg branches "$STATS_BRANCHES" \
      --arg exclusions "$EXCLUDE_DIRS" \
      --arg mode_type "$MODE_TYPE" \
      --arg single_date "$STATS_DATE" \
      --arg since "$STATS_SINCE" \
      --arg until "$STATS_UNTIL" \
      --arg author "$AUTHOR" \
      --argjson repo_count "$repo_count" \
      --argjson total_files "$total_files" \
      --argjson total_additions "$total_additions" \
      --argjson total_deletions "$total_deletions" \
      --argjson total_net "$total_net" \
      --argjson total_commits "$total_commits" \
      --argjson my_files "$my_files" \
      --argjson my_additions "$my_additions" \
      --argjson my_deletions "$my_deletions" \
      --argjson my_net "$my_net" \
      --argjson my_commits "$my_commits" \
      --arg percentage "$percentage" \
      --argjson is_workday "$IS_WORKDAY" \
      --argjson standard "$DAILY_CODE_STANDARD" \
      '{
        scan: {
          directory: $dir,
          depth: $depth,
          branches: $branches,
          exclusions: $exclusions
        },
        date: {
          type: $mode_type,
          single_date: (if $mode_type == "single" then $single_date else null end),
          since: (if $mode_type == "range" then $since else null end),
          until: (if $mode_type == "range" then $until else null end)
        },
        author: $author,
        summary: {
          repository_count: $repo_count,
          all: {
            files_changed: $total_files,
            additions: $total_additions,
            deletions: $total_deletions,
            net: $total_net,
            commits: $total_commits
          },
          author: {
            files_changed: $my_files,
            additions: $my_additions,
            deletions: $my_deletions,
            net: $my_net,
            commits: $my_commits,
            percentage_of_total: (if $percentage == "N/A" then null else ($percentage | tonumber) end)
          }
        },
        workday: {
          is_workday: (if $mode_type == "single" then $is_workday else null end),
          standard: $standard,
          progress: (if $mode_type == "single" and $standard > 0 then ($my_additions * 100.0 / $standard) else null end)
        }
      }'
    )
    if [ -n "$OUTPUT_FILE" ]; then
        echo "$json_output" > "$OUTPUT_FILE"
    else
        echo "$json_output"
    fi
elif [ "$OUTPUT_FORMAT" = "csv" ]; then
    # CSV output
    csv_header="timestamp,mode,date_start,date_end,author,repositories,total_files,total_additions,total_deletions,total_net,total_commits,my_files,my_additions,my_deletions,my_net,my_commits,percentage"
    csv_line="$(date +%Y-%m-%d_%H:%M:%S),$MODE_TYPE,"
    if [ "$MODE_TYPE" = "single" ]; then
        csv_line="$csv_line$STATS_DATE,$STATS_DATE,"
    else
        csv_line="$csv_line$STATS_SINCE,$STATS_UNTIL,"
    fi
    csv_line="$csv_line\"$AUTHOR\",$repo_count,$total_files,$total_additions,$total_deletions,$total_net,$total_commits,$my_files,$my_additions,$my_deletions,$my_net,$my_commits,$percentage"

    if [ -n "$OUTPUT_FILE" ]; then
        if [ ! -f "$OUTPUT_FILE" ]; then
            echo "$csv_header" > "$OUTPUT_FILE"
        fi
        echo "$csv_line" >> "$OUTPUT_FILE"
    else
        echo "$csv_header"
        echo "$csv_line"
    fi
else
    # Default text output
    if [ -n "$OUTPUT_FILE" ]; then
        # Capture text output to file, save original stdout
        exec 3>&1
        exec >"$OUTPUT_FILE"
    fi

    echo "=========================================="
    if [ "$MODE_TYPE" = "single" ]; then
        echo "统计汇总 ($STATS_DATE) | Summary ($STATS_DATE)"
    else
        echo "统计汇总 ($STATS_SINCE 至 $STATS_UNTIL) | Summary ($STATS_SINCE to $STATS_UNTIL)"
    fi
    echo "=========================================="
    echo "扫描仓库数量: $repo_count | Repositories Scanned: $repo_count"
    echo "------------------------------------------"
    echo "所有人汇总: | All Users:"
    echo "  文件变更总数: $total_files | Total Files Changed: $total_files"
    echo "  新增行数总计: $total_additions | Total Lines Added: $total_additions"
    echo "  删除行数总计: $total_deletions | Total Lines Deleted: $total_deletions"
    echo "  净增行数总计: $total_net | Net Lines Added: $total_net"
    echo "  提交次数总计: $total_commits | Total Commits: $total_commits"
    echo "------------------------------------------"
    echo "$AUTHOR 的贡献: | $AUTHOR's Contribution:"
    echo "  文件变更数: $my_files | Files Changed: $my_files"
    echo "  新增行数: $my_additions | Lines Added: $my_additions"
    echo "  删除行数: $my_deletions | Lines Deleted: $my_deletions"
    echo "  净增行数: $my_net | Net Lines Added: $my_net"
    echo "  提交次数: $my_commits | Commits: $my_commits"
    if [ "$percentage" != "N/A" ]; then
        echo "  占总新增: ${percentage}% | Percentage of Total Additions: ${percentage}%"
    else
        echo "  占总新增: N/A | Percentage of Total Additions: N/A"
    fi
    echo "=========================================="

    # Workday check only for single day mode
    if [ "$MODE_TYPE" = "single" ]; then
        echo ""
        echo "=========================================="
        echo "工作日代码量检查 | Workday Code Check"
        echo "=========================================="
        echo "标准要求: 每天 $DAILY_CODE_STANDARD 行代码 | Standard: $DAILY_CODE_STANDARD lines/day"
        echo "实际提交: $my_additions 行 | Actual: $my_additions lines"
        if [ $DAILY_CODE_STANDARD -gt 0 ]; then
            progress=$(awk "BEGIN {printf \"%.1f\", $my_additions * 100.0 / $DAILY_CODE_STANDARD}")
            echo "完成进度: ${progress}% | Progress: ${progress}%"
            echo ""
        fi

        if [ $IS_WORKDAY -eq 1 ]; then
            if [ $my_additions -lt $DAILY_CODE_STANDARD ]; then
                GAP=$((DAILY_CODE_STANDARD - my_additions))
                echo "[WARN] $DATE_DESC 未达到工作日代码量标准！"
                echo "[WARN] Warning: Workday code standard not met on $DATE_DESC!"
                echo "[WARN] 还需要补充: $GAP 行代码 | Need: $GAP more lines"
                echo "[WARN] 请努力赶上进度！ | Please catch up!"
                exit_code=1
            else
                echo "[OK] $DATE_DESC 已完成工作日代码量标准！继续保持！"
                echo "[OK] Workday code standard met! Keep it up!"
                exit_code=0
            fi
        else
            echo "日期类型: 周末 | Type: Weekend"
            echo "周末无强制代码量要求 | No mandatory requirement on weekends"
            echo "实际提交: $my_additions 行 | Actual: $my_additions lines"
            if [ $my_additions -gt 0 ]; then
                echo "[Great] 周末也这么努力，太棒了！"
                echo "[Great] Working hard on weekend too! Awesome!"
            fi
            exit_code=0
        fi
    fi

    if [ -n "$OUTPUT_FILE" ]; then
        # Restore original stdout for confirmation message
        echo "统计结果已写入: $OUTPUT_FILE | Results written to: $OUTPUT_FILE" >&3
        exec 3>&-
    fi
fi

exit ${exit_code:-0}
