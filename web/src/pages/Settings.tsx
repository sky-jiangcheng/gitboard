import { useState, useEffect, useRef } from 'react'
import { getConfig, updateConfig, updateScanRoots, triggerScan } from '../api/client'

interface ConfigData {
  config: Record<string, string>
  scan_roots: string[]
}

type TabKey = 'scan' | 'standards' | 'authors' | 'actions'

const TABS: { key: TabKey; label: string }[] = [
  { key: 'scan', label: '扫描目录' },
  { key: 'standards', label: '代码标准' },
  { key: 'authors', label: '作者配置' },
  { key: 'actions', label: '操作' },
]

function Settings() {
  const [data, setData] = useState<ConfigData | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [newRoot, setNewRoot] = useState('')
  const [codeStandard, setCodeStandard] = useState('500')
  const [scanDepth, setScanDepth] = useState('5')
  const [authorName, setAuthorName] = useState('')
  const [message, setMessage] = useState('')
  const [tab, setTab] = useState<TabKey>('scan')
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  const showMessage = (msg: string) => {
    setMessage(msg)
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setMessage(''), 3000)
  }

  useEffect(() => {
    getConfig()
      .then((d) => {
        setData(d)
        setCodeStandard(d.config.daily_code_standard || '500')
        setScanDepth(d.config.scan_depth || '5')
        setAuthorName(d.config.git_author || '')
      })
      .finally(() => setLoading(false))
    return () => { if (timerRef.current) clearTimeout(timerRef.current) }
  }, [])

  const handleSaveConfig = async () => {
    const num = parseInt(codeStandard, 10)
    if (isNaN(num) || num < 100 || num > 10000) {
      showMessage('每日目标行数应在 100-10000 之间')
      return
    }
    const depth = parseInt(scanDepth, 10)
    if (isNaN(depth) || depth < 1 || depth > 10) {
      showMessage('扫描深度应在 1-10 之间')
      return
    }
    setSaving(true)
    try {
      await updateConfig('daily_code_standard', String(num))
      await updateConfig('scan_depth', String(depth))
      showMessage('配置已保存')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '保存失败'
      showMessage('保存失败: ' + msg)
    } finally {
      setSaving(false)
    }
  }

  const handleSaveAuthor = async () => {
    const trimmed = authorName.trim()
    if (!trimmed) {
      showMessage('请输入 Git 作者名称')
      return
    }
    setSaving(true)
    try {
      await updateConfig('git_author', trimmed)
      showMessage('作者配置已保存')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '保存失败'
      showMessage('保存失败: ' + msg)
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
      showMessage('扫描目录已添加')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '添加失败'
      showMessage('添加失败: ' + msg)
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
      showMessage('扫描目录已移除')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '移除失败'
      showMessage('移除失败: ' + msg)
    } finally {
      setSaving(false)
    }
  }

  const handleRescan = async () => {
    setSaving(true)
    try {
      await triggerScan()
      showMessage('重新扫描完成')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '扫描失败'
      showMessage('扫描失败: ' + msg)
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="settings">
        <h1>设置</h1>
        <div className="skeleton skeleton-text" style={{width: '100%', height: 24, marginBottom: 12}} />
        <div className="skeleton skeleton-text" style={{width: '100%', height: 64, marginBottom: 8}} />
        <div className="skeleton skeleton-text" style={{width: '100%', height: 64, marginBottom: 8}} />
      </div>
    )
  }

  return (
    <div className="settings">
      <h1>设置</h1>

      {message && <div className="message-banner">{message}</div>}

      <div className="settings-tabs">
        {TABS.map((t) => (
          <button
            key={t.key}
            className={`tab-btn ${tab === t.key ? 'tab-active' : ''}`}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'scan' && (
        <div className="settings-section">
          <h2>扫描根目录</h2>
          <div className="form-group">
            <label>添加新目录：</label>
            <div className="input-row">
              <input
                type="text"
                value={newRoot}
                onChange={(e) => setNewRoot(e.target.value)}
                placeholder="/Users/you/Projects"
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
                <span className="root-path">{root}</span>
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
      )}

      {tab === 'standards' && (
        <div className="settings-section">
          <h2>代码量标准</h2>
          <div className="form-group">
            <label>工作日每日目标行数：</label>
            <input
              type="number"
              value={codeStandard}
              onChange={(e) => setCodeStandard(e.target.value)}
              className="form-input"
              min={100}
              max={10000}
            />
            <span className="form-hint">范围: 100-10000</span>
          </div>
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
            <span className="form-hint">范围: 1-10</span>
          </div>
          <button className="btn btn-primary" onClick={handleSaveConfig} disabled={saving}>
            保存配置
          </button>
        </div>
      )}

      {tab === 'authors' && (
        <div className="settings-section">
          <h2>Git 作者配置</h2>
          <p className="section-desc">配置用于统计个人代码量的 Git 作者名称。</p>
          <div className="form-group">
            <label>作者名称：</label>
            <input
              type="text"
              value={authorName}
              onChange={(e) => setAuthorName(e.target.value)}
              placeholder="John Doe"
              className="form-input"
            />
            <span className="form-hint">与 git log --author 过滤的名称一致</span>
          </div>
          <button className="btn btn-primary" onClick={handleSaveAuthor} disabled={saving}>
            保存作者
          </button>
        </div>
      )}

      {tab === 'actions' && (
        <div className="settings-section">
          <h2>操作</h2>
          <p className="section-desc">手动触发全量重新扫描，刷新所有仓库的统计数据。</p>
          <button className="btn btn-primary" onClick={handleRescan} disabled={saving}>
            立即重新扫描所有项目
          </button>
        </div>
      )}
    </div>
  )
}

export default Settings
