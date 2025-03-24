import React, { useState } from 'react';

/**
 * Fallback connector template for unknown connector types
 */
const FallbackConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  const [newConfigKey, setNewConfigKey] = useState('');
  
  // Parse config if it's a string
  let parsedConfig = connector.config;
  if (typeof parsedConfig === 'string') {
    try {
      parsedConfig = JSON.parse(parsedConfig);
    } catch (err) {
      console.error('Error parsing config:', err);
      parsedConfig = {};
    }
  } else if (!parsedConfig) {
    parsedConfig = {};
  }

  // Handle adding a new custom field
  const handleAddCustomField = () => {
    if (newConfigKey) {
      onConnectorConfigChange(index, newConfigKey, '');
      setNewConfigKey('');
    }
  };

  return (
    <div className="connector-template">
      {/* Individual field inputs */}
      {parsedConfig && Object.entries(parsedConfig).map(([key, value]) => (
        <div key={key} className="form-group mb-3">
          <label htmlFor={`connector-${index}-${key}`}>{key}</label>
          <input
            type="text"
            id={`connector-${index}-${key}`}
            className="form-control"
            value={value}
            onChange={(e) => onConnectorConfigChange(index, key, e.target.value)}
          />
        </div>
      ))}

      {/* Add custom configuration field */}
      <div className="add-config-field mt-4">
        <h5>Add Custom Configuration Field</h5>
        <div className="input-group mb-3">
          <input
            type="text"
            placeholder="New config key"
            className="form-control"
            value={newConfigKey}
            onChange={(e) => setNewConfigKey(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleAddCustomField()}
          />
          <button
            type="button"
            className="btn btn-outline-primary"
            onClick={handleAddCustomField}
          >
            <i className="fas fa-plus"></i> Add Field
          </button>
        </div>
      </div>
    </div>
  );
};

export default FallbackConnector;
