import React from 'react';

/**
 * Model Settings section of the agent form
 */
const ModelSettingsSection = ({ formData, handleInputChange }) => {
  return (
    <div id="model-section">
      <h3 className="section-title">Model Settings</h3>
      
      <div className="mb-4">
        <label htmlFor="model">Model</label>
        <input 
          type="text" 
          name="model" 
          id="model" 
          value={formData.model || ''}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="multimodal_model">Multimodal Model</label>
        <input 
          type="text" 
          name="multimodal_model" 
          id="multimodal_model" 
          value={formData.multimodal_model || ''}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="api_url">API URL</label>
        <input 
          type="text" 
          name="api_url" 
          id="api_url" 
          value={formData.api_url || ''}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="api_key">API Key</label>
        <input 
          type="password" 
          name="api_key" 
          id="api_key" 
          value={formData.api_key || ''}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="temperature">Temperature</label>
        <input 
          type="number" 
          name="temperature" 
          id="temperature" 
          min="0" 
          max="2" 
          step="0.1"
          value={formData.temperature || 0.7}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="max_tokens">Max Tokens</label>
        <input 
          type="number" 
          name="max_tokens" 
          id="max_tokens" 
          min="1"
          value={formData.max_tokens || 2000}
          onChange={handleInputChange}
        />
      </div>
    </div>
  );
};

export default ModelSettingsSection;
