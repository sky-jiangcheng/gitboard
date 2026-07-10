import { useEffect, useState } from 'react'
import { getHeatmapData, type HeatmapDay } from '../api/client'

function getLevel(day: HeatmapDay): number {
  const total = day.lines_added + day.lines_deleted
  if (total === 0) return 0
  if (total < 100) return 1
  if (total < 300) return 2
  if (total < 600) return 3
  return 4
}

function getWeeks(days: HeatmapDay[]): (HeatmapDay | null)[][] {
  const result: (HeatmapDay | null)[][] = []
  let week: (HeatmapDay | null)[] = []

  // Pad to start on Sunday
  if (days.length > 0) {
    const first = new Date(days[0].date)
    const pad = first.getDay()
    for (let i = 0; i < pad; i++) {
      week.push(null)
    }
  }

  for (const day of days) {
    week.push(day)
    if (week.length === 7) {
      result.push(week)
      week = []
    }
  }

  if (week.length > 0) {
    while (week.length < 7) {
      week.push(null)
    }
    result.push(week)
  }

  return result
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function getTotalStats(days: HeatmapDay[]) {
  let added = 0, deleted = 0, commits = 0, active = 0
  for (const d of days) {
    added += d.lines_added
    deleted += d.lines_deleted
    commits += d.commits
    if (d.lines_added + d.lines_deleted > 0) active++
  }
  return { added, deleted, commits, active }
}

export default function Heatmap() {
  const [days, setDays] = useState<HeatmapDay[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getHeatmapData()
      .then(res => setDays(res.days))
      .catch(() => setDays([]))
      .finally(() => setLoading(false))
  }, [])

  const weeks = getWeeks(days)
  const stats = getTotalStats(days)

  if (loading) {
    return (
      <div className="heatmap-container">
        <div className="heatmap-loading">加载中...</div>
      </div>
    )
  }

  if (days.length === 0) {
    return (
      <div className="heatmap-container">
        <div className="heatmap-empty">暂无提交数据</div>
      </div>
    )
  }

  const monthLabels: { index: number; label: string }[] = []
  let lastMonth = -1
  weeks.forEach((week, weekIndex) => {
    const day = week.find(d => d !== null)
    if (day) {
      const month = new Date(day.date).getMonth()
      if (month !== lastMonth) {
        monthLabels.push({ index: weekIndex, label: new Date(day.date).toLocaleDateString('zh-CN', { month: 'short' }) })
        lastMonth = month
      }
    }
  })

  return (
    <div className="heatmap-container">
      <div className="heatmap-header">
        <h3 className="heatmap-title">提交热力图</h3>
        <div className="heatmap-stats">
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">活跃天数</span>
            <span className="heatmap-stat-value">{stats.active}</span>
          </div>
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">总提交</span>
            <span className="heatmap-stat-value">{stats.commits}</span>
          </div>
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">新增行</span>
            <span className="heatmap-stat-value">{stats.added.toLocaleString()}</span>
          </div>
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">删除行</span>
            <span className="heatmap-stat-value">{stats.deleted.toLocaleString()}</span>
          </div>
        </div>
      </div>

      <div className="heatmap-months">
        {monthLabels.map(m => (
          <span key={m.index} className="heatmap-month-label" style={{ left: `${m.index * 16}px` }}>
            {m.label}
          </span>
        ))}
      </div>

      <div className="heatmap-grid-wrapper">
        <div className="heatmap-day-labels">
          <span>日</span>
          <span></span>
          <span>二</span>
          <span></span>
          <span>四</span>
          <span></span>
          <span>六</span>
        </div>
        <div className="heatmap-grid">
          {weeks.map((week, wi) => (
            <div key={wi} className="heatmap-week">
              {week.map((day, di) => (
                <div
                  key={di}
                  className={`heatmap-cell level-${day ? getLevel(day) : 0}`}
                  title={day ? `${formatDate(day.date)}: +${day.lines_added} -${day.lines_deleted} (${day.commits} 次提交)` : ''}
                />
              ))}
            </div>
          ))}
        </div>
      </div>

      <div className="heatmap-legend">
        <span>少</span>
        <div className="heatmap-cell level-0" />
        <div className="heatmap-cell level-1" />
        <div className="heatmap-cell level-2" />
        <div className="heatmap-cell level-3" />
        <div className="heatmap-cell level-4" />
        <span>多</span>
      </div>
    </div>
  )
}
