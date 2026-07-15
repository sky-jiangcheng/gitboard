// api/client.ts — Dual-mode API client:
//   - When running inside Wails, calls Go methods via window.go
//   - When running standalone (npm run dev), falls back to HTTP fetch

// --- Type definitions (shared between Wails & HTTP) ---

export interface Project {
  id: number
  name: string
  root_path: string
  level_override: number
  is_auto_grouped: boolean
  is_starred: boolean
  created_at: string
  repo_count: number
  total_added: number
  total_deleted: number
  my_added: number
  my_deleted: number
  my_files: number
  is_workday: boolean
  below_standard: boolean
}

export interface ProjectDetail {
  id: number
  name: string
  root_path: string
  level_override: number
  is_auto_grouped: boolean
  repos: RepoInfo[]
}

export interface RepoInfo {
  id: number
  path: string
  project_id: number
  last_scanned_at: string
  stats: DailyStat[]
}

export interface DailyStat {
  id: number
  repository_id: number
  stat_date: string
  author: string
  files_changed: number
  lines_added: number
  lines_deleted: number
}

export interface Summary {
  date: string
  repo_count: number
  total_files: number
  total_added: number
  total_deleted: number
  my_added: number
  my_deleted: number
  my_files: number
  is_workday: boolean
}

export interface AppConfig {
  config: Record<string, string>
  scan_roots: string[]
}

export interface Todo {
  id: number
  project_id: number
  title: string
  completed: boolean
  priority: number
  sort_order: number
  created_at: string
  updated_at: string
}

export interface Note {
  id: number
  project_id: number
  content: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface TodoCount {
  project_id: number
  count: number
  total: number
}

export interface NoteCount {
  project_id: number
  count: number
}

export interface SearchNotesResult {
  note_id: number
  content: string
  project_id: number
  project_name: string
  updated_at: string
}

export interface HeatmapDay {
  date: string
  lines_added: number
  lines_deleted: number
  commits: number
}

export interface HeatmapResponse {
  days: HeatmapDay[]
}

export interface StatusBarData {
  current_time: string
  last_commit_time: string
  last_commit_repo: string
  last_commit_branch: string
  last_commit_msg: string
}

// --- Wails mode helper ---

const isWails = (): boolean =>
  typeof window !== 'undefined' && !!(window as any).go?.main?.App

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function wail<T>(method: string, ...args: any[]): Promise<T> {
  const app = (window as any).go.main.App
  return app[method](...args) as Promise<T>
}

// --- HTTP mode helper (standalone dev) ---

const BASE = '/api'

async function http<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || 'Request failed')
  }
  return res.json()
}

// --- Public API ---

export function getProjects(date?: string, starredOnly = false): Promise<Project[]> {
  if (isWails()) return wail<Project[]>('GetProjects', date || '', starredOnly).then(d => d ?? [])
  const params = new URLSearchParams()
  if (date) params.set('date', date)
  if (starredOnly) params.set('starred', '1')
  return http<Project[]>(`/projects?${params.toString()}`).then(data => data ?? [])
}

export function getProjectDetail(id: number): Promise<ProjectDetail> {
  if (isWails()) return wail<ProjectDetail>('GetProjectDetail', id)
  return http<ProjectDetail>(`/projects/${id}`)
}

export function getProjectStats(id: number, date?: string): Promise<DailyStat[]> {
  if (isWails()) return wail<DailyStat[]>('GetProjectStats', id, date || '').then(d => d ?? [])
  const params = date ? `?date=${date}` : ''
  return http<DailyStat[]>(`/projects/${id}/stats${params}`).then(data => data ?? [])
}

export function toggleStar(id: number): Promise<boolean> {
  if (isWails()) return wail<boolean>('ToggleStar', id)
  return http<{ starred: boolean }>(`/projects/${id}/star`, { method: 'POST' }).then(r => r.starred)
}

export function updateProjectLevel(
  id: number,
  direction: 'up' | 'down'
): Promise<{ success: boolean; new_level: number }> {
  if (isWails()) return wail('UpdateProjectLevel', id, direction)
  return http(`/projects/${id}/level`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ direction }),
  })
}

export function triggerScan(): Promise<{ success: boolean }> {
  if (isWails()) return wail('TriggerScan')
  return http('/scan', { method: 'POST' })
}

export interface ScanStatus {
  running: boolean
  message: string
  progress: number
  total: number
}

export function getScanStatus(): Promise<ScanStatus> {
  if (isWails()) return wail<ScanStatus>('GetScanStatus').then(d => d ?? { running: false, message: '', progress: 0, total: 0 })
  return http<ScanStatus>('/scan/status').then(d => d ?? { running: false, message: '', progress: 0, total: 0 })
}

export function getConfig(): Promise<AppConfig> {
  if (isWails()) return wail<AppConfig>('GetConfig')
  return http<AppConfig>('/config')
}

export function updateConfig(key: string, value: string): Promise<{ success: boolean }> {
  if (isWails()) {
    return wail<{ success: boolean }>('UpdateConfig', key, value).catch(() => ({ success: false }))
  }
  return http('/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ key, value }),
  })
}

export function updateScanRoots(scan_roots: string[]): Promise<{ success: boolean }> {
  if (isWails()) {
    return wail<{ success: boolean }>('UpdateScanRoots', scan_roots).catch(() => ({ success: false }))
  }
  return http('/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scan_roots }),
  })
}

export function getSummary(date?: string): Promise<Summary> {
  const empty: Summary = {
    date: '', repo_count: 0, total_files: 0, total_added: 0,
    total_deleted: 0, my_added: 0, my_deleted: 0, my_files: 0, is_workday: false,
  }
  if (isWails()) return wail<Summary>('GetSummary', date || '').then(d => d ?? empty)
  const params = date ? `?date=${date}` : ''
  return http<Summary>(`/summary${params}`).then(data => data ?? empty)
}

// --- Todo & Note API ---

export function listTodos(projectId: number): Promise<Todo[]> {
  if (isWails()) return wail<Todo[]>('ListTodos', projectId).then(d => d ?? [])
  return http<Todo[]>(`/todos?project_id=${projectId}`).then(d => d ?? [])
}

export function createTodo(projectId: number, title: string): Promise<Todo> {
  if (isWails()) return wail<Todo>('CreateTodo', projectId, title)
  return http<Todo>('/todos', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ project_id: projectId, title }),
  })
}

export function toggleTodo(todoId: number): Promise<void> {
  if (isWails()) return wail<void>('ToggleTodo', todoId)
  return http<void>(`/todos/${todoId}/toggle`, { method: 'POST' })
}

export function deleteTodo(todoId: number): Promise<void> {
  if (isWails()) return wail<void>('DeleteTodo', todoId)
  return http<void>(`/todos/${todoId}`, { method: 'DELETE' })
}

export function reorderTodos(todoIds: number[]): Promise<void> {
  if (isWails()) return wail<void>('ReorderTodos', todoIds)
  return http<void>('/todos/reorder', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ todo_ids: todoIds }),
  })
}

export function listNotes(projectId: number): Promise<Note[]> {
  if (isWails()) return wail<Note[]>('ListNotes', projectId).then(d => d ?? [])
  return http<Note[]>(`/notes?project_id=${projectId}`).then(d => d ?? [])
}

export function createNote(projectId: number, content: string): Promise<Note> {
  if (isWails()) return wail<Note>('CreateNote', projectId, content)
  return http<Note>('/notes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ project_id: projectId, content }),
  })
}

export function updateNote(noteId: number, content: string): Promise<void> {
  if (isWails()) return wail<void>('UpdateNote', noteId, content)
  return http<void>(`/notes/${noteId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
}

export function deleteNote(noteId: number): Promise<void> {
  if (isWails()) return wail<void>('DeleteNote', noteId)
  return http<void>(`/notes/${noteId}`, { method: 'DELETE' })
}

export function getTodoCounts(): Promise<TodoCount[]> {
  if (isWails()) return wail<TodoCount[]>('GetTodoCounts').then(d => d ?? [])
  return http<TodoCount[]>('/todo-counts').then(d => d ?? [])
}

export function getHeatmapData(): Promise<HeatmapResponse> {
  if (isWails()) return wail<HeatmapResponse>('GetHeatmapData').then(d => d ?? { days: [] })
  return http<HeatmapResponse>('/heatmap').then(d => d ?? { days: [] })
}

export function getNoteCounts(): Promise<NoteCount[]> {
  if (isWails()) return wail<NoteCount[]>('GetNoteCounts').then(d => d ?? [])
  return http<NoteCount[]>('/note-counts').then(d => d ?? [])
}

export function searchNotes(query: string): Promise<SearchNotesResult[]> {
  if (isWails()) return wail<SearchNotesResult[]>('SearchNotes', query).then(d => d ?? [])
  return http<SearchNotesResult[]>(`/notes/search?q=${encodeURIComponent(query)}`).then(d => d ?? [])
}

export function getStatusBar(): Promise<StatusBarData> {
  if (isWails()) return wail<StatusBarData>('GetStatusBar').then(d => d ?? {
    current_time: '', last_commit_time: '', last_commit_repo: '',
    last_commit_branch: '', last_commit_msg: '',
  })
  return http<StatusBarData>('/status-bar').then(d => d ?? {
    current_time: '', last_commit_time: '', last_commit_repo: '',
    last_commit_branch: '', last_commit_msg: '',
  })
}
