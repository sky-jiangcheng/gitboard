import { Link } from 'react-router-dom'
import { Project } from '../api/client'

interface Props {
  project: Project
}

function ProjectCard({ project }: Props) {
  const netAdded = project.my_added - project.my_deleted

  return (
    <Link to={`/project/${project.id}`} className="project-card">
      <div className="card-header">
        <h3>{project.name}</h3>
        {project.below_standard && <span className="badge badge-warning">未达标</span>}
      </div>
      <div className="card-body">
        <div className="stat-row">
          <span className="stat-label">仓库数</span>
          <span className="stat-value">{project.repo_count}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">新增行数</span>
          <span className="stat-value green">+{project.my_added}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">删除行数</span>
          <span className="stat-value red">-{project.my_deleted}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">净增行数</span>
          <span className="stat-value">{netAdded >= 0 ? '+' : ''}{netAdded}</span>
        </div>
        <div className="stat-row">
          <span className="stat-label">文件变更</span>
          <span className="stat-value">{project.my_files}</span>
        </div>
      </div>
    </Link>
  )
}

export default ProjectCard
