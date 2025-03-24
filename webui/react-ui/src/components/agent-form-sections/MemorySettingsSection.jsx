import React from 'react';

/**
 * Memory Settings section of the agent form
 */
const MemorySettingsSection = ({ formData, handleInputChange }) => {
  return (
    <div id="memory-section">
      <h3 className="section-title">Memory Settings</h3>
      
      <div className="mb-4">
        <label htmlFor="memory_provider">Memory Provider</label>
        <select
          name="memory_provider"
          id="memory_provider"
          value={formData.memory_provider || 'local'}
          onChange={handleInputChange}
          className="form-control"
        >
          <option value="local">Local</option>
          <option value="redis">Redis</option>
          <option value="postgres">PostgreSQL</option>
        </select>
      </div>

      <div className="mb-4">
        <label htmlFor="memory_collection">Memory Collection</label>
        <input 
          type="text" 
          name="memory_collection" 
          id="memory_collection" 
          value={formData.memory_collection || ''}
          onChange={handleInputChange}
          className="form-control"
          placeholder="agent_memories"
        />
      </div>

      <div className="mb-4">
        <label htmlFor="memory_url">Memory URL</label>
        <input 
          type="text" 
          name="memory_url" 
          id="memory_url" 
          value={formData.memory_url || ''}
          onChange={handleInputChange}
          className="form-control"
          placeholder="redis://localhost:6379"
        />
        <small className="form-text text-muted">Connection URL for Redis or PostgreSQL</small>
      </div>

      <div className="mb-4">
        <label htmlFor="memory_window_size">Memory Window Size</label>
        <input 
          type="number" 
          name="memory_window_size" 
          id="memory_window_size" 
          min="1"
          value={formData.memory_window_size || 10}
          onChange={handleInputChange}
          className="form-control"
        />
        <small className="form-text text-muted">Number of recent messages to include in context window</small>
      </div>
    </div>
  );
};

export default MemorySettingsSection;
