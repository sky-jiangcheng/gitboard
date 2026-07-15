import { useState, useEffect, useCallback } from 'react'
import { marked } from 'marked'
import { listNotes, createNote, updateNote, deleteNote, Note } from '../api/client'

interface Props {
  projectId: number
}

function NoteSection({ projectId }: Props) {
  const [notes, setNotes] = useState<Note[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<{ id: number; content: string } | null>(null)
  const [isNew, setIsNew] = useState(false)
  const [newContent, setNewContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [previewId, setPreviewId] = useState<number | null>(null)
  const [filter, setFilter] = useState<'all' | 'knowledge' | 'other'>('all')

  const fetchNotes = useCallback(() => {
    listNotes(projectId).then(setNotes).finally(() => setLoading(false))
  }, [projectId])

  useEffect(() => { fetchNotes() }, [fetchNotes])

  const handleCreate = async () => {
    if (!newContent.trim()) return
    setSaving(true)
    try {
      await createNote(projectId, newContent.trim())
      setNewContent('')
      setIsNew(false)
      fetchNotes()
    } catch { /* ignore */ }
    finally { setSaving(false) }
  }

  const handleUpdate = async () => {
    if (!editing || !editing.content.trim()) return
    setSaving(true)
    try {
      await updateNote(editing.id, editing.content.trim())
      setEditing(null)
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

  const renderMarkdown = (content: string) => {
    return { __html: marked.parse(content) as string }
  }

  // Extract title from note content (first line, stripped of markdown)
  const getNoteTitle = (content: string): string => {
    const firstLine = content.split('\n')[0].replace(/^#+\s*/, '').replace(/^\*\*/, '').replace(/\*\*$/, '').trim()
    return firstLine || '笔记'
  }

  // Check if note looks like a knowledge note (has frontmatter or starts with a heading)
  const isKnowledgeNote = (content: string): boolean => {
    return content.startsWith('---') || content.startsWith('# ') || content.startsWith('## ')
  }

  const filteredNotes = notes.filter(n => {
    if (filter === 'knowledge') return isKnowledgeNote(n.content)
    if (filter === 'other') return !isKnowledgeNote(n.content)
    return true
  })

  if (loading) {
    return (
      <div className="note-section">
        <div className="note-header">
          <h3>知识笔记</h3>
        </div>
        <div className="skeleton skeleton-text" style={{ height: 60, marginBottom: 8 }} />
        <div className="skeleton skeleton-text" style={{ height: 60 }} />
      </div>
    )
  }

  return (
    <div className="note-section">
      <div className="note-header">
        <h3>知识笔记 ({notes.length})</h3>
        {!isNew && !editing && (
          <button className="btn btn-sm btn-primary" onClick={() => setIsNew(true)}>
            新建笔记
          </button>
        )}
      </div>

      {/* Filter tabs */}
      {notes.length > 0 && (
        <div className="note-filters">
          <button
            className={`filter-btn ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
          >
            全部
          </button>
          <button
            className={`filter-btn ${filter === 'knowledge' ? 'active' : ''}`}
            onClick={() => setFilter('knowledge')}
          >
            知识
          </button>
          <button
            className={`filter-btn ${filter === 'other' ? 'active' : ''}`}
            onClick={() => setFilter('other')}
          >
            其他
          </button>
        </div>
      )}

      {/* New note form */}
      {isNew && (
        <div className="note-editor">
          <textarea
            value={newContent}
            onChange={e => setNewContent(e.target.value)}
            placeholder="输入 Markdown 内容...\n支持知识库格式：\n  # 标题\n  ---\n  key: value\n  ---\n  正文内容"
            className="form-input note-textarea"
            rows={8}
          />
          <div className="note-editor-actions">
            <button className="btn btn-primary btn-sm" onClick={handleCreate} disabled={saving || !newContent.trim()}>
              保存
            </button>
            <button className="btn btn-sm" onClick={() => { setIsNew(false); setNewContent('') }}>
              取消
            </button>
          </div>
        </div>
      )}

      {/* Note list */}
      {filteredNotes.length === 0 && !isNew ? (
        <p className="empty-hint">暂无{filter !== 'all' ? '匹配的' : ''}笔记</p>
      ) : (
        <div className="note-list">
          {filteredNotes.map(note => (
            <div key={note.id} className="note-card">
              {editing?.id === note.id ? (
                <div className="note-editor">
                  <textarea
                    value={editing.content}
                    onChange={e => setEditing({ ...editing, content: e.target.value })}
                    className="form-input note-textarea"
                    rows={8}
                  />
                  <div className="note-editor-actions">
                    <button className="btn btn-primary btn-sm" onClick={handleUpdate} disabled={saving || !editing.content.trim()}>
                      保存
                    </button>
                    <button className="btn btn-sm" onClick={() => setEditing(null)}>
                      取消
                    </button>
                    <button className="btn btn-sm" onClick={() => setPreviewId(previewId === note.id ? null : note.id)}>
                      {previewId === note.id ? '编辑' : '预览'}
                    </button>
                  </div>
                  {previewId === note.id && (
                    <div className="note-preview markdown-body" dangerouslySetInnerHTML={renderMarkdown(editing.content)} />
                  )}
                </div>
              ) : (
                <>
                  <div className="note-title-row">
                    <span className="note-title-text">{getNoteTitle(note.content)}</span>
                    {isKnowledgeNote(note.content) && <span className="badge-note-sm">知识</span>}
                  </div>
                  <div className="note-body markdown-body" dangerouslySetInnerHTML={renderMarkdown(note.content)} />
                  <div className="note-meta">
                    <span className="note-time">
                      {note.updated_at !== note.created_at ? '更新于 ' + note.updated_at : '创建于 ' + note.created_at}
                    </span>
                    <div className="note-actions">
                      <button className="btn btn-sm" onClick={() => setEditing({ id: note.id, content: note.content })}>
                        编辑
                      </button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDelete(note.id)}>
                        删除
                      </button>
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