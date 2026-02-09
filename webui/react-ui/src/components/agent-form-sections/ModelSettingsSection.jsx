import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Model Settings section of the agent form
 * 
 * @param {Object} props Component props
 * @param {Object} props.formData Current form data values
 * @param {Function} props.handleInputChange Handler for input changes
 * @param {Object} props.metadata Field metadata from the backend
 */
const ModelSettingsSection = ({ formData, handleInputChange, metadata }) => {
  // Get fields from metadata
  const fields = metadata?.ModelSettingsSection || [];

  // Handle field value changes (FormField passes the event)
  const handleFieldChange = (e) => {
    const { name, value, type, checked } = e.target;
    const field = fields.find(f => f.name === name);
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
          type,
          value
        }
      });
    }
  };

  return (
    <div id="model-section">
      <h3 className="section-title">Model Settings</h3>
      
      <FormFieldDefinition
        fields={fields}
        values={formData}
        onChange={handleFieldChange}
        idPrefix="model_"
      />
    </div>
  );
};

export default ModelSettingsSection;
