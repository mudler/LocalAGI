import { useState, useEffect } from "react";
import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import "./App.css";
import { usePrivy } from "@privy-io/react-auth";

function App() {
  const [toast, setToast] = useState({
    visible: false,
    message: "",
    type: "success",
  });
  const [toastQueue, setToastQueue] = useState([]);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();

  // Show toast notification (queue support, can show same toast multiple times)
  const showToast = (message, type = "success", duration = 3_000) => {
    // If no toast is currently visible, show immediately
    if (!toast.visible) {
      setToast({ visible: true, message, type });
      // Auto-hide after duration
      setTimeout(() => {
        setToast({ visible: false, message: "", type: "success" });
      }, duration);
    } else {
      // Add to queue if a toast is already showing
      setToastQueue((prevQueue) => [...prevQueue, { message, type, duration }]);
    }
  };

  // Toast display logic: show next toast in queue when current one is hidden
  useEffect(() => {
    if (!toast.visible && toastQueue.length > 0) {
      const { message, type, duration } = toastQueue[0];
      setToast({ visible: true, message, type });
      
      // Auto-hide after duration and remove from queue
      const timer = setTimeout(() => {
        setToast({ visible: false, message: "", type: "success" });
        setToastQueue((prevQueue) => prevQueue.slice(1));
      }, duration);
      
      return () => clearTimeout(timer);
    }
  }, [toast.visible, toastQueue]);

  // Toggle mobile menu
  const toggleMobileMenu = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  // Check if a path is active
  const isActive = (path) => {
    return location.pathname === path;
  };

  const { ready, authenticated, logout } = usePrivy();

  const isAuthLoading = !ready;

  const isAuthenticated = ready && authenticated;

  // Redirect to /app if authenticated and not already on /app
  useEffect(() => {
    if (!isAuthLoading && !isAuthenticated && location.pathname !== '/app') {
      navigate('/');
    }
  }, [isAuthenticated, location.pathname, navigate, isAuthLoading]);

  if (isAuthLoading) {
    return <div></div>;
  }

  

  if (!isAuthenticated) {
    return (
      <main className="main-content">
        <Outlet context={{ showToast }} />
      </main>
    );
  }


  return (
    <div className="app-container">
      {/* Navigation Menu */}
      <nav className="main-nav">
        <div className="container">
          <div className="logo-container">
            {/* Logo */}
            <Link to="/" className="logo-link">
              <div className="logo-image-container">
                <img src="/app/logo_2.png" alt="Logo" className="logo-image" />
              </div>
            </Link>
          </div>

          <div className="nav-links">
            <Link
              to="/"
              className={`nav-link ${isActive("/") ? "active" : ""}`}
            >
              <img src="/app/nav/house.svg" alt="House" className="nav-icon" />
              Home
            </Link>
            <>
              <Link
                to="/agents"
                className={`nav-link ${isActive("/agents") ? "active" : ""}`}
              >
                <img
                  src="/app/nav/robot.svg"
                  alt="Robot"
                  className="nav-icon"
                />{" "}
                Agent List
              </Link>
              <Link
                to="/actions-playground"
                className={`nav-link ${
                  isActive("/actions-playground") ? "active" : ""
                }`}
              >
                <img src="/app/nav/bolt.svg" alt="Bolt" className="nav-icon" />
                Action Playground
              </Link>
              <Link
                to="/group-create"
                className={`nav-link ${
                  isActive("/group-create") ? "active" : ""
                }`}
              >
                <img
                  src="/app/nav/user-group.svg"
                  alt="User Group"
                  className="nav-icon"
                />
                Create Group Agent
              </Link>
              <Link
                to="/usage"
                className={`nav-link ${isActive("/usage") ? "active" : ""}`}
              >
                <img
                  src="/app/nav/chart.svg"
                  alt="Chart"
                  className="nav-icon"
                />
                Usage
              </Link>
            </>
          </div>

          <div className="user-actions">
            <button 
              onClick={logout}
              className="logout-btn"
              title="Logout"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16,17 21,12 16,7"/>
                <line x1="21" y1="12" x2="9" y2="12"/>
              </svg>
              Logout
            </button>
          </div>

          <div className="mobile-menu-toggle" onClick={toggleMobileMenu}>
            <i className="fas fa-bars"></i>
          </div>
        </div>
      </nav>

      {/* Mobile Menu */}
      {mobileMenuOpen && (
        <div className="mobile-menu">
          <ul className="mobile-nav-links">
            <li>
              <Link
                to="/"
                className="mobile-nav-link"
                onClick={() => setMobileMenuOpen(false)}
              >
                <img
                  src="/app/nav/house.svg"
                  alt="House"
                  className="nav-icon"
                />{" "}
                Home
              </Link>
            </li>
            <>
              <li>
                <Link
                  to="/agents"
                  className="mobile-nav-link"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  <img
                    src="/app/nav/robot.svg"
                    alt="Robot"
                    className="nav-icon"
                  />{" "}
                  Agent List
                </Link>
              </li>
              <li>
                <Link
                  to="/actions-playground"
                  className="mobile-nav-link"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  <img
                    src="/app/nav/bolt.svg"
                    alt="Bolt"
                    className="nav-icon"
                  />{" "}
                  Action Playground
                </Link>
              </li>
              <li>
                <Link
                  to="/group-create"
                  className="mobile-nav-link"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  <img
                    src="/app/nav/user-group.svg"
                    alt="User Group"
                    className="nav-icon"
                  />{" "}
                  Create Group Agent
                </Link>
              </li>
              <li>
                <Link
                  to="/usage"
                  className="mobile-nav-link"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  <img
                    src="/app/nav/chart.svg"
                    alt="Chart"
                    className="nav-icon"
                  />{" "}
                  Usage
                </Link>
              </li>
              <li>
                <button 
                  onClick={() => {
                    setMobileMenuOpen(false);
                    logout();
                  }}
                  className="mobile-nav-link logout-mobile"
                >
                  <svg className="nav-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                    <polyline points="16,17 21,12 16,7"/>
                    <line x1="21" y1="12" x2="9" y2="12"/>
                  </svg>
                  Logout
                </button>
              </li>
            </>
          </ul>
        </div>
      )}

      {/* Toast Notification */}
      {toast.visible && (
        <div className={`toast ${toast.type}`}>
          <span>{toast.message}</span>
          <button
            className="toast-close"
            onClick={() => setToast({ ...toast, visible: false })}
            aria-label="Close notification"
          >
            ×
          </button>
        </div>
      )}

      {/* Main Content Area */}
      <main className="main-content">
        <Outlet context={{ showToast }} />
      </main>
    </div>
  );
}

export default App;
