import { Summary } from '../api/client'

interface Props {
  summary: Summary | null
  globalTodoCount?: number
}

function SkeletonItem() {
  return (
    <div className="summary-item">
      <div className="skeleton skeleton-text" style={{width: 48}} />
      <div className="skeleton skeleton-value" style={{width: 36}} />
    </div>
  )
}

function SummaryBar({ summary, globalTodoCount }: Props) {
  if (!summary) {
    return (
      <div className="summary-bar">
        <SkeletonItem />
        <SkeletonItem />
        <SkeletonItem />
        <SkeletonItem />
        <SkeletonItem />
        <SkeletonItem />
      </div>
    )
  }

  return (
    <div className="summary-bar">
      <div className="summary-item">
        <span className="summary-label">仓库</span>
        <span className="summary-value">{summary.repo_count}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">团队新增</span>
        <span className="summary-value green">+{summary.total_added}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">团队删除</span>
        <span className="summary-value red">-{summary.total_deleted}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">个人新增</span>
        <span className="summary-value green">{summary.my_added > 0 ? '+' : ''}{summary.my_added}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">个人文件</span>
        <span className="summary-value">{summary.my_files}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">日期</span>
        <span className="summary-value">{summary.is_workday ? '\u5DE5\u4F5C\u65E5' : '\u5468\u672B'}</span>
      </div>
      {globalTodoCount !== undefined && globalTodoCount > 0 && (
        <div className="summary-item">
          <span className="summary-label">待办</span>
          <span className="summary-value todo">{globalTodoCount}</span>
        </div>
      )}
    </div>
  )
}

export default SummaryBar
