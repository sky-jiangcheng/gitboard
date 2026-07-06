import { useState, useEffect } from 'react'
import { getProjects, getSummary, triggerScan, Project, Summary } from '../api/client'
import SummaryBar from '../components/SummaryBar'
import DatePicker from '../components/DatePicker'
import ProjectCard from '../components/ProjectCard'

function getYesterday(): string {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return d.toISOString().split('T')[0]
}

function Dashboard() {
  const [projects, setProjects] = useState<Project[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [date, setDate] = useState(getYesterday())
  const [loading, setLoading] = useState(true)
  const [scanning, setScanning] = useState(false)
  const [error, setError] = useState('')

  const fetchData = async (selectedDate: string) => {
    setLoading(true)
    setError('')
    try {
      const [projData, sumData] = await Promise.all([
        getProjects(selectedDate),
        getSummary(selectedDate),
      ])
      setProjects(projData)
      setSummary(sumData)
    } catch (e: any) {
      setError(e.message || 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData(date)
  }, [date])

  const handleScan = async () => {
    setScanning(true)
    try {
      await triggerScan()
      await fetchData(date)
    } catch (e: any) {
      setError(e.message || 'Scan failed')
    } finally {
      setScanning(false)
    }
  }

  const handleDateChange = (newDate: string) => {
    setDate(newDate)
  }

  return (
    <div className="dashboard">
      <SummaryBar summary={summary} />

      <div className="dashboard-controls">
        <DatePicker value={date} onChange={handleDateChange} />
        <button className="btn btn-primary" onClick={handleScan} disabled={scanning}>
          {scanning ? '扫描中...' : '重新扫描'}
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {loading ? (
        <div className="loading">加载中...</div>
      ) : projects.length === 0 ? (
        <div className="empty-state">
          <p>暂无项目数据</p>
          <p className="hint">点击"重新扫描"或前往设置页面配置扫描目录</p>
        </div>
      ) : (
        <div className="project-grid">
          {projects.map((p) => (
            <ProjectCard key={p.id} project={p} />
          ))}
        </div>
      )}
    </div>
  )
}

export default Dashboard
