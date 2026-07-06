import { Summary } from '../api/client'

interface Props {
  summary: Summary | null
}

function SummaryBar({ summary }: Props) {
  if (!summary) {
    return (
      <div className="summary-bar">
        <div className="summary-item">
          <span className="summary-label">加载中...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="summary-bar">
      <div className="summary-item">
        <span className="summary-label">仓库总数</span>
        <span className="summary-value">{summary.repo_count}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">总新增行</span>
        <span className="summary-value green">+{summary.total_added}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">总删除行</span>
        <span className="summary-value red">-{summary.total_deleted}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">个人新增</span>
        <span className="summary-value">{summary.my_added}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">文件变更</span>
        <span className="summary-value">{summary.my_files}</span>
      </div>
      <div className="summary-item">
        <span className="summary-label">日期类型</span>
        <span className="summary-value">{summary.is_workday ? '工作日' : '周末'}</span>
      </div>
    </div>
  )
}

export default SummaryBar
