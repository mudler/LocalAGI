import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Base action component that renders form fields based on field definitions
 * 
 * @param {Object} props Component props
 * @param {number} props.index Action index
 * @param {Function} props.onActionConfigChange Handler for config changes
 * @param {Function} props.getConfigValue Helper to get config values
 * @param {Array} props.fields Field definitions for this action
 */
const BaseAction = ({ 
  index, 
  onActionConfigChange, 
  getConfigValue,
  fields = []
}) => {
  // Create an object with all the current values
  const currentValues = {};
  
  // Pre-populate with current values or defaults
  fields.forEach(field => {
    currentValues[field.name] = getConfigValue(field.name, field.defaultValue);
  });

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    onActionConfigChange(name, value);
  };

  return (
    <div className="action-template">
      <FormFieldDefinition
        fields={fields}
        values={currentValues}
        onChange={handleFieldChange}
        idPrefix={`action${index}_`}
      />
    </div>
  );
};

export default BaseAction;
