import { useState, useEffect, useRef, useMemo } from 'react'
import { getProjects, searchAll, Project, SearchHit } from '../api/client'

interface Props {
  open: boolean
  onClose: () => void
}

// CommandPalette is a Cmd/Ctrl+K quick-switcher: search notes & todos across all
// projects and jump straight to a project. It surfaces the knowledge-search
// capability as a first-class keyboard action.
export default function CommandPalette({ open, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState<SearchHit[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (open) {
      setQuery('')
      setHits([])
      setActiveIndex(0)
      getProjects('', false).then(setProjects).catch(() => setProjects([]))
      setTimeout(() => inputRef.current?.focus(), 30)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const trimmed = query.trim()
    if (!trimmed) { setHits([]); setActiveIndex(0); return }
    let cancelled = false
    const t = setTimeout(() => {
      searchAll(trimmed).then(r => {
        if (!cancelled) { setHits(r ?? []); setActiveIndex(0) }
      }).catch(() => { if (!cancelled) setHits([]) })
    }, 150)
    return () => { cancelled = true; clearTimeout(t) }
  }, [query, open])

  const filteredProjects = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return projects.slice(0, 5)
    return projects.filter(p => p.name.toLowerCase().includes(q)).slice(0, 5)
  }, [projects, query])

  const totalItems = hits.length + filteredProjects.length

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActiveIndex(i => Math.min(i + 1, totalItems - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setActiveIndex(i => Math.max(i - 1, 0)) }
    else if (e.key === 'Enter') {
      e.preventDefault()
      if (activeIndex < hits.length) {
        const h = hits[activeIndex]
        if (h) { window.location.hash = `#/project/${h.project_id}`; onClose() }
      } else {
        const p = filteredProjects[activeIndex - hits.length]
        if (p) { window.location.hash = `#/project/${p.id}`; onClose() }
      }
    } else if (e.key === 'Escape') { e.preventDefault(); onClose() }
  }

  if (!open) return null

  return (
    <div className="cmdk-overlay" onClick={onClose}>
      <div className="cmdk" onClick={e => e.stopPropagation()}>
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="搜索笔记 / 待办 / 跳转项目…"
          className="cmdk-input"
        />
        <div className="cmdk-results">
          {hits.length === 0 && filteredProjects.length === 0 && (
            <div className="cmdk-empty">{query.trim() ? '未找到结果' : '输入关键词开始搜索'}</div>
          )}
          {hits.length > 0 && <div className="cmdk-group">笔记与待办</div>}
          {hits.map((h, i) => (
            <a
              key={`${h.type}-${h.id}`}
              href={`/#/project/${h.project_id}`}
              className={`cmdk-item ${activeIndex === i ? 'cmdk-item-active' : ''}`}
              onMouseEnter={() => setActiveIndex(i)}
              onClick={onClose}
            >
              <span className={`cmdk-type cmdk-type-${h.type}`}>{h.type === 'note' ? '笔' : '办'}</span>
              <div className="cmdk-item-body">
                <div className="cmdk-item-title">{h.title}</div>
                <div className="cmdk-item-sub">{h.project_name} · {h.snippet.slice(0, 60)}</div>
              </div>
            </a>
          ))}
          {filteredProjects.length > 0 && <div className="cmdk-group">项目</div>}
          {filteredProjects.map((p, i) => (
            <a
              key={p.id}
              href={`/#/project/${p.id}`}
              className={`cmdk-item ${activeIndex === hits.length + i ? 'cmdk-item-active' : ''}`}
              onMouseEnter={() => setActiveIndex(hits.length + i)}
              onClick={onClose}
            >
              <span className="cmdk-type cmdk-type-project">项</span>
              <div className="cmdk-item-body">
                <div className="cmdk-item-title">{p.name}</div>
                <div className="cmdk-item-sub">{p.repo_count} 个仓库</div>
              </div>
            </a>
          ))}
        </div>
        <div className="cmdk-foot">
          <span>↑↓ 选择</span><span>↵ 打开</span><span>esc 关闭</span>
        </div>
      </div>
    </div>
  )
}
