import { useState } from 'react'
import { Outlet, Link, useLocation } from 'react-router-dom'
import { useTheme } from './contexts/ThemeContext'
import ThemeToggle from './components/ThemeToggle'
import './App.css'

function App() {
  const [toast, setToast] = useState({ visible: false, message: '', type: 'success' });
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const location = useLocation();
  const { isReady } = useTheme();

  // Show toast notification
  const showToast = (message, type = 'success') => {
    setToast({ visible: true, message, type });
    setTimeout(() => {
      setToast({ visible: false, message: '', type: 'success' });
    }, 3000);
  };

  // Navigation items
  const navItems = [
    { path: '/', icon: 'fas fa-home', label: 'Home' },
    { path: '/agents', icon: 'fas fa-users', label: 'Agents' },
    { path: '/actions-playground', icon: 'fas fa-bolt', label: 'Actions' },
    { path: '/skills', icon: 'fas fa-book', label: 'Skills' },
    { path: '/group-create', icon: 'fas fa-users-cog', label: 'Groups' },
  ];

  // Check if route is active
  const isActive = (path) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  // Don't render until theme is ready
  if (!isReady) {
    return null;
  }

  return (
    <div className="app-layout">
      {/* Mobile Overlay */}
      {mobileMenuOpen && (
        <div className="mobile-overlay" onClick={() => setMobileMenuOpen(false)} />
      )}

      {/* Sidebar */}
      <aside className={`sidebar ${mobileMenuOpen ? 'mobile-open' : ''}`}>
        {/* Sidebar Header - Logo Only */}
        <div className="sidebar-header">
          <Link to="/" className="sidebar-logo">
            <img 
              src="/app/logo_1.png" 
              alt="LocalAGI" 
              className="sidebar-logo-img"
            />
          </Link>
        </div>

        {/* Navigation */}
        <nav className="sidebar-nav">
          <ul className="nav-list">
            {navItems.map((item) => (
              <li key={item.path} className="nav-item">
                <Link
                  to={item.path}
                  className={`nav-link ${isActive(item.path) ? 'active' : ''}`}
                  onClick={() => setMobileMenuOpen(false)}
                >
                  <span className="nav-icon">
                    <i className={item.icon} />
                  </span>
                  <span className="nav-label">{item.label}</span>
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        {/* Sidebar Footer */}
        <div className="sidebar-footer">
          <div className="sidebar-status">
            <span className="status-dot" />
            <span className="status-text">
              System <strong>Active</strong>
            </span>
          </div>
          <ThemeToggle />
        </div>
      </aside>

      {/* Main Area */}
      <div className="main-area">
        {/* Mobile Header */}
        <header className="mobile-header">
          <button className="mobile-menu-btn" onClick={() => setMobileMenuOpen(!mobileMenuOpen)}>
            <i className="fas fa-bars" />
          </button>
          <span className="mobile-title">LocalAGI</span>
          <div className="mobile-spacer" />
        </header>

        {/* Main Content */}
        <main className="main-content">
          <div className="content-wrapper">
            <Outlet context={{ showToast }} />
          </div>
        </main>
      </div>

      {/* Toast Notification */}
      {toast.visible && (
        <div className={`toast ${toast.type}`}>
          <i className={`fas ${
            toast.type === 'success' ? 'fa-check-circle' : 
            toast.type === 'error' ? 'fa-exclamation-circle' : 
            'fa-info-circle'
          }`} />
          <span>{toast.message}</span>
        </div>
      )}
    </div>
  )
}

export default App
