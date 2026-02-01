import React, { useMemo } from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

// Parse mcp_stdio_servers JSON string to array of { name, command, args, env }
function parseStdioJson(str) {
  if (!str || typeof str !== 'string') return [];
  try {
    const parsed = JSON.parse(str);
    const mcpServers = parsed?.mcpServers || {};
    return Object.entries(mcpServers).map(([name, s]) => ({
      name: name || '',
      command: s?.command ?? '',
      args: Array.isArray(s?.args) ? [...s.args] : [],
      env: s?.env && typeof s.env === 'object' && !Array.isArray(s.env) ? { ...s.env } : {},
    }));
  } catch {
    return [];
  }
}

// Build JSON string from array of STDIO servers (unique keys: name or server0, server1, etc.)
function buildStdioJson(list) {
  const mcpServers = {};
  const usedKeys = new Set();
  list.forEach((item, index) => {
    let key = (item.name && item.name.trim()) ? item.name.trim() : `server${index}`;
    while (usedKeys.has(key)) {
      key = `${key}_${index}`;
    }
    usedKeys.add(key);
    mcpServers[key] = {
      command: item.command || '',
      args: item.args || [],
      env: item.env && typeof item.env === 'object' ? { ...item.env } : {},
    };
  });
  return JSON.stringify({ mcpServers }, null, 2);
}

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
  // MCP configuration fields excluding mcp_stdio_servers (handled by dynamic STDIO block below)
  const mcpFields = useMemo(
    () => (metadata?.MCPSection || []).filter((f) => f.name !== 'mcp_stdio_servers'),
    [metadata?.MCPSection]
  );

  // Parsed STDIO servers list derived from formData.mcp_stdio_servers
  const stdioList = useMemo(
    () => parseStdioJson(formData.mcp_stdio_servers),
    [formData.mcp_stdio_servers]
  );

  const setStdioJson = (newList) => {
    handleInputChange({
      target: { name: 'mcp_stdio_servers', value: buildStdioJson(newList) },
    });
  };

  const addStdioServer = () => {
    setStdioJson([...stdioList, { name: '', command: '', args: [], env: {} }]);
  };

  const removeStdioServer = (index) => {
    setStdioJson(stdioList.filter((_, i) => i !== index));
  };

  const updateStdioServer = (index, field, value) => {
    const next = stdioList.map((s, i) =>
      i === index ? { ...s, [field]: value } : s
    );
    setStdioJson(next);
  };

  const addArg = (serverIndex, argValue = '') => {
    const server = stdioList[serverIndex];
    if (!server) return;
    updateStdioServer(serverIndex, 'args', [...(server.args || []), argValue]);
  };

  const removeArg = (serverIndex, argIndex) => {
    const server = stdioList[serverIndex];
    if (!server) return;
    const args = (server.args || []).filter((_, i) => i !== argIndex);
    updateStdioServer(serverIndex, 'args', args);
  };

  const updateArg = (serverIndex, argIndex, value) => {
    const server = stdioList[serverIndex];
    if (!server) return;
    const args = [...(server.args || [])];
    args[argIndex] = value;
    updateStdioServer(serverIndex, 'args', args);
  };

  const addEnv = (serverIndex, key = '', value = '') => {
    const server = stdioList[serverIndex];
    if (!server) return;
    const env = { ...(server.env || {}), [key || `key_${Date.now()}`]: value };
    updateStdioServer(serverIndex, 'env', env);
  };

  const removeEnv = (serverIndex, envKey) => {
    const server = stdioList[serverIndex];
    if (!server) return;
    const env = { ...(server.env || {}) };
    delete env[envKey];
    updateStdioServer(serverIndex, 'env', env);
  };

  const updateEnvKey = (serverIndex, oldKey, newKey) => {
    const server = stdioList[serverIndex];
    if (!server) return;
    const env = { ...(server.env || {}) };
    const val = env[oldKey];
    delete env[oldKey];
    env[newKey || oldKey] = val;
    updateStdioServer(serverIndex, 'env', env);
  };

  const updateEnvValue = (serverIndex, envKey, value) => {
    const server = stdioList[serverIndex];
    if (!server) return;
    const env = { ...(server.env || {}), [envKey]: value };
    updateStdioServer(serverIndex, 'env', env);
  };

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

      <div className="mcp-block mcp-stdio-block mb-4">
        <h4 className="subsection-title">MCP STDIO Servers</h4>
        <p className="section-description">
          Configure MCP servers that run as local commands (e.g. docker run). Each server has a name, command, args, and env.
        </p>
        {stdioList.map((server, index) => (
          <div key={index} className="mcp-server-item mb-4">
            <div className="mcp-server-header">
              <input
                type="text"
                className="form-control stdio-server-name-input"
                placeholder="Server name (e.g. memory)"
                value={server.name || ''}
                onChange={(e) => updateStdioServer(index, 'name', e.target.value)}
                aria-label="STDIO server name"
                style={{ flex: 1, marginRight: '12px' }}
              />
              <button
                type="button"
                className="action-btn delete-btn"
                onClick={() => removeStdioServer(index)}
                aria-label="Remove STDIO server"
              >
                <i className="fas fa-times"></i>
              </button>
            </div>
            <div className="form-group mb-3">
              <label htmlFor={`stdio_cmd_${index}`}>Command</label>
              <input
                type="text"
                id={`stdio_cmd_${index}`}
                className="form-control"
                placeholder="e.g. docker"
                value={server.command || ''}
                onChange={(e) => updateStdioServer(index, 'command', e.target.value)}
              />
            </div>
            <div className="form-group mb-3">
              <label>Args</label>
              {(server.args || []).map((arg, argIndex) => (
                <div key={argIndex} className="input-group mb-2" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Argument"
                    value={arg}
                    onChange={(e) => updateArg(index, argIndex, e.target.value)}
                  />
                  <button
                    type="button"
                    className="action-btn delete-btn"
                    onClick={() => removeArg(index, argIndex)}
                    aria-label="Remove arg"
                  >
                    <i className="fas fa-times"></i>
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="action-btn"
                onClick={() => addArg(index)}
              >
                <i className="fas fa-plus"></i> Add Arg
              </button>
            </div>
            <div className="form-group mb-3">
              <label>Environment</label>
              {Object.entries(server.env || {}).map(([envKey, envVal]) => (
                <div key={envKey} className="input-group mb-2" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Key"
                    value={envKey}
                    onChange={(e) => updateEnvKey(index, envKey, e.target.value)}
                  />
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Value"
                    value={envVal}
                    onChange={(e) => updateEnvValue(index, envKey, e.target.value)}
                  />
                  <button
                    type="button"
                    className="action-btn delete-btn"
                    onClick={() => removeEnv(index, envKey)}
                    aria-label="Remove env"
                  >
                    <i className="fas fa-times"></i>
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="action-btn"
                onClick={() => addEnv(index)}
              >
                <i className="fas fa-plus"></i> Add Env
              </button>
            </div>
          </div>
        ))}
        <button type="button" className="action-btn" onClick={addStdioServer}>
          <i className="fas fa-plus"></i> Add MCP STDIO Server
        </button>
        <details className="mt-3">
          <summary className="subsection-title" style={{ cursor: 'pointer' }}>Edit as JSON</summary>
          <textarea
            className="form-control mt-2"
            rows={8}
            placeholder='{"mcpServers":{"memory":{"command":"docker","args":["run","-i","--rm",...],"env":{...}}}}'
            value={formData.mcp_stdio_servers || ''}
            onChange={(e) => handleInputChange({ target: { name: 'mcp_stdio_servers', value: e.target.value } })}
            spellCheck={false}
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </details>
      </div>

      <div className="mcp-block mcp-http-block mcp-servers-container">
        <h4 className="subsection-title">MCP HTTP Servers</h4>
        <p className="section-description">
          Configure MCP servers that connect over HTTP (URL and optional API key).
        </p>
        {formData.mcp_servers && formData.mcp_servers.map((server, index) => (
          <div key={index} className="mcp-server-item mb-4">
            <div className="mcp-server-header">
              <h4>MCP HTTP Server #{index + 1}</h4>
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
          <i className="fas fa-plus"></i> Add MCP HTTP Server
        </button>
      </div>
    </div>
  );
};

export default MCPServersSection;
