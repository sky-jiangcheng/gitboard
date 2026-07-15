import { useState, useEffect, useRef, useMemo } from 'react'
import { getProjects, getSummary, triggerScan, getTodoCounts, getNoteCounts, searchNotes, getScanStatus, toggleStar, Project, Summary, TodoCount, NoteCount, SearchNotesResult } from '../api/client'
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
  const [noteCounts, setNoteCounts] = useState<NoteCount[]>([])
  const [showStarredOnly, setShowStarredOnly] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<SearchNotesResult[] | null>(null)
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
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
    } finally {
      setLoading(false)
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
      const msg = e instanceof Error ? e.message : '操作失败'
      setError(msg)
    }
  }

  useEffect(() => {
    fetchData(date)
    checkScanStatus()
    return () => {
      if (pollTimer.current) clearInterval(pollTimer.current)
    }
  }, [date])

  useEffect(() => {
    fetchData(date, showStarredOnly)
  }, [showStarredOnly])

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

  const handleSearch = async (query: string) => {
    setSearchQuery(query)
    if (!query.trim()) {
      setSearchResults(null)
      return
    }
    setSearching(true)
    try {
      const results = await searchNotes(query)
      setSearchResults(results)
    } catch {
      setSearchResults([])
    } finally {
      setSearching(false)
    }
  }

  // Close search on click outside
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

  const noteMap = useMemo(() => {
    const map = new Map<number, number>()
    noteCounts.forEach(c => map.set(c.project_id, c.count))
    return map
  }, [noteCounts])

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
            <div className="search-box" ref={searchRef}>
              <input
                type="text"
                value={searchQuery}
                onChange={e => handleSearch(e.target.value)}
                placeholder="搜索知识笔记..."
                className="form-input search-input"
              />
              {searchResults !== null && (
                <div className="search-dropdown">
                  {searching ? (
                    <div className="search-loading">搜索中...</div>
                  ) : searchResults.length === 0 ? (
                    <div className="search-empty">未找到匹配的笔记</div>
                  ) : (
                    searchResults.map(r => (
                      <a
                        key={r.note_id}
                        href={`/#/project/${r.project_id}`}
                        className="search-result-item"
                      >
                        <div className="search-result-header">
                          <span className="search-result-project">{r.project_name}</span>
                        </div>
                        <div className="search-result-preview">
                          {r.content.length > 120 ? r.content.slice(0, 120) + '...' : r.content}
                        </div>
                      </a>
                    ))
                  )}
                </div>
              )}
            </div>
            <div className="filter-toggle">
              <button
                className={`filter-btn ${!showStarredOnly ? 'active' : ''}`}
                onClick={() => setShowStarredOnly(false)}
              >
                全部
              </button>
              <button
                className={`filter-btn ${showStarredOnly ? 'active' : ''}`}
                onClick={() => setShowStarredOnly(true)}
              >
                关注
              </button>
            </div>
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
            <div className="empty-icon">{showStarredOnly ? '&#11088;' : '&#128269;'}</div>
            <h3>{showStarredOnly ? '暂无关注项目' : '暂无项目数据'}</h3>
            <p>
              {showStarredOnly
                ? '你还没有关注任何项目。点击项目卡片右上角的星标即可关注，或切换到「全部」查看所有项目。'
                : 'GitBoard 尚未扫描到任何 Git 仓库。请先配置扫描目录。'}
            </p>
            <div className="empty-actions">
              {showStarredOnly ? (
                <button className="btn btn-primary" onClick={() => setShowStarredOnly(false)}>
                  查看全部项目
                </button>
              ) : (
                <>
                  <button className="btn btn-primary" onClick={() => setConfirmScan(true)}>
                    开始扫描
                  </button>
                  <a href="/#/settings" className="btn btn-secondary">
                    配置目录
                  </a>
                </>
              )}
            </div>
          </div>
        ) : (
          <div className="project-grid">
            {sorted.map((p) => (
              <ProjectCard key={p.id} project={p} date={date} todoCount={todoMap.get(p.id)} noteCount={noteMap.get(p.id)} onToggleStar={handleToggleStar} />
            ))}
          </div>
        )}
      </div>

      <StatusBar />
    </div>
  )
}

export default Dashboard
