import { useEffect, useState } from 'react'
import { getHeatmapData, type HeatmapDay } from '../api/client'

const WEEKS = 52
const DAYS_PER_WEEK = 7

interface Props {
  onDayClick?: (date: string) => void
}

function getLevel(day: HeatmapDay | null): number {
  if (!day) return 0
  const total = day.lines_added + day.lines_deleted
  if (total === 0) return 0
  if (total < 100) return 1
  if (total < 300) return 2
  if (total < 600) return 3
  return 4
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

// Generate fixed 52 weeks grid ending today
function generateGrid(days: HeatmapDay[]): (HeatmapDay | null)[][] {
  const dayMap = new Map<string, HeatmapDay>()
  for (const d of days) {
    dayMap.set(d.date, d)
  }

  const grid: (HeatmapDay | null)[][] = []
  const today = new Date()
  today.setHours(0, 0, 0, 0)

  // Calculate the Sunday of the week containing 52 weeks ago
  const endDate = new Date(today)
  const startDate = new Date(today)
  startDate.setDate(startDate.getDate() - (WEEKS * 7 - 1))

  // Adjust start date to Sunday
  const startDay = startDate.getDay()
  startDate.setDate(startDate.getDate() - startDay)

  for (let w = 0; w < WEEKS; w++) {
    const week: (HeatmapDay | null)[] = []
    for (let d = 0; d < DAYS_PER_WEEK; d++) {
      const date = new Date(startDate)
      date.setDate(date.getDate() + w * 7 + d)
      const dateStr = date.toISOString().split('T')[0]
      week.push(dayMap.get(dateStr) || null)
    }
    grid.push(week)
  }

  return grid
}

export default function Heatmap({ onDayClick }: Props) {
  const [days, setDays] = useState<HeatmapDay[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getHeatmapData()
      .then(res => setDays(res.days))
      .catch(() => setDays([]))
      .finally(() => setLoading(false))
  }, [])

  const grid = generateGrid(days)
  const stats = getTotalStats(days)

  if (loading) {
    return (
      <div className="heatmap-container">
        <div className="heatmap-loading">加载中...</div>
      </div>
    )
  }

  // Generate month labels at correct positions
  const monthLabels: { index: number; label: string }[] = []
  let lastMonth = -1
  grid.forEach((week, weekIndex) => {
    // Find first non-null day in week to get the month
    for (const day of week) {
      if (day) {
        const month = new Date(day.date).getMonth()
        if (month !== lastMonth) {
          monthLabels.push({ 
            index: weekIndex, 
            label: new Date(day.date).toLocaleDateString('zh-CN', { month: 'short' }) 
          })
          lastMonth = month
        }
        break
      }
    }
  })

  return (
    <div className="heatmap-container">
      <div className="heatmap-header">
        <h3 className="heatmap-title">提交热力图</h3>
        <div className="heatmap-stats">
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">活跃</span>
            <span className="heatmap-stat-value">{stats.active}</span>
          </div>
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">提交</span>
            <span className="heatmap-stat-value">{stats.commits}</span>
          </div>
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">新增</span>
            <span className="heatmap-stat-value">{stats.added.toLocaleString()}</span>
          </div>
          <div className="heatmap-stat">
            <span className="heatmap-stat-label">删除</span>
            <span className="heatmap-stat-value">{stats.deleted.toLocaleString()}</span>
          </div>
        </div>
      </div>

      <div className="heatmap-content">
        <div className="heatmap-months">
          {monthLabels.map(m => (
            <span 
              key={m.index} 
              className="heatmap-month-label" 
              style={{ left: `${m.index * 15 + 28}px` }}
            >
              {m.label}
            </span>
          ))}
        </div>

        <div className="heatmap-grid-wrapper">
          <div className="heatmap-day-labels">
            <span></span>
            <span>一</span>
            <span></span>
            <span>三</span>
            <span></span>
            <span>五</span>
            <span></span>
          </div>
          <div className="heatmap-grid">
            {grid.map((week, wi) => (
              <div key={wi} className="heatmap-week">
                {week.map((day, di) => (
                  <div
                    key={di}
                    className={`heatmap-cell level-${getLevel(day)}`}
                    title={day ? `${formatDate(day.date)}: +${day.lines_added} -${day.lines_deleted} (${day.commits} 次提交)${onDayClick ? ' — 点击查看' : ''}` : ''}
                    onClick={day && onDayClick ? () => onDayClick(day.date) : undefined}
                    role={day && onDayClick ? 'button' : undefined}
                  />
                ))}
              </div>
            ))}
          </div>
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