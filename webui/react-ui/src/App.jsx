import { useState, useEffect } from "react";
import { Outlet, Link, useLocation } from "react-router-dom";
import "./App.css";

function App() {
  const [toast, setToast] = useState({
    visible: false,
    message: "",
    type: "success",
  });
  const [toastQueue, setToastQueue] = useState([]);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const location = useLocation();

  // Show toast notification (queue support, can show same toast multiple times)
  const showToast = (message, type = "success", duration = 10_000) => {
    setToastQueue((prevQueue) => [...prevQueue, { message, type, duration }]);
  };

  // Toast display logic: show next toast in queue
  useEffect(() => {
    if (!toast.visible && toastQueue.length > 0) {
      const { message, type, duration } = toastQueue[0];
      setToast({ visible: true, message, type });
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
            <Link
              to="/agents"
              className={`nav-link ${isActive("/agents") ? "active" : ""}`}
            >
              <img src="/app/nav/robot.svg" alt="Robot" className="nav-icon" />{" "}
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
          </div>

          <div className="status-text">
            <span className="status-indicator"></span>
            Active
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
                <img src="/app/nav/bolt.svg" alt="Bolt" className="nav-icon" />{" "}
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
