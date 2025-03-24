import React, { useState } from 'react';
import { useNavigate, useOutletContext } from 'react-router-dom';

// Import form sections
import BasicInfoSection from './agent-form-sections/BasicInfoSection';
import ConnectorsSection from './agent-form-sections/ConnectorsSection';
import ActionsSection from './agent-form-sections/ActionsSection';
import MCPServersSection from './agent-form-sections/MCPServersSection';
import MemorySettingsSection from './agent-form-sections/MemorySettingsSection';
import ModelSettingsSection from './agent-form-sections/ModelSettingsSection';
import PromptsGoalsSection from './agent-form-sections/PromptsGoalsSection';
import AdvancedSettingsSection from './agent-form-sections/AdvancedSettingsSection';

const AgentForm = ({ 
  isEdit = false, 
  formData, 
  setFormData, 
  onSubmit, 
  loading = false, 
  submitButtonText, 
  isGroupForm = false,
  noFormWrapper = false
}) => {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [activeSection, setActiveSection] = useState(isGroupForm ? 'model-section' : 'basic-section');

  // Handle input changes
  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    
    if (name.includes('.')) {
      const [parent, child] = name.split('.');
      setFormData({
        ...formData,
        [parent]: {
          ...formData[parent],
          [child]: type === 'checkbox' ? checked : value
        }
      });
    } else {
      setFormData({
        ...formData,
        [name]: type === 'checkbox' ? checked : value
      });
    }
  };

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    if (onSubmit) {
      onSubmit(e);
    }
  };

  // Handle navigation between sections
  const handleSectionChange = (section) => {
    setActiveSection(section);
  };

  // Handle adding a connector
  const handleAddConnector = () => {
    setFormData({
      ...formData,
      connectors: [
        ...(formData.connectors || []),
        { name: '', config: '{}' }
      ]
    });
  };

  // Handle removing a connector
  const handleRemoveConnector = (index) => {
    const updatedConnectors = [...formData.connectors];
    updatedConnectors.splice(index, 1);
    setFormData({
      ...formData,
      connectors: updatedConnectors
    });
  };

  // Handle connector name change
  const handleConnectorNameChange = (index, value) => {
    const updatedConnectors = [...formData.connectors];
    updatedConnectors[index] = {
      ...updatedConnectors[index],
      type: value
    };
    setFormData({
      ...formData,
      connectors: updatedConnectors
    });
  };

  // Handle connector config change
  const handleConnectorConfigChange = (index, key, value) => {
    const updatedConnectors = [...formData.connectors];
    const currentConnector = updatedConnectors[index];
    
    // Parse the current config if it's a string
    let currentConfig = {};
    if (typeof currentConnector.config === 'string') {
      try {
        currentConfig = JSON.parse(currentConnector.config);
      } catch (err) {
        console.error('Error parsing config:', err);
        currentConfig = {};
      }
    } else if (currentConnector.config) {
      currentConfig = currentConnector.config;
    }
    
    // Update the config with the new key-value pair
    currentConfig = {
      ...currentConfig,
      [key]: value
    };
    
    // Update the connector with the stringified config
    updatedConnectors[index] = {
      ...currentConnector,
      config: JSON.stringify(currentConfig)
    };
    
    setFormData({
      ...formData,
      connectors: updatedConnectors
    });
  };

  // Handle adding an MCP server
  const handleAddMCPServer = () => {
    setFormData({
      ...formData,
      mcp_servers: [
        ...(formData.mcp_servers || []),
        { url: '' }
      ]
    });
  };

  // Handle removing an MCP server
  const handleRemoveMCPServer = (index) => {
    const updatedMCPServers = [...formData.mcp_servers];
    updatedMCPServers.splice(index, 1);
    setFormData({
      ...formData,
      mcp_servers: updatedMCPServers
    });
  };

  // Handle MCP server change
  const handleMCPServerChange = (index, value) => {
    const updatedMCPServers = [...formData.mcp_servers];
    updatedMCPServers[index] = { url: value };
    setFormData({
      ...formData,
      mcp_servers: updatedMCPServers
    });
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  return (
    <div className="agent-form-container">
      {/* Wizard Sidebar */}
      <div className="wizard-sidebar">
        <ul className="wizard-nav">
          {!isGroupForm && (
            <li 
              className={`wizard-nav-item ${activeSection === 'basic-section' ? 'active' : ''}`} 
              onClick={() => handleSectionChange('basic-section')}
            >
              <i className="fas fa-info-circle"></i>
              Basic Information
            </li>
          )}
          <li 
            className={`wizard-nav-item ${activeSection === 'model-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('model-section')}
          >
            <i className="fas fa-brain"></i>
            Model Settings
          </li>
          <li 
            className={`wizard-nav-item ${activeSection === 'connectors-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('connectors-section')}
          >
            <i className="fas fa-plug"></i>
            Connectors
          </li>
          <li 
            className={`wizard-nav-item ${activeSection === 'actions-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('actions-section')}
          >
            <i className="fas fa-bolt"></i>
            Actions
          </li>
          <li 
            className={`wizard-nav-item ${activeSection === 'mcp-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('mcp-section')}
          >
            <i className="fas fa-server"></i>
            MCP Servers
          </li>
          <li 
            className={`wizard-nav-item ${activeSection === 'memory-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('memory-section')}
          >
            <i className="fas fa-memory"></i>
            Memory Settings
          </li>
          <li 
            className={`wizard-nav-item ${activeSection === 'prompts-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('prompts-section')}
          >
            <i className="fas fa-comment-alt"></i>
            Prompts & Goals
          </li>
          <li 
            className={`wizard-nav-item ${activeSection === 'advanced-section' ? 'active' : ''}`} 
            onClick={() => handleSectionChange('advanced-section')}
          >
            <i className="fas fa-cogs"></i>
            Advanced Settings
          </li>
        </ul>
      </div>

      {/* Form Content */}
      <div className="form-content-area">
        {noFormWrapper ? (
          <div className='agent-form'>
            {/* Form Sections */}
            <div style={{ display: activeSection === 'basic-section' ? 'block' : 'none' }}>
              <BasicInfoSection formData={formData} handleInputChange={handleInputChange} isEdit={isEdit} isGroupForm={isGroupForm} />
            </div>

            <div style={{ display: activeSection === 'model-section' ? 'block' : 'none' }}>
              <ModelSettingsSection formData={formData} handleInputChange={handleInputChange} />
            </div>

            <div style={{ display: activeSection === 'connectors-section' ? 'block' : 'none' }}>
              <ConnectorsSection formData={formData} handleAddConnector={handleAddConnector} handleRemoveConnector={handleRemoveConnector} handleConnectorNameChange={handleConnectorNameChange} handleConnectorConfigChange={handleConnectorConfigChange} />
            </div>

            <div style={{ display: activeSection === 'actions-section' ? 'block' : 'none' }}>
              <ActionsSection formData={formData} setFormData={setFormData} />
            </div>

            <div style={{ display: activeSection === 'mcp-section' ? 'block' : 'none' }}>
              <MCPServersSection formData={formData} handleAddMCPServer={handleAddMCPServer} handleRemoveMCPServer={handleRemoveMCPServer} handleMCPServerChange={handleMCPServerChange} />
            </div>

            <div style={{ display: activeSection === 'memory-section' ? 'block' : 'none' }}>
              <MemorySettingsSection formData={formData} handleInputChange={handleInputChange} />
            </div>

            <div style={{ display: activeSection === 'prompts-section' ? 'block' : 'none' }}>
              <PromptsGoalsSection formData={formData} handleInputChange={handleInputChange} isGroupForm={isGroupForm} />
            </div>

            <div style={{ display: activeSection === 'advanced-section' ? 'block' : 'none' }}>
              <AdvancedSettingsSection formData={formData} handleInputChange={handleInputChange} />
            </div>
          </div>
        ) : (
          <form className="agent-form" onSubmit={handleSubmit}>
            {/* Form Sections */}
            <div style={{ display: activeSection === 'basic-section' ? 'block' : 'none' }}>
              <BasicInfoSection formData={formData} handleInputChange={handleInputChange} isEdit={isEdit} isGroupForm={isGroupForm} />
            </div>

            <div style={{ display: activeSection === 'model-section' ? 'block' : 'none' }}>
              <ModelSettingsSection formData={formData} handleInputChange={handleInputChange} />
            </div>

            <div style={{ display: activeSection === 'connectors-section' ? 'block' : 'none' }}>
              <ConnectorsSection formData={formData} handleAddConnector={handleAddConnector} handleRemoveConnector={handleRemoveConnector} handleConnectorNameChange={handleConnectorNameChange} handleConnectorConfigChange={handleConnectorConfigChange} />
            </div>

            <div style={{ display: activeSection === 'actions-section' ? 'block' : 'none' }}>
              <ActionsSection formData={formData} setFormData={setFormData} />
            </div>

            <div style={{ display: activeSection === 'mcp-section' ? 'block' : 'none' }}>
              <MCPServersSection formData={formData} handleAddMCPServer={handleAddMCPServer} handleRemoveMCPServer={handleRemoveMCPServer} handleMCPServerChange={handleMCPServerChange} />
            </div>

            <div style={{ display: activeSection === 'memory-section' ? 'block' : 'none' }}>
              <MemorySettingsSection formData={formData} handleInputChange={handleInputChange} />
            </div>

            <div style={{ display: activeSection === 'prompts-section' ? 'block' : 'none' }}>
              <PromptsGoalsSection formData={formData} handleInputChange={handleInputChange} isGroupForm={isGroupForm} />
            </div>

            <div style={{ display: activeSection === 'advanced-section' ? 'block' : 'none' }}>
              <AdvancedSettingsSection formData={formData} handleInputChange={handleInputChange} />
            </div>

            {/* Form Controls */}
            <div className="form-actions">
              <button type="button" className="btn btn-secondary" onClick={() => navigate('/agents')}>
                Cancel
              </button>
              <button type="submit" className="btn btn-primary" disabled={loading}>
                {submitButtonText || (isEdit ? 'Update Agent' : 'Create Agent')}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
};

export default AgentForm;
