import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * MCP Servers section of the agent form
 */
const MCPServersSection = ({ 
  formData, 
  handleAddMCPServer, 
  handleRemoveMCPServer, 
  handleMCPServerChange 
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

  // Handle field value changes for a specific server
  const handleFieldChange = (index, name, value) => {
    handleMCPServerChange(index, name, value);
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
                className="remove-btn"
                onClick={() => handleRemoveMCPServer(index)}
              >
                <i className="fas fa-times"></i>
              </button>
            </div>
            
            <FormFieldDefinition
              fields={getServerFields()}
              values={server}
              onChange={(name, value) => handleFieldChange(index, name, value)}
              idPrefix={`mcp_server_${index}_`}
            />
          </div>
        ))}
        
        <button 
          type="button" 
          className="add-btn"
          onClick={handleAddMCPServer}
        >
          <i className="fas fa-plus"></i> Add MCP Server
        </button>
      </div>
    </div>
  );
};

export default MCPServersSection;
