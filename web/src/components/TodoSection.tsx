import { useState, useEffect, useCallback } from 'react'
import { listTodos, createTodo, toggleTodo, deleteTodo, reorderTodos, Todo } from '../api/client'

interface Props {
  projectId: number
}

function TodoSection({ projectId }: Props) {
  const [todos, setTodos] = useState<Todo[]>([])
  const [loading, setLoading] = useState(true)
  const [title, setTitle] = useState('')
  const [adding, setAdding] = useState(false)

  const fetchTodos = useCallback(() => {
    listTodos(projectId).then(setTodos).finally(() => setLoading(false))
  }, [projectId])

  useEffect(() => { fetchTodos() }, [fetchTodos])

  const handleAdd = async () => {
    if (!title.trim()) return
    setAdding(true)
    try {
      await createTodo(projectId, title.trim())
      setTitle('')
      fetchTodos()
    } catch { /* ignore */ }
    finally { setAdding(false) }
  }

  const handleToggle = async (todo: Todo) => {
    try {
      await toggleTodo(todo.id)
      setTodos(prev =>
        prev.map(t => t.id === todo.id ? { ...t, completed: !t.completed } : t)
      )
    } catch { /* ignore */ }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteTodo(id)
      setTodos(prev => prev.filter(t => t.id !== id))
    } catch { /* ignore */ }
  }

  const move = async (index: number, direction: number) => {
    const newIndex = index + direction
    if (newIndex < 0 || newIndex >= todos.length) return
    const reordered = [...todos]
    const [item] = reordered.splice(index, 1)
    reordered.splice(newIndex, 0, item)
    setTodos(reordered)
    try {
      await reorderTodos(reordered.map(t => t.id))
    } catch { /* ignore */ }
  }

  if (loading) {
    return (
      <div className="panel-section">
        <h3>待办事项</h3>
        <div className="skeleton skeleton-text" style={{ height: 36, marginBottom: 8 }} />
        <div className="skeleton skeleton-text" style={{ height: 36, marginBottom: 8 }} />
        <div className="skeleton skeleton-text" style={{ height: 36 }} />
      </div>
    )
  }

  return (
    <div className="panel-section">
      <h3>待办事项 ({todos.filter(t => !t.completed).length}/{todos.length})</h3>

      <div className="todo-add">
        <input
          type="text"
          value={title}
          onChange={e => setTitle(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
          placeholder="添加待办..."
          className="form-input"
        />
        <button className="btn btn-primary btn-sm" onClick={handleAdd} disabled={adding || !title.trim()}>
          添加
        </button>
      </div>

      {todos.length === 0 ? (
        <p className="empty-state">暂无待办，点击添加</p>
      ) : (
        <ul className="todo-list">
          {todos.map((todo, i) => (
            <li key={todo.id} className={`todo-item ${todo.completed ? 'completed' : ''}`}>
              <input
                type="checkbox"
                checked={todo.completed}
                onChange={() => handleToggle(todo)}
                className="todo-checkbox"
              />
              <span className="todo-title">{todo.title}</span>
              <div className="todo-actions">
                <button className="btn-icon" onClick={() => move(i, -1)} disabled={i === 0} title="上移">
                  &#x25B2;
                </button>
                <button className="btn-icon" onClick={() => move(i, 1)} disabled={i === todos.length - 1} title="下移">
                  &#x25BC;
                </button>
                <button className="btn-icon btn-delete" onClick={() => handleDelete(todo.id)} title="删除">
                  &#x2715;
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default TodoSection
