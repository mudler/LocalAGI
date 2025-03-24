import React from 'react';

/**
 * Prompts & Goals section of the agent form
 */
const PromptsGoalsSection = ({ formData, handleInputChange, isGroupForm }) => {
  // In group form context, we hide the system prompt as it comes from each agent profile
  return (
    <div id="prompts-section">
      <h3 className="section-title">Prompts & Goals</h3>
      
      {!isGroupForm && (
        <div className="mb-4">
          <label htmlFor="system_prompt">System Prompt</label>
          <textarea 
            name="system_prompt" 
            id="system_prompt" 
            value={formData.system_prompt || ''}
            onChange={handleInputChange}
            className="form-control"
            rows="5"
          />
          <small className="form-text text-muted">Instructions that define the agent's behavior</small>
        </div>
      )}

      <div className="mb-4">
        <label htmlFor="goals">Goals</label>
        <textarea 
          name="goals" 
          id="goals" 
          value={formData.goals || ''}
          onChange={handleInputChange}
          className="form-control"
          rows="5"
        />
        <small className="form-text text-muted">Define the agent's goals (one per line)</small>
      </div>

      <div className="mb-4">
        <label htmlFor="constraints">Constraints</label>
        <textarea 
          name="constraints" 
          id="constraints" 
          value={formData.constraints || ''}
          onChange={handleInputChange}
          className="form-control"
          rows="5"
        />
        <small className="form-text text-muted">Define the agent's constraints (one per line)</small>
      </div>

      <div className="mb-4">
        <label htmlFor="tools">Tools</label>
        <textarea 
          name="tools" 
          id="tools" 
          value={formData.tools || ''}
          onChange={handleInputChange}
          className="form-control"
          rows="5"
        />
        <small className="form-text text-muted">Define the agent's tools (one per line)</small>
      </div>
    </div>
  );
};

export default PromptsGoalsSection;
