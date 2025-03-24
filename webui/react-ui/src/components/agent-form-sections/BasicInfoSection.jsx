import React from 'react';

/**
 * Basic Information section of the agent form
 */
const BasicInfoSection = ({ formData, handleInputChange, isEdit, isGroupForm }) => {
  // In group form context, we hide the basic info section entirely
  if (isGroupForm) {
    return null;
  }
  
  return (
    <div id="basic-section">
      <h3 className="section-title">Basic Information</h3>
      
      <div className="mb-4">
        <label htmlFor="name">Name</label>
        <input 
          type="text" 
          name="name" 
          id="name" 
          value={formData.name || ''}
          onChange={handleInputChange}
          required
          disabled={isEdit} // Disable name field in edit mode
        />
        {isEdit && <small className="form-text text-muted">Agent name cannot be changed after creation</small>}
      </div>

      <div className="mb-4">
        <label htmlFor="description">Description</label>
        <textarea 
          name="description" 
          id="description" 
          value={formData.description || ''}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="identity_guidance">Identity Guidance</label>
        <textarea 
          name="identity_guidance" 
          id="identity_guidance" 
          value={formData.identity_guidance || ''}
          onChange={handleInputChange}
        />
      </div>

      <div className="mb-4">
        <label htmlFor="random_identity" className="checkbox-label">
          <input 
            type="checkbox" 
            name="random_identity" 
            id="random_identity"
            checked={formData.random_identity || false}
            onChange={handleInputChange}
          />
          Random Identity
        </label>
      </div>

      <div className="mb-4">
        <label htmlFor="hud" className="checkbox-label">
          <input 
            type="checkbox" 
            name="hud" 
            id="hud"
            checked={formData.hud || false}
            onChange={handleInputChange}
          />
          HUD
        </label>
      </div>
    </div>
  );
};

export default BasicInfoSection;
