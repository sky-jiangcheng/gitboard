import { useState, useEffect, useCallback } from 'react'
import {
  listNotes, createNoteWithMeta, updateNote, updateNoteMeta, deleteNote, pinNote, Note,
} from '../api/client'
import { renderMarkdown } from '../utils/markdown'

interface Props {
  projectId: number
}

type KindFilter = 'all' | 'knowledge' | 'other'

const KINDS = [
  { value: 'knowledge', label: '知识' },
  { value: 'log', label: '日志' },
  { value: 'idea', label: '想法' },
  { value: 'other', label: '其他' },
]

function draftKey(projectId: number) {
  return `gitboard-note-draft-${projectId}`
}

function loadDraft(projectId: number): { content: string; title: string; tags: string; kind: string } {
  try {
    const raw = localStorage.getItem(draftKey(projectId))
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  return { content: '', title: '', tags: '', kind: 'knowledge' }
}

function NoteSection({ projectId }: Props) {
  const [notes, setNotes] = useState<Note[]>([])
  const [loading, setLoading] = useState(true)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editContent, setEditContent] = useState('')
  const [editTitle, setEditTitle] = useState('')
  const [editTags, setEditTags] = useState('')
  const [editKind, setEditKind] = useState('knowledge')
  const [editPinned, setEditPinned] = useState(false)
  const [showPreview, setShowPreview] = useState(true)
  const [isNew, setIsNew] = useState(false)
  const [saving, setSaving] = useState(false)
  const [filter, setFilter] = useState<KindFilter>('all')

  const [draft, setDraft] = useState(() => loadDraft(projectId))

  const fetchNotes = useCallback(() => {
    listNotes(projectId).then(setNotes).finally(() => setLoading(false))
  }, [projectId])

  useEffect(() => { fetchNotes() }, [fetchNotes])

  useEffect(() => {
    try { localStorage.setItem(draftKey(projectId), JSON.stringify(draft)) } catch { /* ignore */ }
  }, [draft, projectId])

  const startNew = () => {
    setIsNew(true)
    setEditingId(null)
  }

  const handleCreate = async () => {
    if (!draft.content.trim()) return
    setSaving(true)
    try {
      await createNoteWithMeta(projectId, draft.content.trim(), {
        title: draft.title.trim(),
        tags: draft.tags.trim(),
        kind: draft.kind,
      })
      setDraft({ content: '', title: '', tags: '', kind: 'knowledge' })
      try { localStorage.removeItem(draftKey(projectId)) } catch { /* ignore */ }
      setIsNew(false)
      fetchNotes()
    } catch { /* ignore */ }
    finally { setSaving(false) }
  }

  const startEdit = (note: Note) => {
    setEditingId(note.id)
    setEditContent(note.content)
    setEditTitle(note.title)
    setEditTags(note.tags)
    setEditKind(note.kind || 'other')
    setEditPinned(note.pinned)
    setIsNew(false)
  }

  const handleSaveEdit = async () => {
    if (editingId === null || !editContent.trim()) return
    setSaving(true)
    try {
      await updateNote(editingId, editContent.trim())
      await updateNoteMeta(editingId, editTitle.trim(), editTags.trim(), editKind, editPinned)
      setEditingId(null)
      fetchNotes()
    } catch { /* ignore */ }
    finally { setSaving(false) }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteNote(id)
      setNotes(prev => prev.filter(n => n.id !== id))
    } catch { /* ignore */ }
  }

  const handlePin = async (note: Note) => {
    setNotes(prev => prev.map(n => n.id === note.id ? { ...n, pinned: !note.pinned } : n))
    try { await pinNote(note.id, !note.pinned) } catch { setNotes(prev => prev.map(n => n.id === note.id ? { ...n, pinned: note.pinned } : n)) }
  }

  const filteredNotes = notes.filter(n => {
    if (filter === 'knowledge') return n.kind === 'knowledge'
    if (filter === 'other') return n.kind !== 'knowledge'
    return true
  })

  if (loading) {
    return (
      <div className="panel-section">
        <h3>知识笔记</h3>
        <div className="skeleton skeleton-text" style={{ height: 60, marginBottom: 8 }} />
        <div className="skeleton skeleton-text" style={{ height: 60 }} />
      </div>
    )
  }

  return (
    <div className="note-section">
      <div className="note-header">
        <h3>知识笔记 ({notes.length})</h3>
        {!isNew && editingId === null && (
          <button className="btn btn-sm btn-primary" onClick={startNew}>新建笔记</button>
        )}
      </div>

      {notes.length > 0 && (
        <div className="note-filters">
          <button className={`filter-btn ${filter === 'all' ? 'active' : ''}`} onClick={() => setFilter('all')}>全部</button>
          <button className={`filter-btn ${filter === 'knowledge' ? 'active' : ''}`} onClick={() => setFilter('knowledge')}>知识</button>
          <button className={`filter-btn ${filter === 'other' ? 'active' : ''}`} onClick={() => setFilter('other')}>其他</button>
        </div>
      )}

      {/* New-note editor */}
      {isNew && (
        <div className="note-editor-block">
          <div className="note-meta-row">
            <input
              type="text"
              value={draft.title}
              onChange={e => setDraft({ ...draft, title: e.target.value })}
              placeholder="标题（留空将取首行）"
              className="form-input note-title-input"
            />
            <select
              value={draft.kind}
              onChange={e => setDraft({ ...draft, kind: e.target.value })}
              className="form-input note-kind-select"
            >
              {KINDS.map(k => <option key={k.value} value={k.value}>{k.label}</option>)}
            </select>
          </div>
          <input
            type="text"
            value={draft.tags}
            onChange={e => setDraft({ ...draft, tags: e.target.value })}
            placeholder="标签（逗号分隔，如：架构, 待办）"
            className="form-input note-tags-input"
          />
          <div className="note-editor-split">
            <textarea
              value={draft.content}
              onChange={e => setDraft({ ...draft, content: e.target.value })}
              placeholder="输入 Markdown 内容…"
              className="form-input note-textarea"
              rows={10}
            />
            {showPreview && (
              <div className="note-preview markdown-body" dangerouslySetInnerHTML={{ __html: renderMarkdown(draft.content) }} />
            )}
          </div>
          <div className="note-editor-actions">
            <button className="btn btn-primary btn-sm" onClick={handleCreate} disabled={saving || !draft.content.trim()}>保存</button>
            <button className="btn btn-sm" onClick={() => setShowPreview(v => !v)}>{showPreview ? '隐藏预览' : '显示预览'}</button>
            <button className="btn btn-sm" onClick={() => { setIsNew(false); setDraft({ content: '', title: '', tags: '', kind: 'knowledge' }) }}>取消</button>
            {draft.content && <span className="draft-hint">草稿已自动保存</span>}
          </div>
        </div>
      )}

      {/* Note list */}
      {filteredNotes.length === 0 && !isNew ? (
        <p className="empty-hint">{filter !== 'all' ? '暂无匹配的笔记' : '暂无笔记，点击上方按钮新建'}</p>
      ) : (
        <div className="note-list">
          {filteredNotes.map(note => (
            <div key={note.id} className={`note-card ${note.pinned ? 'pinned' : ''}`}>
              {editingId === note.id ? (
                <div className="note-editor-block">
                  <div className="note-meta-row">
                    <input
                      type="text"
                      value={editTitle}
                      onChange={e => setEditTitle(e.target.value)}
                      placeholder="标题"
                      className="form-input note-title-input"
                    />
                    <select
                      value={editKind}
                      onChange={e => setEditKind(e.target.value)}
                      className="form-input note-kind-select"
                    >
                      {KINDS.map(k => <option key={k.value} value={k.value}>{k.label}</option>)}
                    </select>
                    <label className="note-pin-toggle">
                      <input type="checkbox" checked={editPinned} onChange={e => setEditPinned(e.target.checked)} />
                      置顶
                    </label>
                  </div>
                  <input
                    type="text"
                    value={editTags}
                    onChange={e => setEditTags(e.target.value)}
                    placeholder="标签（逗号分隔）"
                    className="form-input note-tags-input"
                  />
                  <div className="note-editor-split">
                    <textarea
                      value={editContent}
                      onChange={e => setEditContent(e.target.value)}
                      className="form-input note-textarea"
                      rows={10}
                    />
                    {showPreview && (
                      <div className="note-preview markdown-body" dangerouslySetInnerHTML={{ __html: renderMarkdown(editContent) }} />
                    )}
                  </div>
                  <div className="note-editor-actions">
                    <button className="btn btn-primary btn-sm" onClick={handleSaveEdit} disabled={saving || !editContent.trim()}>保存</button>
                    <button className="btn btn-sm" onClick={() => setShowPreview(v => !v)}>{showPreview ? '隐藏预览' : '显示预览'}</button>
                    <button className="btn btn-sm" onClick={() => setEditingId(null)}>取消</button>
                  </div>
                </div>
              ) : (
                <>
                  <div className="note-title-row">
                    <span className="note-title-text">{note.title || note.content.split('\n')[0] || '笔记'}</span>
                    <div className="note-title-badges">
                      {note.kind === 'knowledge' && <span className="badge-note-sm">知识</span>}
                      {note.source === 'claude' && <span className="badge-note-sm badge-source">Claude</span>}
                      <button className={`pin-btn ${note.pinned ? 'pinned' : ''}`} onClick={() => handlePin(note)} title={note.pinned ? '取消置顶' : '置顶'}>★</button>
                    </div>
                  </div>
                  <div className="note-body markdown-body" dangerouslySetInnerHTML={{ __html: renderMarkdown(note.content) }} />
                  {note.tags && (
                    <div className="note-tags-row">
                      {note.tags.split(',').map(t => t.trim()).filter(Boolean).map(t => <span key={t} className="note-tag-chip">#{t}</span>)}
                    </div>
                  )}
                  <div className="note-meta">
                    <span className="note-time">
                      {note.updated_at !== note.created_at ? '更新于 ' + note.updated_at : '创建于 ' + note.created_at}
                    </span>
                    <div className="note-actions">
                      <button className="btn btn-sm" onClick={() => startEdit(note)}>编辑</button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDelete(note.id)}>删除</button>
                    </div>
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default NoteSection
