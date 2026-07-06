import { useState, useEffect } from 'react'
import { getConfig, updateConfig, updateScanRoots, triggerScan } from '../api/client'

interface ConfigData {
  config: Record<string, string>
  scan_roots: string[]
}

function Settings() {
  const [data, setData] = useState<ConfigData | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [newRoot, setNewRoot] = useState('')
  const [codeStandard, setCodeStandard] = useState('500')
  const [scanDepth, setScanDepth] = useState('5')
  const [message, setMessage] = useState('')

  useEffect(() => {
    getConfig()
      .then((d) => {
        setData(d)
        setCodeStandard(d.config.daily_code_standard || '500')
        setScanDepth(d.config.scan_depth || '5')
      })
      .finally(() => setLoading(false))
  }, [])

  const handleSaveConfig = async () => {
    setSaving(true)
    setMessage('')
    try {
      await updateConfig('daily_code_standard', codeStandard)
      await updateConfig('scan_depth', scanDepth)
      setMessage('配置已保存')
    } catch (e: any) {
      setMessage('保存失败: ' + e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleAddRoot = async () => {
    if (!newRoot.trim() || !data) return
    setSaving(true)
    try {
      const updated = [...data.scan_roots, newRoot.trim()]
      await updateScanRoots(updated)
      setData({ ...data, scan_roots: updated })
      setNewRoot('')
      setMessage('扫描目录已添加')
    } catch (e: any) {
      setMessage('添加失败: ' + e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleRemoveRoot = async (path: string) => {
    if (!data) return
    setSaving(true)
    try {
      const updated = data.scan_roots.filter((r) => r !== path)
      await updateScanRoots(updated)
      setData({ ...data, scan_roots: updated })
      setMessage('扫描目录已移除')
    } catch (e: any) {
      setMessage('移除失败: ' + e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleRescan = async () => {
    setSaving(true)
    try {
      await triggerScan()
      setMessage('重新扫描完成')
    } catch (e: any) {
      setMessage('扫描失败: ' + e.message)
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="loading">加载中...</div>

  return (
    <div className="settings">
      <h1>设置</h1>

      {message && <div className="message-banner">{message}</div>}

      <div className="settings-section">
        <h2>代码量标准</h2>
        <div className="form-group">
          <label>工作日每日目标行数：</label>
          <input
            type="number"
            value={codeStandard}
            onChange={(e) => setCodeStandard(e.target.value)}
            className="form-input"
          />
        </div>
      </div>

      <div className="settings-section">
        <h2>扫描配置</h2>
        <div className="form-group">
          <label>最大扫描深度：</label>
          <input
            type="number"
            value={scanDepth}
            onChange={(e) => setScanDepth(e.target.value)}
            className="form-input"
            min={1}
            max={10}
          />
        </div>
        <button className="btn btn-primary" onClick={handleSaveConfig} disabled={saving}>
          保存配置
        </button>
      </div>

      <div className="settings-section">
        <h2>扫描根目录</h2>
        <div className="form-group">
          <label>添加新目录：</label>
          <div className="input-row">
            <input
              type="text"
              value={newRoot}
              onChange={(e) => setNewRoot(e.target.value)}
              placeholder="/path/to/projects"
              className="form-input"
            />
            <button className="btn btn-primary" onClick={handleAddRoot} disabled={saving}>
              添加
            </button>
          </div>
        </div>
        <ul className="root-list">
          {data?.scan_roots.map((root) => (
            <li key={root} className="root-item">
              <span>{root}</span>
              <button
                className="btn btn-danger btn-sm"
                onClick={() => handleRemoveRoot(root)}
                disabled={saving}
              >
                移除
              </button>
            </li>
          ))}
          {(!data?.scan_roots || data.scan_roots.length === 0) && (
            <li className="root-item empty">暂无扫描目录，请添加</li>
          )}
        </ul>
      </div>

      <div className="settings-section">
        <h2>操作</h2>
        <button className="btn btn-primary" onClick={handleRescan} disabled={saving}>
          立即重新扫描所有项目
        </button>
      </div>
    </div>
  )
}

export default Settings
