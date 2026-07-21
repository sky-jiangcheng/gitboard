import { Link } from 'react-router-dom'
import { Project } from '../api/client'

interface Props {
  project: Project
  date?: string
  todoCount?: number
  noteCount?: number
  dailyGoal?: number
  isWorkday?: boolean
  onToggleStar?: (id: number) => void
}

function ProjectCard({ project, date, todoCount, noteCount, dailyGoal = 0, isWorkday = true, onToggleStar }: Props) {
  const netAdded = project.my_added - project.my_deleted
  const to = date ? `/project/${project.id}?date=${date}` : `/project/${project.id}`

  const contributionRatio = project.my_added + project.my_deleted > 0
    ? Math.round((project.my_added / (project.my_added + project.my_deleted)) * 100)
    : 50

  // Goal progress: how much of the daily target this project's additions represent.
  const goalPct = dailyGoal > 0 ? Math.min(Math.round((project.my_added / dailyGoal) * 100), 100) : 0
  const reachedGoal = isWorkday && project.my_added > 0 && project.my_added >= dailyGoal

  const handleStarClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onToggleStar?.(project.id)
  }

  return (
    <Link to={to} className={`project-card ${reachedGoal ? 'card-goal-reached' : ''}`}>
      <button
        className={`card-star ${project.is_starred ? 'starred' : ''}`}
        onClick={handleStarClick}
        title={project.is_starred ? '取消关注' : '关注项目'}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill={project.is_starred ? 'currentColor' : 'none'} stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
        </svg>
      </button>
      <div className="card-header">
        <h3>{project.name}</h3>
        <div className="card-badges">
          {reachedGoal && <span className="badge badge-goal" title="已达成今日目标">达标</span>}
          {noteCount !== undefined && noteCount > 0 && (
            <span className="badge badge-note" title="知识笔记">{noteCount}</span>
          )}
          {todoCount !== undefined && todoCount > 0 && (
            <span className="badge badge-todo">{todoCount}</span>
          )}
          {project.below_standard && <span className="badge badge-warning">未达标</span>}
          {!project.is_workday && <span className="badge badge-info">非工作日</span>}
        </div>
      </div>

      <div className="card-hero-num">
        <span className="card-hero-label">今日新增</span>
        <span className={`card-hero-value ${project.my_added > 0 ? 'green' : ''}`}>+{project.my_added}</span>
      </div>

      {isWorkday && dailyGoal > 0 && project.my_added > 0 && (
        <div className="card-goal-bar">
          <div className="card-goal-track">
            <div className="card-goal-fill" style={{ width: `${goalPct}%` }} />
          </div>
          <span className="card-goal-pct">{goalPct}% 目标</span>
        </div>
      )}

      <div className="card-grid">
        <div className="card-stat">
          <span className="stat-label">仓库</span>
          <span className="stat-value">{project.repo_count}</span>
        </div>
        <div className="card-stat">
          <span className="stat-label">文件</span>
          <span className="stat-value">{project.my_files}</span>
        </div>
        <div className="card-stat">
          <span className="stat-label">新增</span>
          <span className="stat-value green">+{project.my_added}</span>
        </div>
        <div className="card-stat">
          <span className="stat-label">删除</span>
          <span className="stat-value red">-{project.my_deleted}</span>
        </div>
      </div>

      <div className="card-progress">
        <div className="progress-bar">
          <div className="progress-fill" style={{ width: `${contributionRatio}%` }} />
          <div className="progress-fill deleted" style={{ width: `${100 - contributionRatio}%` }} />
        </div>
        <div className="progress-info">
          <span className="progress-label">净增</span>
          <span className={`progress-value ${netAdded >= 0 ? 'green' : 'red'}`}>
            {netAdded >= 0 ? '+' : ''}{netAdded}
          </span>
        </div>
      </div>

      <div className="card-footer">
        {project.total_added > 0 && (
          <span className="stat-tag team">团队 +{project.total_added}</span>
        )}
      </div>
    </Link>
  )
}

export default ProjectCard
