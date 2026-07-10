import { HashRouter, Routes, Route, Link, useLocation } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import ProjectDetail from './pages/ProjectDetail'
import Settings from './pages/Settings'

function NavBar() {
  const { pathname } = useLocation()

  return (
    <nav className="navbar">
      <Link to="/" className="nav-brand">GitBoard</Link>
      <div className="nav-links">
        <Link to="/" className={pathname === '/' || pathname.startsWith('/project') ? 'active' : ''}>
          仪表盘
        </Link>
        <Link to="/settings" className={pathname === '/settings' ? 'active' : ''}>
          设置
        </Link>
      </div>
    </nav>
  )
}

function App() {
  return (
    <HashRouter>
      <div className="app">
        <NavBar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/project/:id" element={<ProjectDetail />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </HashRouter>
  )
}

export default App
