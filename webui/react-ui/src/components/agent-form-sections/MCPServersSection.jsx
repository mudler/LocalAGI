import React from 'react';

/**
 * MCP Servers section of the agent form
 */
const MCPServersSection = ({ 
  formData, 
  handleAddMCPServer, 
  handleRemoveMCPServer, 
  handleMCPServerChange 
}) => {
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
            
            <div className="mb-3">
              <label htmlFor={`mcp-url-${index}`}>URL</label>
              <input 
                type="text" 
                id={`mcp-url-${index}`}
                value={server.url || ''}
                onChange={(e) => handleMCPServerChange(index, 'url', e.target.value)}
                className="form-control"
                placeholder="https://example.com/mcp"
              />
            </div>
            
            <div className="mb-3">
              <label htmlFor={`mcp-api-key-${index}`}>API Key</label>
              <input 
                type="password" 
                id={`mcp-api-key-${index}`}
                value={server.api_key || ''}
                onChange={(e) => handleMCPServerChange(index, 'api_key', e.target.value)}
                className="form-control"
              />
            </div>
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
