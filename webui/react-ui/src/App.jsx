import { useState } from 'react'
import { Outlet, Link } from 'react-router-dom'
import './App.css'

function App() {
  const [toast, setToast] = useState({ visible: false, message: '', type: 'success' });
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  // Show toast notification
  const showToast = (message, type = 'success') => {
    setToast({ visible: true, message, type });
    setTimeout(() => {
      setToast({ visible: false, message: '', type: 'success' });
    }, 3000);
  };

  // Toggle mobile menu
  const toggleMobileMenu = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  return (
    <div className="app-container">
      {/* Navigation Menu */}
      <nav className="main-nav">
        <div className="container">
          <div className="nav-content">
            <div className="logo-container">
              {/* Logo */}
              <Link to="/" className="logo-link">
                <div className="logo-image-container">
                  <img src="/app/logo_2.png" alt="Logo" className="logo-image" />
                </div>
                {/* <span className="logo-text">LocalAGI</span> */}
              </Link>
            </div>

            <div className="desktop-menu">
              <ul className="nav-links">
                <li>
                  <Link to="/" className="nav-link">
                    <i className="fas fa-home mr-2"></i> Home
                  </Link>
                </li>
                <li>
                  <Link to="/agents" className="nav-link">
                    <i className="fas fa-users mr-2"></i> Agent List
                  </Link>
                </li>
                <li>
                  <Link to="/actions-playground" className="nav-link">
                    <i className="fas fa-bolt mr-2"></i> Actions Playground
                  </Link>
                </li>
                <li>
                  <Link to="/group-create" className="nav-link">
                    <i className="fas fa-users-cog mr-2"></i> Create Agent Group
                  </Link>
                </li>
              </ul>
            </div>

            <div className="">
              <span className="status-indicator"></span>
              <span className="status-text">State: <span className="status-value">active</span></span>
            </div>

            <div className="mobile-menu-toggle" onClick={toggleMobileMenu}>
              <i className="fas fa-bars"></i>
            </div>
          </div>
        </div>
      </nav>

      {/* Mobile Menu */}
      {mobileMenuOpen && (
        <div className="mobile-menu">
          <ul className="mobile-nav-links">
            <li>
              <Link to="/" className="mobile-nav-link" onClick={() => setMobileMenuOpen(false)}>
                <i className="fas fa-home mr-2"></i> Home
              </Link>
            </li>
            <li>
              <Link to="/agents" className="mobile-nav-link" onClick={() => setMobileMenuOpen(false)}>
                <i className="fas fa-users mr-2"></i> Agent List
              </Link>
            </li>
            <li>
              <Link to="/actions-playground" className="mobile-nav-link" onClick={() => setMobileMenuOpen(false)}>
                <i className="fas fa-bolt mr-2"></i> Actions Playground
              </Link>
            </li>
            <li>
              <Link to="/group-create" className="mobile-nav-link" onClick={() => setMobileMenuOpen(false)}>
                <i className="fas fa-users-cog mr-2"></i> Create Agent Group
              </Link>
            </li>
          </ul>
        </div>
      )}

      {/* Toast Notification */}
      {toast.visible && (
        <div className={`toast ${toast.type}`}>
          <span>{toast.message}</span>
        </div>
      )}

      {/* Main Content Area */}
      <main className="main-content">
        <div className="container">
          <Outlet context={{ showToast }} />
        </div>
      </main>

    </div>
  )
}

export default App
