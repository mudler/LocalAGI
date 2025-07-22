import { useState, useEffect } from "react";
import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import "./App.css";
import { usePrivy, useLogin, useLogout } from "@privy-io/react-auth";
import NavItem from "./components/NavItem";

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

  const { ready, authenticated } = usePrivy();
  const { login } = useLogin();
  const { logout } = useLogout({
    onSuccess: () => {
      showToast("Logged out successfully.", "success");
    }
  });

  const isAuthLoading = !ready;
  const isAuthenticated = ready && authenticated;

  const navItems = [
    {
      to: "/",
      icon: "/app/nav/house.svg",
      label: "Home",
      requiresAuth: false
    },
    {
      to: "/agents",
      icon: "/app/nav/robot.svg", 
      label: "Agent List",
      requiresAuth: true
    },
    {
      to: "/actions-playground",
      icon: "/app/nav/bolt.svg",
      label: "Action Playground", 
      requiresAuth: true
    },
    {
      to: "/group-create",
      icon: "/app/nav/user-group.svg",
      label: "Create Group Agent",
      requiresAuth: true
    },
    {
      to: "/usage", 
      icon: "/app/nav/chart.svg",
      label: "Usage",
      requiresAuth: true
    }
  ];

  if (isAuthLoading) {
    return <div></div>;
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
            {navItems.map((item) => (
              <NavItem
                key={item.to}
                to={item.to}
                icon={item.icon}
                label={item.label}
                authenticated={authenticated}
                onLogin={login}
                requiresAuth={item.requiresAuth}
                isMobile={false}
              />
            ))}
          </div>

          <div className="user-actions">
            {authenticated ? (
              <button 
                onClick={logout}
                className="logout-btn"
                title="Logout"
              >
                Logout
              </button>
            ) : (
              <button 
                onClick={login}
                className="login-btn"
                title="Login"
              >
                Log in
              </button>
            )}
          </div>
          <div className="nav-right-container mobile-only">
           {
            !authenticated && (
              <button 
                onClick={login}
                className="login-btn"
                title="Login"
              >
                Log in
              </button>
            )
           }
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
            {navItems.map((item) => (
              <li key={item.to}>
                <NavItem
                  to={item.to}
                  icon={item.icon}
                  label={item.label}
                  authenticated={authenticated}
                  onLogin={login}
                  requiresAuth={item.requiresAuth}
                  isMobile={true}
                  onClick={() => setMobileMenuOpen(false)}
                />
              </li>
            ))}
            <li>
              {authenticated ? (
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
              ) : (
                null
              )}
            </li>
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
