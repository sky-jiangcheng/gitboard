const BASE = '/api'

export interface Project {
  id: number
  name: string
  root_path: string
  level_override: number
  is_auto_grouped: boolean
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

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || 'Request failed')
  }
  return res.json()
}

export function getProjects(date?: string): Promise<Project[]> {
  const params = date ? `?date=${date}` : ''
  return request<Project[]>(`/projects${params}`)
}

export function getProjectDetail(id: number): Promise<ProjectDetail> {
  return request<ProjectDetail>(`/projects/${id}`)
}

export function getProjectStats(id: number, date?: string): Promise<DailyStat[]> {
  const params = date ? `?date=${date}` : ''
  return request<DailyStat[]>(`/projects/${id}/stats${params}`)
}

export function updateProjectLevel(id: number, direction: 'up' | 'down'): Promise<{ success: boolean; new_level: number }> {
  return request(`/projects/${id}/level`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ direction }),
  })
}

export function triggerScan(): Promise<{ success: boolean; repos_found: number; projects: number }> {
  return request('/scan', { method: 'POST' })
}

export function getConfig(): Promise<AppConfig> {
  return request<AppConfig>('/config')
}

export function updateConfig(key: string, value: string): Promise<{ success: boolean }> {
  return request('/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ key, value }),
  })
}

export function updateScanRoots(scan_roots: string[]): Promise<{ success: boolean }> {
  return request('/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scan_roots }),
  })
}

export function getSummary(date?: string): Promise<Summary> {
  const params = date ? `?date=${date}` : ''
  return request<Summary>(`/summary${params}`)
}
