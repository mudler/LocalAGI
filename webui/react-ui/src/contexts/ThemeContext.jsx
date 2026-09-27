import { createContext, useContext, useState, useEffect } from 'react';

const ThemeContext = createContext();

export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState('dark');
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    // Check for saved preference or system preference
    const savedTheme = localStorage.getItem('localagi-theme');
    if (savedTheme) {
      setTheme(savedTheme);
    } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
      setTheme('light');
    }
    setIsReady(true);
  }, []);

  useEffect(() => {
    if (isReady) {
      // Apply theme to document
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem('localagi-theme', theme);
    }
  }, [theme, isReady]);

  const toggleTheme = () => {
    setTheme(prev => prev === 'dark' ? 'light' : 'dark');
  };

  const setDarkMode = () => setTheme('dark');
  const setLightMode = () => setTheme('light');

  const value = {
    theme,
    isDark: theme === 'dark',
    isLight: theme === 'light',
    toggleTheme,
    setDarkMode,
    setLightMode,
    isReady
  };

  // Don't render until theme is determined to prevent flash
  if (!isReady) {
    return null;
  }

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}
