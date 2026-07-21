#!/bin/bash
# sync-knowledge.sh — Sync Claude memory files to GitBoard project notes (CLI)
#
# DEPRECATED: prefer the in-app import (Settings -> 操作 -> 导入 Claude 记忆),
# which calls App.ImportClaudeMemory with parameterized, idempotent queries.
# This script is kept for headless/automation use; content is escaped by
# doubling single quotes (the SQL standard for string literals).
# Usage: bash sync-knowledge.sh
# This script reads Claude's project memory files and inserts them into
# GitBoard's SQLite database as project notes, making them visible in the UI.

set -euo pipefail

# === Config ===
CLAUDE_MEMORY_DIR="$HOME/.claude/projects"
GITBOARD_DB="${HOME}/Library/Application Support/gitboard/dashboard.db"
SQLITE=$(command -v sqlite3 || command -v sqlite3)

if [ -z "$SQLITE" ]; then
  echo "Error: sqlite3 not found. Install it first."
  exit 1
fi

if [ ! -f "$GITBOARD_DB" ]; then
  echo "Error: GitBoard database not found at $GITBOARD_DB"
  echo "Make sure GitBoard has been run at least once."
  exit 1
fi

echo "=== Syncing Claude memory to GitBoard ==="
echo "Memory dir: $CLAUDE_MEMORY_DIR"
echo "GitBoard DB: $GITBOARD_DB"
echo ""

SYNCED=0
SKIPPED=0

# Iterate over each project in Claude memory
for project_dir in "$CLAUDE_MEMORY_DIR"/*/; do
  project_name="$(basename "$project_dir")"
  memory_dir="${project_dir}memory"

  if [ ! -d "$memory_dir" ]; then
    continue
  fi

  # Extract the actual project name from the directory name
  # Claude project dirs are like: "-Users-jiangcheng-Workspace-Java-CBIBank-CBiUnion"
  # We extract the last meaningful part: "CBiUnion"
  display_name=""
  if [[ "$project_name" == -* ]]; then
    # Format: -Users-jiangcheng-Workspace-...-ProjectName
    display_name=$(echo "$project_name" | sed 's/^-//' | awk -F'-' '{print $NF}')
  else
    display_name="$project_name"
  fi

  # Find matching GitBoard project by name (try exact match, then fuzzy)
  project_id=$("$SQLITE" "$GITBOARD_DB" "SELECT id FROM projects WHERE name = '$display_name' OR name LIKE '%$display_name%' LIMIT 1" 2>/dev/null || echo "")

  if [ -z "$project_id" ]; then
    # Try matching by the full directory path
    full_path=$(echo "$project_name" | sed 's/^-//' | sed 's/-/\//g')
    project_id=$("$SQLITE" "$GITBOARD_DB" "SELECT p.id FROM projects p JOIN repositories r ON r.project_id = p.id WHERE r.path LIKE '%$full_path%' LIMIT 1" 2>/dev/null || echo "")
    # Try to match by the last path segment
    if [ -z "$project_id" ]; then
      project_id=$("$SQLITE" "$GITBOARD_DB" "SELECT p.id FROM projects p JOIN repositories r ON r.project_id = p.id WHERE r.path LIKE '%/$display_name' OR r.path LIKE '%/$display_name.git' LIMIT 1" 2>/dev/null || echo "")
    fi
  fi

  if [ -z "$project_id" ]; then
    echo "  ⚠  No matching GitBoard project for: $display_name (Claude dir: $project_name)"
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  echo "  ✓ Project: $display_name (ID: $project_id)"

  # Process each memory file
  for mem_file in "$memory_dir"/*.md; do
    [ -f "$mem_file" ] || continue
    filename=$(basename "$mem_file" .md)

    # Skip MEMORY.md (it's the index), but import it as a project overview note
    if [ "$filename" = "MEMORY" ]; then
      continue
    fi

    # Read file content, strip YAML frontmatter
    raw_content=$(cat "$mem_file")

    # Strip YAML frontmatter (between --- markers)
    body=$(echo "$raw_content" | awk 'BEGIN {in_front=0} /^---$/ {in_front++; next} in_front == 1 {next} in_front >= 2 {print}')

    # Add a title prefix based on the filename
    title=""
    case "$filename" in
      project) title="## 项目知识" ;;
      user) title="## 用户信息" ;;
      feedback) title="## 反馈记录" ;;
      reference) title="## 参考信息" ;;
      *) title="## $filename" ;;
    esac

    note_content="$title

$body"

    # Escape single quotes for SQL
    escaped_content=$(echo "$note_content" | sed "s/'/''/g")

    # Check if a note with this content already exists for this project
    existing=$("$SQLITE" "$GITBOARD_DB" "SELECT COUNT(*) FROM project_notes WHERE project_id = $project_id AND content LIKE '${title}%'" 2>/dev/null || echo "0")

    if [ "$existing" -gt 0 ]; then
      # Update existing note
      "$SQLITE" "$GITBOARD_DB" "UPDATE project_notes SET content = '$escaped_content', updated_at = datetime('now') WHERE project_id = $project_id AND content LIKE '${title}%' LIMIT 1" 2>/dev/null || true
      echo "    ↻ Updated: $filename"
    else
      # Insert new note
      "$SQLITE" "$GITBOARD_DB" "INSERT INTO project_notes (project_id, content, sort_order, created_at, updated_at) VALUES ($project_id, '$escaped_content', 0, datetime('now'), datetime('now'))" 2>/dev/null || true
      echo "    + Added: $filename"
    fi
    SYNCED=$((SYNCED + 1))
  done
done

echo ""
echo "=== Sync complete ==="
echo "  Synced: $SYNCED notes"
echo "  Skipped: $SKIPPED projects (no GitBoard match)"
echo ""
echo "Refresh GitBoard to see the knowledge notes."