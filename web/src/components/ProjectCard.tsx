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
      <div className="card-body">
        <div className="stat-row">
          <span className="stat-label">仓库数</span>
          <span className="stat-value">{project.repo_count}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">个人新增</span>
          <span className="stat-value green">+{project.my_added}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">个人删除</span>
          <span className="stat-value red">-{project.my_deleted}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">净增</span>
          <span className="stat-value">{netAdded >= 0 ? '+' : ''}{netAdded}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">文件变更</span>
          <span className="stat-value">{project.my_files}</span>
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
