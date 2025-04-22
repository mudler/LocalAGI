import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * MCP Servers section of the agent form
 */
const MCPServersSection = ({ 
  formData, 
  handleAddMCPServer, 
  handleRemoveMCPServer, 
  handleMCPServerChange,
  handleAddMCPSTDIOServer,
  handleRemoveMCPSTDIOServer,
  handleMCPSTDIOServerChange
}) => {
  // Define field definitions for each MCP server
  const getServerFields = () => [
    {
      name: 'url',
      label: 'URL',
      type: 'text',
      defaultValue: '',
      placeholder: 'https://example.com/mcp',
    },
    {
      name: 'token',
      label: 'API Key',
      type: 'password',
      defaultValue: '',
    },
  ];

  // Define field definitions for each MCP STDIO server
  const getSTDIOServerFields = () => [
    {
      name: 'cmd',
      label: 'Command',
      type: 'text',
      defaultValue: '',
      required: true,
    },
    {
      name: 'args',
      label: 'Arguments',
      type: 'text',
      defaultValue: '',
      required: true,
      helpText: 'Comma-separated list of arguments',
    },
    {
      name: 'env',
      label: 'Environment Variables',
      type: 'text',
      defaultValue: '',
      required: true,
      helpText: 'Comma-separated list of environment variables in KEY=VALUE format',
    },
  ];

  // Handle field value changes for a specific server
  const handleFieldChange = (index, e, isStdio = false) => {
    const { name, value, type, checked } = e.target;
    
    // Convert value to number if it's a number input
    const processedValue = type === 'number' ? Number(value) : value;
    
    // Handle comma-separated values for args and env
    if (name === 'args' || name === 'env') {
      const values = value.split(',').map(v => v.trim()).filter(v => v);
      if (isStdio) {
        handleMCPSTDIOServerChange(index, name, values);
      } else {
        handleMCPServerChange(index, name, values);
      }
    } else {
      if (isStdio) {
        handleMCPSTDIOServerChange(index, name, type === 'checkbox' ? checked : processedValue);
      } else {
        handleMCPServerChange(index, name, type === 'checkbox' ? checked : processedValue);
      }
    }
  };

  return (
    <div id="mcp-section">
      <h3 className="section-title">MCP Servers</h3>
      <p className="section-description">
        Configure MCP servers for this agent.
      </p>
      
      <div className="mcp-servers-container">
        {formData.mcp_servers && formData.mcp_servers.map((server, index) => (
          <div key={index} className="mcp-server-item mb-4">
            <div className="mcp-server-header">
              <h4>MCP Server #{index + 1}</h4>
              <button 
                type="button" 
                className="action-btn delete-btn"
                onClick={() => handleRemoveMCPServer(index)}
              >
                <i className="fas fa-times"></i>
              </button>
            </div>
            
            <FormFieldDefinition
              fields={getServerFields()}
              values={server}
              onChange={(e) => handleFieldChange(index, e)}
              idPrefix={`mcp_server_${index}_`}
            />
          </div>
        ))}
        
        <button 
          type="button" 
          className="action-btn"
          onClick={handleAddMCPServer}
        >
          <i className="fas fa-plus"></i> Add MCP Server
        </button>
      </div>

      <h3 className="section-title mt-4">MCP STDIO Servers</h3>
      <p className="section-description">
        Configure MCP STDIO servers for this agent.
      </p>
      
      <div className="mcp-stdio-servers-container">
        {formData.mcp_stdio_servers && formData.mcp_stdio_servers.map((server, index) => (
          <div key={index} className="mcp-stdio-server-item mb-4">
            <div className="mcp-stdio-server-header">
              <h4>MCP STDIO Server #{index + 1}</h4>
              <button 
                type="button" 
                className="action-btn delete-btn"
                onClick={() => handleRemoveMCPSTDIOServer(index)}
              >
                <i className="fas fa-times"></i>
              </button>
            </div>
            
            <FormFieldDefinition
              fields={getSTDIOServerFields()}
              values={server}
              onChange={(e) => handleFieldChange(index, e, true)}
              idPrefix={`mcp_stdio_server_${index}_`}
            />
          </div>
        ))}
        
        <button 
          type="button" 
          className="action-btn"
          onClick={handleAddMCPSTDIOServer}
        >
          <i className="fas fa-plus"></i> Add MCP STDIO Server
        </button>
      </div>
    </div>
  );
};

export default MCPServersSection;
