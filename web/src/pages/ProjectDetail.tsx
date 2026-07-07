import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getProjectDetail, updateProjectLevel, ProjectDetail } from '../api/client'
import TrendChart from '../components/TrendChart'

function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    setLoading(true)
    getProjectDetail(Number(id))
      .then(setProject)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [id])

  const handleLevelChange = async (direction: 'up' | 'down') => {
    if (!id) return
    try {
      await updateProjectLevel(Number(id), direction)
      const updated = await getProjectDetail(Number(id))
      setProject(updated)
    } catch (e: any) {
      setError(e.message)
    }
  }

  if (loading) return <div className="loading">加载中...</div>
  if (error) return <div className="error-banner">{error}</div>
  if (!project) return <div className="error-banner">项目未找到</div>

  // Prepare trend data from daily stats
  const trendData = new Map<string, number>()
  if (project.repos) {
    project.repos.forEach((repo) => {
      if (repo.stats) {
        repo.stats.forEach((stat) => {
          const existing = trendData.get(stat.stat_date) || 0
          trendData.set(stat.stat_date, existing + stat.lines_added)
        })
      }
    })
  }
  const sortedDates = Array.from(trendData.keys()).sort()
  const trendValues = sortedDates.map((d) => trendData.get(d) || 0)

  return (
    <div className="project-detail">
      <button className="btn btn-secondary" onClick={() => navigate('/')}>
        &larr; 返回仪表盘
      </button>

      <div className="detail-header">
        <h1>{project.name}</h1>
        <p className="detail-path">{project.root_path}</p>
        <div className="level-controls">
          <span>项目级别调整：</span>
          <button className="btn btn-sm" onClick={() => handleLevelChange('up')}>
            向上合并
          </button>
          <button className="btn btn-sm" onClick={() => handleLevelChange('down')}>
            向下拆分
          </button>
          <span className="level-badge">
            {project.is_auto_grouped ? '自动分组' : '手动调整'} (偏移: {project.level_override})
          </span>
        </div>
      </div>

      <div className="detail-section">
        <h2>趋势图</h2>
        <TrendChart labels={sortedDates} values={trendValues} />
      </div>

      <div className="detail-section">
        <h2>子仓库 ({project.repos?.length || 0})</h2>
        <div className="repo-list">
          {(project.repos || []).map((repo) => (
            <div key={repo.id} className="repo-item">
              <div className="repo-path">{repo.path}</div>
              <div className="repo-stats">
                {(repo.stats && repo.stats.length > 0) ? (
                  repo.stats.map((stat) => (
                    <span key={stat.id} className="stat-tag">
                      {stat.stat_date}: +{stat.lines_added} -{stat.lines_deleted} ({stat.author})
                    </span>
                  ))
                ) : (
                  <span className="stat-tag">暂无统计</span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default ProjectDetailPage
