import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { getProjectDetail, updateProjectLevel, ProjectDetail } from '../api/client'
import TrendChart, { TrendDataset } from '../components/TrendChart'
import Heatmap from '../components/Heatmap'
import StatusBar from '../components/StatusBar'
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
  const [range, setRange] = useState<'week' | 'month' | 'all'>('week')

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

  const stats = useMemo(() => {
    const map = new Map<string, { added: number; deleted: number; files: number; commits: number }>()
    if (project?.repos) {
      project.repos.forEach((repo) => {
        repo.stats?.forEach((stat) => {
          const cur = map.get(stat.stat_date) || { added: 0, deleted: 0, files: 0, commits: 0 }
          cur.added += stat.lines_added
          cur.deleted += stat.lines_deleted
          cur.files += stat.files_changed
          cur.commits++
          map.set(stat.stat_date, cur)
        })
      })
    }
    return map
  }, [project])

  const trendData = useMemo(() => {
    let dates = Array.from(stats.keys()).sort()
    if (range === 'week') {
      const weekDates = getLastDays(7)
      dates = dates.filter((d) => weekDates.includes(d))
    } else if (range === 'month') {
      const monthDates = getLastDays(30)
      dates = dates.filter((d) => monthDates.includes(d))
    }

    return {
      labels: dates,
      datasets: [
        { label: '新增行数', data: dates.map((d) => stats.get(d)!.added), color: '#4a7d4a' },
        { label: '删除行数', data: dates.map((d) => stats.get(d)!.deleted), color: '#c95757' },
        { label: '文件变更', data: dates.map((d) => stats.get(d)!.files), color: '#5a7fa0' },
      ] as TrendDataset[],
    }
  }, [stats, range])

  const totals = useMemo(() => {
    let added = 0, deleted = 0, files = 0, active = 0
    stats.forEach((v) => {
      added += v.added
      deleted += v.deleted
      files += v.files
      if (v.added + v.deleted > 0) active++
    })
    return { added, deleted, files, active, repos: project?.repos?.length || 0 }
  }, [stats, project])

  if (loading) {
    return (
      <div className="project-detail">
        <div className="project-fixed">
          <div className="skeleton skeleton-text" style={{width: 200, height: 28, marginBottom: 8}} />
          <div className="skeleton skeleton-text" style={{width: '50%', height: 14, marginBottom: 20}} />
          <div className="skeleton skeleton-text" style={{width: '100%', height: 80, marginBottom: 16}} />
        </div>
        <div className="project-scroll">
          <div className="skeleton skeleton-text" style={{width: '100%', height: 280, marginBottom: 16}} />
        </div>
        <StatusBar />
      </div>
    )
  }

  if (error || !project) {
    return (
      <div className="project-detail">
        <button className="btn btn-secondary back-btn" onClick={() => navigate('/')}>&larr; 返回仪表盘</button>
        <div className="error-banner">
          <span>{error || '项目未找到'}</span>
          <button className="btn btn-sm" onClick={() => id && getProjectDetail(Number(id)).then(setProject).catch(() => {})}>重试</button>
        </div>
        <StatusBar />
      </div>
    )
  }

  return (
    <div className="project-detail">
      <div className="project-fixed">
        <button className="btn btn-secondary back-btn" onClick={() => navigate('/')}>
          &larr; 返回仪表盘
        </button>

        <div className="detail-header-card">
          <div className="detail-title-row">
            <div>
              <h1>{project.name}</h1>
              <p className="detail-path">{project.root_path}</p>
            </div>
            <div className="detail-actions">
              <button className="btn btn-sm" onClick={() => handleLevelChange('down')}>
                向下拆分
              </button>
              <button className="btn btn-sm" onClick={() => handleLevelChange('up')}>
                向上合并
              </button>
            </div>
          </div>

          <div className="detail-stats-grid">
            <div className="detail-stat">
              <span className="stat-label">子仓库</span>
              <span className="stat-value">{totals.repos}</span>
            </div>
            <div className="detail-stat">
              <span className="stat-label">活跃天数</span>
              <span className="stat-value">{totals.active}</span>
            </div>
            <div className="detail-stat">
              <span className="stat-label">文件变更</span>
              <span className="stat-value">{totals.files}</span>
            </div>
            <div className="detail-stat">
              <span className="stat-label">新增</span>
              <span className="stat-value green">+{totals.added}</span>
            </div>
            <div className="detail-stat">
              <span className="stat-label">删除</span>
              <span className="stat-value red">-{totals.deleted}</span>
            </div>
          </div>

          <div className="detail-meta-row">
            <span className="meta-pill">{project.is_auto_grouped ? '自动分组' : '手动分组'}</span>
            {dateParam && <span className="meta-pill">日期: {dateParam}</span>}
          </div>
        </div>
      </div>

      <div className="project-scroll">
        <div className="detail-section">
          <div className="section-header">
            <h2>提交热力图</h2>
          </div>
          <Heatmap />
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
                className={`btn btn-sm ${range === 'month' ? 'btn-active' : ''}`}
                onClick={() => setRange('month')}
              >
                近30天
              </button>
              <button
                className={`btn btn-sm ${range === 'all' ? 'btn-active' : ''}`}
                onClick={() => setRange('all')}
              >
                全部
              </button>
            </div>
          </div>
          {trendData.labels.length > 0 ? (
            <TrendChart labels={trendData.labels} datasets={trendData.datasets} />
          ) : (
            <div className="empty-section">该时间范围内暂无数据</div>
          )}
        </div>

        <div className="project-layout">
          <div className="project-main">
            <div className="detail-section">
              <h2>子仓库 ({project.repos?.length || 0})</h2>
              <div className="repo-list">
                {(project.repos || []).map((repo) => {
                  const repoTotals = (repo.stats || []).reduce(
                    (acc, s) => ({
                      added: acc.added + s.lines_added,
                      deleted: acc.deleted + s.lines_deleted,
                      files: acc.files + s.files_changed,
                    }),
                    { added: 0, deleted: 0, files: 0 }
                  )
                  return (
                    <div key={repo.id} className="repo-item">
                      <div className="repo-header">
                        <div className="repo-path">{repo.path.split('/').slice(-2).join('/')}</div>
                        <div className="repo-totals">
                          <span className="green">+{repoTotals.added}</span>
                          <span className="red">-{repoTotals.deleted}</span>
                        </div>
                      </div>
                      {repo.stats && repo.stats.length > 0 && (
                        <div className="repo-stats">
                          {repo.stats.slice(0, 5).map((stat) => (
                            <span key={stat.id} className="stat-tag">
                              {stat.stat_date}: <span className="green">+{stat.lines_added}</span>{' '}
                              <span className="red">-{stat.lines_deleted}</span>
                            </span>
                          ))}
                          {repo.stats.length > 5 && (
                            <span className="stat-tag more">+{repo.stats.length - 5} 更多</span>
                          )}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>

          <ProjectPanel projectId={Number(id)} />
        </div>
      </div>

      <StatusBar />
    </div>
  )
}

export default ProjectDetailPage
