import { useEffect, useState } from 'react'
import { HashRouter, Routes, Route, Link, useLocation } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import ProjectDetail from './pages/ProjectDetail'
import Settings from './pages/Settings'
import Knowledge from './pages/Knowledge'
import CommandPalette from './components/CommandPalette'
import { applyTheme, getStoredTheme, listenSystemTheme } from './utils/theme'

function NavBar({ onOpenPalette }: { onOpenPalette: () => void }) {
  const { pathname } = useLocation()

  const navClass = (active: boolean) => active ? 'active' : ''

  return (
    <nav className="navbar">
      <div className="nav-left">
        <Link to="/" className="nav-brand">
          <span className="nav-brand-mark">▦</span>
          GitBoard
        </Link>
        <div className="nav-links">
          <Link to="/" className={navClass(pathname === '/' || pathname.startsWith('/project'))}>
            仪表盘
          </Link>
          <Link to="/knowledge" className={navClass(pathname === '/knowledge')}>
            知识库
          </Link>
          <Link to="/settings" className={navClass(pathname === '/settings')}>
            设置
          </Link>
        </div>
      </div>
      <button className="nav-palette-btn" onClick={onOpenPalette} title="搜索 (⌘K)">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        <span>搜索</span>
        <kbd className="nav-kbd">⌘K</kbd>
      </button>
    </nav>
  )
}

function App() {
  const [paletteOpen, setPaletteOpen] = useState(false)

  useEffect(() => {
    const mode = getStoredTheme()
    applyTheme(mode)
    return listenSystemTheme(() => {
      if (getStoredTheme() === 'system') applyTheme('system')
    })
  }, [])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setPaletteOpen(o => !o)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  return (
    <HashRouter>
      <div className="app">
        <NavBar onOpenPalette={() => setPaletteOpen(true)} />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/project/:id" element={<ProjectDetail />} />
            <Route path="/knowledge" element={<Knowledge />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
        <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
      </div>
    </HashRouter>
  )
}

export default App
