import { useState } from 'react';
import { Outlet, Link, useLocation } from 'react-router-dom';

const Sidebar = ({ children }) => {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const location = useLocation();

  const navItems = [
    { path: '/', icon: 'fas fa-home', label: 'Home' },
    { path: '/agents', icon: 'fas fa-users', label: 'Agents' },
    { path: '/actions-playground', icon: 'fas fa-bolt', label: 'Actions' },
    { path: '/group-create', icon: 'fas fa-users-cog', label: 'Groups' },
  ];

  const isActive = (path) => {
    if (path === '/') return location.pathname === '/';
    return location.pathname.startsWith(path);
  };

  const toggleMobile = () => setMobileOpen(!mobileOpen);
  const closeMobile = () => setMobileOpen(false);

  return (
    <div className={`app-layout ${collapsed ? 'sidebar-collapsed' : ''}`}>
      {/* Mobile Overlay */}
      {mobileOpen && <div className="sidebar-overlay" onClick={closeMobile} />}

      {/* Sidebar */}
      <aside className={`sidebar ${mobileOpen ? 'mobile-open' : ''}`}>
        {/* Logo */}
        <div className="sidebar-header">
          <Link to="/" className="sidebar-logo" onClick={closeMobile}>
            <div className="logo-icon">
              <img src="/app/logo_1.png" alt="LocalAGI" />
            </div>
            {!collapsed && <span className="logo-text">LocalAGI</span>}
          </Link>
          <button 
            className="sidebar-toggle desktop-only" 
            onClick={() => setCollapsed(!collapsed)}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <i className={`fas fa-chevron-${collapsed ? 'right' : 'left'}`} />
          </button>
        </div>

        {/* Navigation */}
        <nav className="sidebar-nav">
          <ul className="nav-list">
            {navItems.map((item) => (
              <li key={item.path} className="nav-item">
                <Link
                  to={item.path}
                  className={`nav-link ${isActive(item.path) ? 'active' : ''}`}
                  onClick={closeMobile}
                  title={collapsed ? item.label : ''}
                >
                  <i className={item.icon} />
                  {!collapsed && <span className="nav-label">{item.label}</span>}
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        {/* Status Footer */}
        <div className="sidebar-footer">
          <div className="status-indicator-wrapper">
            <span className="status-dot" />
            {!collapsed && (
              <span className="status-label">
                System <strong>Active</strong>
              </span>
            )}
          </div>
        </div>
      </aside>

      {/* Main Area */}
      <div className="main-area">
        {/* Mobile Header */}
        <header className="mobile-header">
          <button className="mobile-menu-btn" onClick={toggleMobile}>
            <i className="fas fa-bars" />
          </button>
          <span className="mobile-title">LocalAGI</span>
        </header>

        {/* Page Content */}
        <main className="content-area">
          {children}
        </main>
      </div>
    </div>
  );
};

export default Sidebar;
