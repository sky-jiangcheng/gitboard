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

  if (loading) {
    return (
      <div className="panel-section">
        <h3>笔记</h3>
        <div className="skeleton skeleton-text" style={{ height: 60, marginBottom: 8 }} />
        <div className="skeleton skeleton-text" style={{ height: 60 }} />
      </div>
    )
  }

  return (
    <div className="panel-section">
      <div className="note-header">
        <h3>笔记 ({notes.length})</h3>
        {!isNew && !editing && (
          <button className="btn btn-sm btn-primary" onClick={() => setIsNew(true)}>
            新建笔记
          </button>
        )}
      </div>

      {/* New note form */}
      {isNew && (
        <div className="note-editor">
          <textarea
            value={newContent}
            onChange={e => setNewContent(e.target.value)}
            placeholder="输入 Markdown 内容..."
            className="form-input note-textarea"
            rows={6}
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
      {notes.length === 0 && !isNew ? (
        <p className="empty-hint">暂无笔记</p>
      ) : (
        <div className="note-list">
          {notes.map(note => (
            <div key={note.id} className="note-card">
              {editing?.id === note.id ? (
                <div className="note-editor">
                  <textarea
                    value={editing.content}
                    onChange={e => setEditing({ ...editing, content: e.target.value })}
                    className="form-input note-textarea"
                    rows={5}
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
