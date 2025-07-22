import { Link, useLocation } from "react-router-dom";

function NavItem({ 
  to, 
  icon, 
  label, 
  authenticated, 
  onLogin, 
  isMobile = false, 
  onClick,
  requiresAuth = false 
}) {
  const location = useLocation();
  
  const isActive = location.pathname === to;
  
  if (requiresAuth && !authenticated) {
    const handleClick = (e) => {
      e.preventDefault();
      if (onClick) onClick(); 
      onLogin();
    };

    const className = isMobile 
      ? "mobile-nav-link" 
      : `nav-link ${isActive ? "active" : ""}`;

    return (
      <div className={className} onClick={handleClick} style={{ cursor: 'pointer' }}>
        <img src={icon} alt={label} className="nav-icon" />
        {label}
      </div>
    );
  }

  const className = isMobile 
    ? "mobile-nav-link" 
    : `nav-link ${isActive ? "active" : ""}`;

  return (
    <Link
      to={to}
      className={className}
      onClick={onClick}
    >
      <img src={icon} alt={label} className="nav-icon" />
      {label}
    </Link>
  );
}

export default NavItem; 