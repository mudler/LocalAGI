import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Memory Settings section of the agent form
 * 
 * @param {Object} props Component props
 * @param {Object} props.formData Current form data values
 * @param {Function} props.handleInputChange Handler for input changes
 * @param {Object} props.metadata Field metadata from the backend
 */
const MemorySettingsSection = ({ formData, handleInputChange, metadata }) => {
  // Get fields from metadata
  const fields = metadata?.MemorySettingsSection || [];

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
          value
        }
      });
    }
  };

  return (
    <div id="memory-section">
      <h3 className="section-title">Memory Settings</h3>
      
      <FormFieldDefinition
        fields={fields}
        values={formData}
        onChange={handleFieldChange}
        idPrefix="memory_"
      />
    </div>
  );
};

export default MemorySettingsSection;
