import React from 'react';

/**
 * Advanced Settings section of the agent form
 */
const AdvancedSettingsSection = ({ formData, handleInputChange }) => {
  return (
    <div id="advanced-section">
      <h3 className="section-title">Advanced Settings</h3>
      
      <div className="mb-4">
        <label htmlFor="max_steps">Max Steps</label>
        <input 
          type="number" 
          name="max_steps" 
          id="max_steps" 
          min="1"
          value={formData.max_steps || 10}
          onChange={handleInputChange}
          className="form-control"
        />
        <small className="form-text text-muted">Maximum number of steps the agent can take</small>
      </div>

      <div className="mb-4">
        <label htmlFor="max_iterations">Max Iterations</label>
        <input 
          type="number" 
          name="max_iterations" 
          id="max_iterations" 
          min="1"
          value={formData.max_iterations || 5}
          onChange={handleInputChange}
          className="form-control"
        />
        <small className="form-text text-muted">Maximum number of iterations for each step</small>
      </div>

      <div className="mb-4">
        <label htmlFor="autonomous" className="checkbox-label">
          <input 
            type="checkbox" 
            name="autonomous" 
            id="autonomous"
            checked={formData.autonomous || false}
            onChange={handleInputChange}
          />
          Autonomous Mode
        </label>
        <small className="form-text text-muted">Allow the agent to operate autonomously</small>
      </div>

      <div className="mb-4">
        <label htmlFor="verbose" className="checkbox-label">
          <input 
            type="checkbox" 
            name="verbose" 
            id="verbose"
            checked={formData.verbose || false}
            onChange={handleInputChange}
          />
          Verbose Mode
        </label>
        <small className="form-text text-muted">Enable detailed logging</small>
      </div>

      <div className="mb-4">
        <label htmlFor="allow_code_execution" className="checkbox-label">
          <input 
            type="checkbox" 
            name="allow_code_execution" 
            id="allow_code_execution"
            checked={formData.allow_code_execution || false}
            onChange={handleInputChange}
          />
          Allow Code Execution
        </label>
        <small className="form-text text-muted">Allow the agent to execute code (use with caution)</small>
      </div>
    </div>
  );
};

export default AdvancedSettingsSection;
