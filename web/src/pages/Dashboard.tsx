import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import {
  getProjects, getSummary, triggerScan, getTodoCounts, getNoteCounts, searchAll,
  getScanStatus, toggleStar, getConfig, Project, Summary, TodoCount, NoteCount, SearchHit,
} from '../api/client'
import SummaryBar from '../components/SummaryBar'
import GoalRing from '../components/GoalRing'
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
  const [dailyGoal, setDailyGoal] = useState(500)
  const [date, setDate] = useState(getYesterday())
  const [loading, setLoading] = useState(true)
  const [scanning, setScanning] = useState(false)
  const [scanMsg, setScanMsg] = useState('')
  const [error, setError] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('my_added')
  const [confirmScan, setConfirmScan] = useState(false)
  const [todoCounts, setTodoCounts] = useState<TodoCount[]>([])
  const [noteCounts, setNoteCounts] = useState<NoteCount[]>([])
  const [showStarredOnly, setShowStarredOnly] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<SearchHit[] | null>(null)
  const [searching, setSearching] = useState(false)
  const pollTimer = useRef<number | null>(null)
  const searchRef = useRef<HTMLDivElement>(null)

  const fetchData = async (selectedDate: string, starredOnly = showStarredOnly) => {
    setLoading(true)
    setError('')
    try {
      const [projData, sumData, counts, noteCountsData] = await Promise.all([
        getProjects(selectedDate, starredOnly),
        getSummary(selectedDate),
        getTodoCounts(),
        getNoteCounts(),
      ])
      setProjects(projData)
      setSummary(sumData)
      setTodoCounts(counts)
      setNoteCounts(noteCountsData)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const checkScanStatus = useCallback(async () => {
    try {
      const status = await getScanStatus()
      if (status.running || status.backfilling) {
        setScanning(true)
        setScanMsg(status.message)
        if (!pollTimer.current) {
          pollTimer.current = window.setInterval(async () => {
            const s = await getScanStatus()
            if (!s.running && !s.backfilling) {
              if (pollTimer.current) clearInterval(pollTimer.current)
              pollTimer.current = null
              setScanning(false)
              setScanMsg('')
              fetchData(date, showStarredOnly)
            } else {
              setScanMsg(s.message)
            }
          }, 2000)
        }
      }
    } catch { /* ignore */ }
  }, [date, showStarredOnly])

  useEffect(() => {
    getConfig()
      .then(c => {
        const v = parseInt(c.config.daily_code_standard || '500', 10)
        if (!isNaN(v) && v > 0) setDailyGoal(v)
      })
      .catch(() => {})
    fetchData(date, showStarredOnly)
    checkScanStatus()
    return () => { if (pollTimer.current) clearInterval(pollTimer.current) }
  }, [date, showStarredOnly, checkScanStatus])

  const handleScan = async () => {
    setConfirmScan(false)
    setError('')
    try {
      await triggerScan()
      setScanning(true)
      setScanMsg('正在扫描仓库…')
      if (pollTimer.current) clearInterval(pollTimer.current)
      pollTimer.current = window.setInterval(async () => {
        const s = await getScanStatus()
        if (!s.running && !s.backfilling) {
          if (pollTimer.current) clearInterval(pollTimer.current)
          pollTimer.current = null
          setScanning(false)
          setScanMsg('')
          fetchData(date)
        } else {
          setScanMsg(s.message)
        }
      }, 2000)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '扫描失败')
    }
  }

  const handleToggleStar = async (projectId: number) => {
    try {
      const newStarred = await toggleStar(projectId)
      setProjects(prev => prev.map(p => p.id === projectId ? { ...p, is_starred: newStarred } : p))
      if (showStarredOnly && !newStarred) {
        setProjects(prev => prev.filter(p => p.id !== projectId))
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '操作失败')
    }
  }

  const handleSearch = useCallback(async (query: string) => {
    if (!query.trim()) { setSearchResults(null); setSearching(false); return }
    setSearching(true)
    try {
      const results = await searchAll(query)
      setSearchResults(results)
    } catch {
      setSearchResults([])
    } finally {
      setSearching(false)
    }
  }, [])

  // Debounced search wrapper with proper cleanup
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()
  const handleSearchDebounced = useCallback((query: string) => {
    setSearchQuery(query)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (!query.trim()) { setSearchResults(null); setSearching(false); return }
    setSearching(true)
    debounceRef.current = setTimeout(() => {
      handleSearch(query)
    }, 300)
  }, [handleSearch])

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
  }, [])

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(e.target as Node)) {
        setSearchResults(null)
        setSearchQuery('')
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const sorted = useMemo(() => {
    const list = [...projects]
    list.sort((a, b) => {
      switch (sortKey) {
        case 'name': return a.name.localeCompare(b.name)
        case 'my_added': return b.my_added - a.my_added
        case 'my_files': return b.my_files - a.my_files
        case 'repo_count': return b.repo_count - a.repo_count
        default: return 0
      }
    })
    return list
  }, [projects, sortKey])

  const todoMap = useMemo(() => {
    const map = new Map<number, number>()
    todoCounts.forEach(c => map.set(c.project_id, c.count))
    return map
  }, [todoCounts])

  const noteMap = useMemo(() => {
    const map = new Map<number, number>()
    noteCounts.forEach(c => map.set(c.project_id, c.count))
    return map
  }, [noteCounts])

  const globalTodoCount = useMemo(() => todoCounts.reduce((sum, c) => sum + c.count, 0), [todoCounts])

  const myAdded = summary?.my_added ?? 0
  const isWorkday = summary?.is_workday ?? false

  return (
    <div className="dashboard">
      <div className="dashboard-fixed">
        <div className="hero-row">
          <div className="hero-card">
            <GoalRing
              value={myAdded}
              goal={isWorkday ? dailyGoal : 0}
              label={isWorkday ? '今日目标' : '非工作日'}
              sublabel={isWorkday ? `${myAdded} / ${dailyGoal} 行` : `${myAdded} 行`}
            />
            <div className="hero-text">
              <div className="hero-eyebrow">{date} · {isWorkday ? '工作日' : '周末'}</div>
              <div className="hero-title">
                {isWorkday
                  ? (myAdded >= dailyGoal ? '今日目标已达成 🎉' : `还差 ${Math.max(dailyGoal - myAdded, 0)} 行达标`)
                  : '周末愉快，无达标要求'}
              </div>
              <div className="hero-sub">
                个人新增 <strong className="green">+{myAdded}</strong> ·
                文件 <strong>{summary?.my_files ?? 0}</strong> ·
                涉及 <strong>{summary?.repo_count ?? 0}</strong> 个仓库
              </div>
            </div>
          </div>

          <SummaryBar summary={summary} globalTodoCount={globalTodoCount} />
        </div>

        <Heatmap onDayClick={setDate} />

        <div className="dashboard-controls">
          <DatePicker value={date} onChange={setDate} />
          <div className="dashboard-actions">
            <div className="search-box" ref={searchRef}>
              <input
                type="text"
                value={searchQuery}
                    onChange={e => handleSearchDebounced(e.target.value)}
                placeholder="搜索笔记与待办…"
                className="form-input search-input"
              />
              {searchResults !== null && (
                <div className="search-dropdown">
                  {searching ? (
                    <div className="search-loading">搜索中...</div>
                  ) : searchResults.length === 0 ? (
                    <div className="search-empty">未找到匹配内容</div>
                  ) : (
                    searchResults.map(h => (
                      <a key={`${h.type}-${h.id}`} href={`/#/project/${h.project_id}`} className="search-result-item">
                        <div className="search-result-header">
                          <span className={`hit-type-mini hit-type-${h.type}`}>{h.type === 'note' ? '笔记' : '待办'}</span>
                          <span className="search-result-project">{h.project_name}</span>
                        </div>
                        <div className="search-result-title">{h.title}</div>
                        <div className="search-result-preview">{h.snippet}</div>
                      </a>
                    ))
                  )}
                </div>
              )}
            </div>
            <div className="filter-toggle">
              <button className={`filter-btn ${!showStarredOnly ? 'active' : ''}`} onClick={() => setShowStarredOnly(false)}>全部</button>
              <button className={`filter-btn ${showStarredOnly ? 'active' : ''}`} onClick={() => setShowStarredOnly(true)}>关注</button>
            </div>
            <div className="sort-control">
              <label>排序：</label>
              <select value={sortKey} onChange={(e) => setSortKey(e.target.value as SortKey)} className="form-input sort-select">
                {SORT_OPTIONS.map(opt => <option key={opt.key} value={opt.key}>{opt.label}</option>)}
              </select>
            </div>
            {confirmScan ? (
              <div className="confirm-group">
                <span className="confirm-text">确定重新扫描？</span>
                <button className="btn btn-primary btn-sm" onClick={handleScan} disabled={scanning}>确认</button>
                <button className="btn btn-sm" onClick={() => setConfirmScan(false)}>取消</button>
              </div>
            ) : (
              <button className="btn btn-primary" onClick={() => setConfirmScan(true)} disabled={scanning}>
                {scanning ? (scanMsg || '处理中...') : '重新扫描'}
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
                  <div className="skeleton skeleton-text" style={{ width: '60%', height: 20 }} />
                </div>
                <div className="card-grid">
                  {Array.from({ length: 4 }).map((_, j) => (
                    <div key={j} className="card-stat">
                      <div className="skeleton skeleton-text" style={{ width: 32, height: 10 }} />
                      <div className="skeleton skeleton-text" style={{ width: 40, height: 16 }} />
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : sorted.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">{showStarredOnly ? '⭐' : '🔍'}</div>
            <h3>{showStarredOnly ? '暂无关注项目' : '暂无项目数据'}</h3>
            <p>
              {showStarredOnly
                ? '你还没有关注任何项目。点击项目卡片右上角的星标即可关注，或切换到「全部」查看所有项目。'
                : 'GitBoard 尚未扫描到任何 Git 仓库。请先配置扫描目录。'}
            </p>
            <div className="empty-actions">
              {showStarredOnly ? (
                <button className="btn btn-primary" onClick={() => setShowStarredOnly(false)}>查看全部项目</button>
              ) : (
                <>
                  <button className="btn btn-primary" onClick={() => setConfirmScan(true)}>开始扫描</button>
                  <a href="/#/settings" className="btn btn-secondary">配置目录</a>
                </>
              )}
            </div>
          </div>
        ) : (
          <div className="project-grid">
            {sorted.map(p => (
              <ProjectCard
                key={p.id}
                project={p}
                date={date}
                todoCount={todoMap.get(p.id)}
                noteCount={noteMap.get(p.id)}
                dailyGoal={isWorkday ? dailyGoal : 0}
                isWorkday={isWorkday}
                onToggleStar={handleToggleStar}
              />
            ))}
          </div>
        )}
      </div>

      <StatusBar />
    </div>
  )
}

export default Dashboard
