import { Link } from 'react-router-dom'
import { Project } from '../api/client'

interface Props {
  project: Project
  date?: string
  todoCount?: number
}

function ProjectCard({ project, date, todoCount }: Props) {
  const netAdded = project.my_added - project.my_deleted
  const to = date ? `/project/${project.id}?date=${date}` : `/project/${project.id}`

  const contributionRatio = project.my_added + project.my_deleted > 0
    ? Math.round((project.my_added / (project.my_added + project.my_deleted)) * 100)
    : 50

  return (
    <Link to={to} className="project-card">
      <div className="card-header">
        <h3>{project.name}</h3>
        <div className="card-badges">
          {todoCount !== undefined && todoCount > 0 && (
            <span className="badge badge-todo">{todoCount}</span>
          )}
          {project.below_standard && <span className="badge badge-warning">未达标</span>}
          {!project.is_workday && <span className="badge badge-info">非工作日</span>}
        </div>
      </div>

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
          <div 
            className="progress-fill" 
            style={{ width: `${contributionRatio}%` }}
          />
          <div 
            className="progress-fill deleted" 
            style={{ width: `${100 - contributionRatio}%` }}
          />
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
