import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  listAllNotes, listAllTags, searchAll, pinNote, importClaudeMemory,
  NoteWithProject, SearchHit,
} from '../api/client'
import { renderMarkdown, stripMarkdown, parseTags } from '../utils/markdown'

type KindFilter = 'all' | 'knowledge' | 'other'

function KnowledgePage() {
  const navigate = useNavigate()
  const [notes, setNotes] = useState<NoteWithProject[]>([])
  const [tags, setTags] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState<SearchHit[] | null>(null)
  const [kindFilter, setKindFilter] = useState<KindFilter>('all')
  const [activeTag, setActiveTag] = useState<string | null>(null)
  const [pinnedOnly, setPinnedOnly] = useState(false)
  const [importing, setImporting] = useState(false)
  const [message, setMessage] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()

  const fetchAll = useCallback(() => {
    Promise.all([listAllNotes(), listAllTags()])
      .then(([n, t]) => { setNotes(n); setTags(t) })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  const handleSearch = (q: string) => {
    setQuery(q)
    const trimmed = q.trim()
    if (!trimmed) { setHits(null); return }
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      searchAll(trimmed).then(setHits).catch(() => setHits([]))
    }, 300)
  }

  const handlePin = async (id: number, pinned: boolean) => {
    setNotes(prev => prev.map(n => n.id === id ? { ...n, pinned: !pinned } : n))
    try { await pinNote(id, !pinned) } catch { setNotes(prev => prev.map(n => n.id === id ? { ...n, pinned } : n)) }
  }

  const handleImport = async () => {
    setImporting(true)
    try {
      const r = await importClaudeMemory()
      setMessage(`导入完成：新增 ${r.synced}，更新 ${r.updated}，跳过 ${r.skipped}`)
      fetchAll()
    } catch (e) {
      setMessage('导入失败：' + (e instanceof Error ? e.message : '未知错误'))
    } finally {
      setImporting(false)
      setTimeout(() => setMessage(''), 4000)
    }
  }

  const filtered = useMemo(() => {
    let list = notes
    if (kindFilter === 'knowledge') list = list.filter(n => n.kind === 'knowledge')
    else if (kindFilter === 'other') list = list.filter(n => n.kind !== 'knowledge')
    if (activeTag) list = list.filter(n => parseTags(n.tags).includes(activeTag))
    if (pinnedOnly) list = list.filter(n => n.pinned)
    return list
  }, [notes, kindFilter, activeTag, pinnedOnly])

  const projectNames = useMemo(() => {
    const set = new Map<string, number>()
    notes.forEach(n => set.set(n.project_name, n.project_id))
    return Array.from(set.entries()).sort((a, b) => a[0].localeCompare(b[0]))
  }, [notes])

  const pinnedCount = useMemo(() => notes.filter(n => n.pinned).length, [notes])

  if (loading) {
    return (
      <div className="knowledge">
        <h1>知识库</h1>
        <div className="skeleton skeleton-text" style={{ width: '100%', height: 48, marginBottom: 12 }} />
        <div className="skeleton skeleton-text" style={{ width: '100%', height: 80 }} />
        <div className="skeleton skeleton-text" style={{ width: '100%', height: 80, marginTop: 12 }} />
      </div>
    )
  }

  return (
    <div className="knowledge">
      <div className="page-head">
        <div>
          <h1>知识库</h1>
          <p className="page-sub">跨项目汇总 {notes.length} 条笔记，支持全文搜索、标签筛选与置顶。</p>
        </div>
        <div className="page-head-actions">
          <button className="btn btn-secondary btn-sm" onClick={handleImport} disabled={importing}>
            {importing ? '导入中…' : '导入 Claude 记忆'}
          </button>
        </div>
      </div>

      {message && <div className="message-banner">{message}</div>}

      <div className="knowledge-search">
        <input
          type="text"
          value={query}
          onChange={e => handleSearch(e.target.value)}
          placeholder="搜索笔记与待办…"
          className="form-input knowledge-search-input"
        />
        {query && <span className="search-hint">回车查看全部，清空返回列表</span>}
      </div>

      {hits !== null ? (
        <div className="knowledge-section">
          <div className="section-header">
            <h2>搜索结果 ({hits.length})</h2>
          </div>
          {hits.length === 0 ? (
            <p className="empty-hint">未找到匹配内容</p>
          ) : (
            <div className="hit-list">
              {hits.map(h => (
                <a
                  key={`${h.type}-${h.id}`}
                  href={`/#/project/${h.project_id}`}
                  className="hit-item"
                >
                  <div className="hit-head">
                    <span className={`hit-type hit-type-${h.type}`}>{h.type === 'note' ? '笔记' : '待办'}</span>
                    <span className="hit-project">{h.project_name}</span>
                  </div>
                  <div className="hit-title">{h.title}</div>
                  <div className="hit-snippet">{h.snippet}</div>
                </a>
              ))}
            </div>
          )}
        </div>
      ) : (
        <>
          <div className="knowledge-filters">
            <div className="filter-toggle">
              <button className={`filter-btn ${kindFilter === 'all' ? 'active' : ''}`} onClick={() => setKindFilter('all')}>全部</button>
              <button className={`filter-btn ${kindFilter === 'knowledge' ? 'active' : ''}`} onClick={() => setKindFilter('knowledge')}>知识</button>
              <button className={`filter-btn ${kindFilter === 'other' ? 'active' : ''}`} onClick={() => setKindFilter('other')}>其他</button>
            </div>
            <button
              className={`filter-btn ${pinnedOnly ? 'active pinned-active' : ''}`}
              onClick={() => setPinnedOnly(v => !v)}
              title="只看置顶"
            >
              ★ 置顶 {pinnedCount}
            </button>
          </div>

          {tags.length > 0 && (
            <div className="tag-chips">
              <button
                className={`tag-chip ${activeTag === null ? 'tag-chip-active' : ''}`}
                onClick={() => setActiveTag(null)}
              >
                全部标签
              </button>
              {tags.map(t => (
                <button
                  key={t}
                  className={`tag-chip ${activeTag === t ? 'tag-chip-active' : ''}`}
                  onClick={() => setActiveTag(activeTag === t ? null : t)}
                >
                  #{t}
                </button>
              ))}
            </div>
          )}

          <div className="knowledge-section">
            <div className="section-header">
              <h2>笔记 ({filtered.length})</h2>
              <select
                className="form-input sort-select"
                value={activeTag ?? ''}
                onChange={() => {}}
                style={{ display: 'none' }}
              >
                <option value="">按更新时间</option>
              </select>
            </div>

            {projectNames.length > 0 && (
              <div className="project-jump">
                <span className="project-jump-label">跳转项目：</span>
                {projectNames.slice(0, 8).map(([name, id]) => (
                  <a key={id} href={`/#/project/${id}`} className="project-jump-item" title={name}>{name}</a>
                ))}
              </div>
            )}

            {filtered.length === 0 ? (
              <div className="empty-state small">
                <div className="empty-icon">📝</div>
                <h3>{notes.length === 0 ? '还没有任何笔记' : '暂无匹配的笔记'}</h3>
                <p>{notes.length === 0
                  ? '前往项目详情页新建笔记，或点击右上角「导入 Claude 记忆」一键同步。'
                  : '尝试调整筛选条件或清除搜索关键词。'}</p>
              </div>
            ) : (
              <div className="note-grid">
                {filtered.map(n => {
                  const nt = parseTags(n.tags)
                  return (
                    <div key={n.id} className={`knowledge-card ${n.pinned ? 'pinned' : ''}`}>
                      <div className="knowledge-card-head">
                        <span className={`kind-badge kind-${n.kind}`}>{n.kind === 'knowledge' ? '知识' : n.kind === 'idea' ? '想法' : n.kind === 'log' ? '日志' : '笔记'}</span>
                        <button
                          className={`pin-btn ${n.pinned ? 'pinned' : ''}`}
                          onClick={() => handlePin(n.id, n.pinned)}
                          title={n.pinned ? '取消置顶' : '置顶'}
                        >
                          ★
                        </button>
                      </div>
                      <a href={`/#/project/${n.project_id}`} className="knowledge-card-body">
                        <div className="knowledge-card-title">{n.title || stripMarkdown(n.content, 40)}</div>
                        <div className="knowledge-card-snippet markdown-body" dangerouslySetInnerHTML={{ __html: renderMarkdown(stripMarkdown(n.content, 120)) }} />
                        <div className="knowledge-card-foot">
                          <span className="knowledge-project-name">{n.project_name}</span>
                          <span className="knowledge-time">{n.updated_at.slice(0, 10)}</span>
                        </div>
                      </a>
                      {nt.length > 0 && (
                        <div className="knowledge-card-tags">
                          {nt.map(t => <span key={t} className="knowledge-tag" onClick={() => setActiveTag(t)}>#{t}</span>)}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}

export default KnowledgePage
