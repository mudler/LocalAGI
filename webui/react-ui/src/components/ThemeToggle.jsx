import { useTheme } from '../contexts/ThemeContext';

function ThemeToggle() {
  const { theme, toggleTheme, isDark } = useTheme();

  return (
    <button
      className="theme-toggle"
      onClick={toggleTheme}
      title={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      aria-label={`Current theme: ${theme}. Click to toggle.`}
    >
      <div className="theme-toggle-track">
        <div className={`theme-toggle-thumb ${isDark ? 'dark' : 'light'}`}>
          <i className={`fas ${isDark ? 'fa-moon' : 'fa-sun'}`} />
        </div>
      </div>
      <span className="theme-toggle-label">
        {isDark ? 'Dark' : 'Light'}
      </span>
    </button>
  );
}

export default ThemeToggle;
