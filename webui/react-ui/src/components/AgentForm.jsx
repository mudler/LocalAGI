import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

// Import form sections
import BasicInfoSection from './agent-form-sections/BasicInfoSection';
import ConnectorsSection from './agent-form-sections/ConnectorsSection';
import ActionsSection from './agent-form-sections/ActionsSection';
import MCPServersSection from './agent-form-sections/MCPServersSection';
import MemorySettingsSection from './agent-form-sections/MemorySettingsSection';
import ModelSettingsSection from './agent-form-sections/ModelSettingsSection';
import PromptsGoalsSection from './agent-form-sections/PromptsGoalsSection';
import AdvancedSettingsSection from './agent-form-sections/AdvancedSettingsSection';
import ExportSection from './agent-form-sections/ExportSection';
import FiltersSection from './agent-form-sections/FiltersSection';

const AgentForm = ({ 
  isEdit = false, 
  formData, 
  setFormData, 
  onSubmit, 
  loading = false, 
  submitButtonText, 
  isGroupForm = false,
  noFormWrapper = false,
  metadata = null,
}) => {
  const navigate = useNavigate();
  const [activeSection, setActiveSection] = useState(isGroupForm ? 'model-section' : 'basic-section');

  // Handle input changes
  const handleInputChange = (e) => {
    const { value, type, checked } = e.target;
    const name = e.target.name;
    // Guard: name must be a string (some sections pass synthetic events where name can be wrong)
    if (typeof name !== 'string') return;
    
    // Convert value to number if it's a number input
    const processedValue = type === 'number' ? Number(value) : value;
    
    if (name.includes('.')) {
      const [parent, child] = name.split('.');
      setFormData({
        ...formData,
        [parent]: {
          ...formData[parent],
          [child]: type === 'checkbox' ? checked : processedValue
        }
      });
    } else {
      setFormData({
        ...formData,
        [name]: type === 'checkbox' ? checked : processedValue
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

  // Handle connector change (simplified)
  const handleConnectorChange = (index, updatedConnector) => {
    const updatedConnectors = [...formData.connectors];
    updatedConnectors[index] = updatedConnector;
    setFormData({
      ...formData,
      connectors: updatedConnectors
    });
  };


  // Handle adding a connector
  const handleAddConnector = () => {
    setFormData({
      ...formData,
      connectors: [
        ...(formData.connectors || []),
        { type: '', config: '{}' }
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
  
  const handleAddDynamicPrompt = () => {
    setFormData({
      ...formData,
      dynamic_prompts: [
        ...(formData.dynamic_prompts || []),
        { type: '', config: '{}' }
      ]
    });
  };

  const handleRemoveDynamicPrompt = (index) => {
    const updatedDynamicPrompts = [...formData.dynamic_prompts];
    updatedDynamicPrompts.splice(index, 1);
    setFormData({
      ...formData,
      dynamic_prompts: updatedDynamicPrompts,
    });
  };
  
  const handleDynamicPromptChange = (index, updatedPrompt) => {
    const updatedPrompts = [...formData.dynamic_prompts];
    updatedPrompts[index] = updatedPrompt;
    setFormData({
      ...formData,
      dynamic_prompts: updatedPrompts
    });
  };

  // Handle adding an MCP server
  const handleAddMCPServer = () => {
    setFormData({
      ...formData,
      mcp_servers: [
        ...(formData.mcp_servers || []),
        { url: '', token: '' }
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
  const handleMCPServerChange = (index, field, value) => {
    const updatedMCPServers = [...formData.mcp_servers];
    updatedMCPServers[index] = { 
      ...updatedMCPServers[index],
      [field]: value 
    };
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
            className={`wizard-nav-item ${activeSection === 'filters-section' ? 'active' : ''}`}
            onClick={() => handleSectionChange('filters-section')}
          >
            <i className="fas fa-shield"></i>
            Filters &amp; Triggers
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
          {isEdit && (
            <>
              <li 
                className={`wizard-nav-item ${activeSection === 'export-section' ? 'active' : ''}`} 
                onClick={() => handleSectionChange('export-section')}
              >
                <i className="fas fa-file-export"></i>
                Export Data
              </li>
            </>
          )}
        </ul>
      </div>

      {/* Form Content */}
      <div className="form-content-area">
        {noFormWrapper ? (
          <div className='agent-form'>
            {/* Form Sections */}
            <div style={{ display: activeSection === 'basic-section' ? 'block' : 'none' }}>
              <BasicInfoSection formData={formData} handleInputChange={handleInputChange} isEdit={isEdit} isGroupForm={isGroupForm} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'model-section' ? 'block' : 'none' }}>
              <ModelSettingsSection formData={formData} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'connectors-section' ? 'block' : 'none' }}>
              <ConnectorsSection formData={formData} handleAddConnector={handleAddConnector} handleRemoveConnector={handleRemoveConnector} handleConnectorChange={handleConnectorChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'filters-section' ? 'block' : 'none' }}>
              <FiltersSection formData={formData} setFormData={setFormData} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'actions-section' ? 'block' : 'none' }}>
              <ActionsSection formData={formData} setFormData={setFormData} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'mcp-section' ? 'block' : 'none' }}>
              <MCPServersSection formData={formData} handleAddMCPServer={handleAddMCPServer} handleRemoveMCPServer={handleRemoveMCPServer} handleMCPServerChange={handleMCPServerChange} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'memory-section' ? 'block' : 'none' }}>
              <MemorySettingsSection formData={formData} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'prompts-section' ? 'block' : 'none' }}>
              <PromptsGoalsSection 
                formData={formData} 
                handleInputChange={handleInputChange} 
                isGroupForm={isGroupForm} 
                metadata={metadata}
                onAddPrompt={handleAddDynamicPrompt}
                onRemovePrompt={handleRemoveDynamicPrompt}
                handleDynamicPromptChange={handleDynamicPromptChange}
              />
            </div>

            <div style={{ display: activeSection === 'advanced-section' ? 'block' : 'none' }}>
              <AdvancedSettingsSection formData={formData} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            {isEdit && (
              <>
                <div style={{ display: activeSection === 'export-section' ? 'block' : 'none' }}>
                  <ExportSection agentName={formData.name} />
                </div>
              </>
            )}
          </div>
        ) : (
          <form className="agent-form" onSubmit={handleSubmit} noValidate>
            {/* Form Sections */}
            <div style={{ display: activeSection === 'basic-section' ? 'block' : 'none' }}>
              <BasicInfoSection formData={formData} handleInputChange={handleInputChange} isEdit={isEdit} isGroupForm={isGroupForm} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'model-section' ? 'block' : 'none' }}>
              <ModelSettingsSection formData={formData} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'connectors-section' ? 'block' : 'none' }}>
              <ConnectorsSection formData={formData} handleAddConnector={handleAddConnector} handleRemoveConnector={handleRemoveConnector} handleConnectorChange={handleConnectorChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'filters-section' ? 'block' : 'none' }}>
              <FiltersSection formData={formData} setFormData={setFormData} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'actions-section' ? 'block' : 'none' }}>
              <ActionsSection formData={formData} setFormData={setFormData} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'mcp-section' ? 'block' : 'none' }}>
              <MCPServersSection formData={formData} handleAddMCPServer={handleAddMCPServer} handleRemoveMCPServer={handleRemoveMCPServer} handleMCPServerChange={handleMCPServerChange} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'memory-section' ? 'block' : 'none' }}>
              <MemorySettingsSection formData={formData} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            <div style={{ display: activeSection === 'prompts-section' ? 'block' : 'none' }}>
              <PromptsGoalsSection 
                formData={formData} 
                handleInputChange={handleInputChange} 
                isGroupForm={isGroupForm} 
                metadata={metadata}
                onAddPrompt={handleAddDynamicPrompt}
                onRemovePrompt={handleRemoveDynamicPrompt}
                handleDynamicPromptChange={handleDynamicPromptChange}
              />
            </div>

            <div style={{ display: activeSection === 'advanced-section' ? 'block' : 'none' }}>
              <AdvancedSettingsSection formData={formData} handleInputChange={handleInputChange} metadata={metadata} />
            </div>

            {isEdit && (
              <>
                <div style={{ display: activeSection === 'export-section' ? 'block' : 'none' }}>
                  <ExportSection agentName={formData.name} />
                </div>
              </>
            )}

            {/* Form Controls */}
            <div className="form-actions" style={{ display: 'flex', gap: '1rem', justifyContent: 'flex-end' }}>
              <button type="button" className="action-btn" onClick={() => navigate('/agents')}>
                <i className="fas fa-times"></i> Cancel
              </button>
              <button type="submit" className="action-btn" disabled={loading}>
                <i className="fas fa-save"></i> {submitButtonText || (isEdit ? 'Update Agent' : 'Create Agent')}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
};

export default AgentForm;
