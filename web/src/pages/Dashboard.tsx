import { useState, useEffect, useRef, useMemo } from 'react'
import { getProjects, getSummary, triggerScan, getTodoCounts, getScanStatus, Project, Summary, TodoCount } from '../api/client'
import SummaryBar from '../components/SummaryBar'
import Heatmap from '../components/Heatmap'
import StatusBar from '../components/StatusBar'
import DatePicker from '../components/DatePicker'
import ProjectCard from '../components/ProjectCard'

function getYesterday(): string {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return d.toISOString().split('T')[0]
}

type SortKey = 'name' | 'my_added' | 'my_files' | 'repo_count'

const SORT_OPTIONS: { key: SortKey; label: string }[] = [
  { key: 'name', label: '名称' },
  { key: 'my_added', label: '新增行数' },
  { key: 'my_files', label: '文件变更' },
  { key: 'repo_count', label: '仓库数' },
]

function Dashboard() {
  const [projects, setProjects] = useState<Project[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [date, setDate] = useState(getYesterday())
  const [loading, setLoading] = useState(true)
  const [scanning, setScanning] = useState(false)
  const [error, setError] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('my_added')
  const [confirmScan, setConfirmScan] = useState(false)
  const [todoCounts, setTodoCounts] = useState<TodoCount[]>([])
  const pollTimer = useRef<number | null>(null)

  const fetchData = async (selectedDate: string) => {
    setLoading(true)
    setError('')
    try {
      const [projData, sumData, counts] = await Promise.all([
        getProjects(selectedDate),
        getSummary(selectedDate),
        getTodoCounts(),
      ])
      setProjects(projData)
      setSummary(sumData)
      setTodoCounts(counts)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData(date)
    checkScanStatus()
    return () => {
      if (pollTimer.current) clearInterval(pollTimer.current)
    }
  }, [date])

  const checkScanStatus = async () => {
    try {
      const status = await getScanStatus()
      if (status.running) {
        setScanning(true)
        if (!pollTimer.current) {
          pollTimer.current = window.setInterval(async () => {
            const s = await getScanStatus()
            if (!s.running) {
              if (pollTimer.current) clearInterval(pollTimer.current)
              pollTimer.current = null
              setScanning(false)
              fetchData(date)
            }
          }, 2000)
        }
      }
    } catch {
      // ignore
    }
  }

  const handleScan = async () => {
    setConfirmScan(false)
    setError('')
    try {
      await triggerScan()
      setScanning(true)
      if (pollTimer.current) clearInterval(pollTimer.current)
      pollTimer.current = window.setInterval(async () => {
        const s = await getScanStatus()
        if (!s.running) {
          if (pollTimer.current) clearInterval(pollTimer.current)
          pollTimer.current = null
          setScanning(false)
          fetchData(date)
        }
      }, 2000)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '扫描失败'
      setError(msg)
    }
  }

  const sorted = useMemo(() => {
    const list = [...projects]
    list.sort((a, b) => {
      switch (sortKey) {
        case 'name':
          return a.name.localeCompare(b.name)
        case 'my_added':
          return b.my_added - a.my_added
        case 'my_files':
          return b.my_files - a.my_files
        case 'repo_count':
          return b.repo_count - a.repo_count
        default:
          return 0
      }
    })
    return list
  }, [projects, sortKey])

  const todoMap = useMemo(() => {
    const map = new Map<number, number>()
    todoCounts.forEach(c => map.set(c.project_id, c.count))
    return map
  }, [todoCounts])

  const globalTodoCount = useMemo(() => {
    return todoCounts.reduce((sum, c) => sum + c.count, 0)
  }, [todoCounts])

  return (
    <div className="dashboard">
      <div className="dashboard-fixed">
        <SummaryBar summary={summary} globalTodoCount={globalTodoCount} />
        <Heatmap />

        <div className="dashboard-controls">
          <DatePicker value={date} onChange={setDate} />
          <div className="dashboard-actions">
            <div className="sort-control">
              <label>排序：</label>
              <select
                value={sortKey}
                onChange={(e) => setSortKey(e.target.value as SortKey)}
                className="form-input sort-select"
              >
                {SORT_OPTIONS.map((opt) => (
                  <option key={opt.key} value={opt.key}>{opt.label}</option>
                ))}
              </select>
            </div>
            {confirmScan ? (
              <div className="confirm-group">
                <span className="confirm-text">确定重新扫描？</span>
                <button className="btn btn-primary btn-sm" onClick={handleScan} disabled={scanning}>
                  确认
                </button>
                <button className="btn btn-sm" onClick={() => setConfirmScan(false)}>
                  取消
                </button>
              </div>
            ) : (
              <button className="btn btn-primary" onClick={() => setConfirmScan(true)} disabled={scanning}>
                {scanning ? '扫描中...' : '重新扫描'}
              </button>
            )}
          </div>
        </div>

        {error && (
          <div className="error-banner">
            <span>{error}</span>
            <button className="btn btn-sm" onClick={() => fetchData(date)}>重试</button>
          </div>
        )}
      </div>

      <div className="dashboard-scroll">
        {loading ? (
          <div className="project-grid">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="project-card skeleton-card">
                <div className="card-header">
                  <div className="skeleton skeleton-text" style={{width: '60%', height: 20}} />
                </div>
                <div className="card-grid">
                  {Array.from({ length: 4 }).map((_, j) => (
                    <div key={j} className="card-stat">
                      <div className="skeleton skeleton-text" style={{width: 32, height: 10}} />
                      <div className="skeleton skeleton-text" style={{width: 40, height: 16}} />
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : sorted.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">&#128269;</div>
            <h3>暂无项目数据</h3>
            <p>GitBoard 尚未扫描到任何 Git 仓库。请先配置扫描目录。</p>
            <div className="empty-actions">
              <button className="btn btn-primary" onClick={() => setConfirmScan(true)}>
                开始扫描
              </button>
              <a href="/settings" className="btn btn-secondary">
                配置目录
              </a>
            </div>
          </div>
        ) : (
          <div className="project-grid">
            {sorted.map((p) => (
              <ProjectCard key={p.id} project={p} date={date} todoCount={todoMap.get(p.id)} />
            ))}
          </div>
        )}
      </div>

      <StatusBar />
    </div>
  )
}

export default Dashboard
