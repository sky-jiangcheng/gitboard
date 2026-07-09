import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { getProjectDetail, updateProjectLevel, ProjectDetail } from '../api/client'
import TrendChart, { TrendDataset } from '../components/TrendChart'
import ProjectPanel from '../components/ProjectPanel'

function getLastDays(n: number): string[] {
  const result: string[] = []
  const now = new Date()
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    result.push(d.toISOString().split('T')[0])
  }
  return result
}

function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const dateParam = searchParams.get('date') || ''

  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [range, setRange] = useState<'week' | 'all'>('week')

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError('')
    getProjectDetail(Number(id))
      .then(setProject)
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
      .finally(() => setLoading(false))
  }, [id])

  const handleLevelChange = async (direction: 'up' | 'down') => {
    if (!id) return
    try {
      await updateProjectLevel(Number(id), direction)
      const updated = await getProjectDetail(Number(id))
      setProject(updated)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '操作失败'
      setError(msg)
    }
  }

  const trendData = useMemo(() => {
    const map = new Map<string, { added: number; deleted: number; files: number }>()
    if (project?.repos) {
      project.repos.forEach((repo) => {
        repo.stats?.forEach((stat) => {
          const cur = map.get(stat.stat_date) || { added: 0, deleted: 0, files: 0 }
          cur.added += stat.lines_added
          cur.deleted += stat.lines_deleted
          cur.files += stat.files_changed
          map.set(stat.stat_date, cur)
        })
      })
    }

    let dates = Array.from(map.keys()).sort()
    if (range === 'week') {
      const weekDates = getLastDays(7)
      dates = dates.filter((d) => weekDates.includes(d))
    }

    return {
      labels: dates,
      datasets: [
        { label: '新增行数', data: dates.map((d) => map.get(d)!.added), color: '#4caf50' },
        { label: '删除行数', data: dates.map((d) => map.get(d)!.deleted), color: '#f44336' },
        { label: '文件变更', data: dates.map((d) => map.get(d)!.files), color: '#2196f3' },
      ] as TrendDataset[],
    }
  }, [project, range])

  if (loading) {
    return (
      <div className="project-detail">
        <div className="skeleton skeleton-text" style={{width: 120, marginBottom: 16}} />
        <div className="project-layout">
          <div className="project-main">
            <div className="skeleton skeleton-text" style={{width: '40%', height: 28, marginBottom: 8}} />
            <div className="skeleton skeleton-text" style={{width: '60%', height: 16, marginBottom: 24}} />
            <div className="skeleton skeleton-text" style={{width: '100%', height: 280, marginBottom: 24}} />
            <div className="skeleton skeleton-text" style={{width: 80, height: 20, marginBottom: 12}} />
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="skeleton skeleton-text" style={{width: '100%', height: 48, marginBottom: 8}} />
            ))}
          </div>
          <div className="side-panel">
            <div className="skeleton skeleton-text" style={{width: '60%', height: 20, marginBottom: 12}} />
            <div className="skeleton skeleton-text" style={{width: '100%', height: 36, marginBottom: 8}} />
            <div className="skeleton skeleton-text" style={{width: '100%', height: 36}} />
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="project-detail">
        <button className="btn btn-secondary" onClick={() => navigate('/')}>&larr; 返回仪表盘</button>
        <div className="error-banner">
          <span>{error}</span>
          <button className="btn btn-sm" onClick={() => id && getProjectDetail(Number(id)).then(setProject).catch(() => {})}>重试</button>
        </div>
      </div>
    )
  }

  if (!project) return <div className="error-banner">项目未找到</div>

  return (
    <div className="project-detail">
      <button className="btn btn-secondary" onClick={() => navigate('/')}>
        &larr; 返回仪表盘
      </button>

      <div className="project-layout">
        <div className="project-main">
          <div className="detail-header">
            <h1>{project.name}</h1>
            <p className="detail-path">{project.root_path}</p>
            <div className="detail-meta">
              <span className="meta-tag">子仓库 {project.repos?.length || 0} 个</span>
              <span className="meta-tag">{project.is_auto_grouped ? '自动分组' : '手动分组'}</span>
              {dateParam && <span className="meta-tag">日期: {dateParam}</span>}
            </div>
            <div className="level-controls">
              <span>调整分组层级：</span>
              <button className="btn btn-sm" onClick={() => handleLevelChange('up')}>
                向上合并
              </button>
              <button className="btn btn-sm" onClick={() => handleLevelChange('down')}>
                向下拆分
              </button>
            </div>
          </div>

          <div className="detail-section">
            <div className="section-header">
              <h2>趋势图</h2>
              <div className="range-toggle">
                <button
                  className={`btn btn-sm ${range === 'week' ? 'btn-active' : ''}`}
                  onClick={() => setRange('week')}
                >
                  近7天
                </button>
                <button
                  className={`btn btn-sm ${range === 'all' ? 'btn-active' : ''}`}
                  onClick={() => setRange('all')}
                >
                  全部
                </button>
              </div>
            </div>
            <TrendChart labels={trendData.labels} datasets={trendData.datasets} />
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
                          {stat.stat_date}: <span className="green">+{stat.lines_added}</span>{' '}
                          <span className="red">-{stat.lines_deleted}</span>{' '}
                          ({stat.author})
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

        <ProjectPanel projectId={Number(id)} />
      </div>
    </div>
  )
}

export default ProjectDetailPage
