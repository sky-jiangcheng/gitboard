import { useEffect, useState } from 'react'
import { getStatusBar, type StatusBarData } from '../api/client'

export default function StatusBar() {
  const [data, setData] = useState<StatusBarData | null>(null)

  const fetch = () => {
    getStatusBar().then(setData).catch(() => {})
  }

  useEffect(() => {
    fetch()
    const timer = setInterval(fetch, 30000)
    return () => clearInterval(timer)
  }, [])

  // Update current time every second locally
  const [now, setNow] = useState(new Date())
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(t)
  }, [])

  const currentTime = now.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })

  return (
    <div className="status-bar">
      <div className="status-left">
        <span className="status-item">
          <span className="status-dot" />
          当前时间：{currentTime}
        </span>
      </div>
      <div className="status-right">
        {data?.last_commit_time ? (
          <>
            <span className="status-item" title={data.last_commit_msg}>
              最近提交：<strong>{data.last_commit_time}</strong>
            </span>
            <span className="status-separator">|</span>
            <span className="status-item">
              项目：<strong>{data.last_commit_repo}</strong>
            </span>
            <span className="status-separator">|</span>
            <span className="status-item">
              分支：<strong>{data.last_commit_branch || 'unknown'}</strong>
            </span>
          </>
        ) : (
          <span className="status-item muted">暂无提交记录</span>
        )}
      </div>
    </div>
  )
}
