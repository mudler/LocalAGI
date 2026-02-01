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
  handleInputChange,
  metadata 
}) => {
  // Get MCP configuration fields from metadata (mcp_stdio_servers, mcp_prepare_script)
  const mcpFields = metadata?.MCPSection || [];

  // Handle MCP configuration field value changes (FormField passes the event)
  const handleMCPFieldChange = (e) => {
    const { name, value, type, checked } = e.target;
    const field = mcpFields.find(f => f.name === name);
    if (field && field.type === 'checkbox') {
      handleInputChange({
        target: {
          name,
          type: 'checkbox',
          checked
        }
      });
    } else {
      handleInputChange({
        target: {
          name,
          value
        }
      });
    }
  };

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
  const handleFieldChange = (index, e) => {
    const { name, value, type, checked } = e.target;
    
    // Convert value to number if it's a number input
    const processedValue = type === 'number' ? Number(value) : value;
    
    handleMCPServerChange(index, name, type === 'checkbox' ? checked : processedValue);
  };

  return (
    <div id="mcp-section">
      <h3 className="section-title">MCP Servers</h3>
      <p className="section-description">
        Configure MCP servers for this agent.
      </p>

      {mcpFields.length > 0 && (
        <div className="mcp-config-fields mb-4">
          <h4 className="subsection-title">MCP configuration</h4>
          <FormFieldDefinition
            fields={mcpFields}
            values={formData}
            onChange={handleMCPFieldChange}
            idPrefix="mcp_"
          />
        </div>
      )}
      
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
    </div>
  );
};

export default MCPServersSection;
