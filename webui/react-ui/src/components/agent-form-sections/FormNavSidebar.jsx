import React from 'react';

/**
 * Navigation sidebar for the agent form
 */
const FormNavSidebar = ({ activeSection, handleSectionChange }) => {
  // Define the navigation items
  const navItems = [
    { id: 'basic-section', icon: 'fas fa-info-circle', label: 'Basic Information' },
    { id: 'connectors-section', icon: 'fas fa-plug', label: 'Connectors' },
    { id: 'actions-section', icon: 'fas fa-bolt', label: 'Actions' },
    { id: 'mcp-section', icon: 'fas fa-server', label: 'MCP Servers' },
    { id: 'memory-section', icon: 'fas fa-memory', label: 'Memory Settings' },
    { id: 'model-section', icon: 'fas fa-robot', label: 'Model Settings' },
    { id: 'prompts-section', icon: 'fas fa-comment-alt', label: 'Prompts & Goals' },
    { id: 'advanced-section', icon: 'fas fa-cogs', label: 'Advanced Settings' }
  ];

  return (
    <div className="wizard-sidebar">
      <ul className="wizard-nav">
        {navItems.map(item => (
          <li 
            key={item.id}
            className={`wizard-nav-item ${activeSection === item.id ? 'active' : ''}`} 
            onClick={() => handleSectionChange(item.id)}
          >
            <i className={item.icon}></i> {item.label}
          </li>
        ))}
      </ul>
    </div>
  );
};

export default FormNavSidebar;
