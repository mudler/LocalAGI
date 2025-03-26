import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Base connector component that renders form fields based on field definitions
 * 
 * @param {Object} props Component props
 * @param {Object} props.connector Connector data
 * @param {number} props.index Connector index
 * @param {Function} props.onConnectorConfigChange Handler for config changes
 * @param {Function} props.getConfigValue Helper to get config values
 * @param {Array} props.fields Field definitions for this connector
 */
const BaseConnector = ({ 
  connector, 
  index, 
  onConnectorConfigChange, 
  getConfigValue,
  fields = []
}) => {
  // Create an object with all the current values
  const currentValues = {};
  
  // Pre-populate with current values or defaults
  fields.forEach(field => {
    currentValues[field.name] = getConfigValue(connector, field.name, field.defaultValue);
  });

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    onConnectorConfigChange(index, name, value);
  };

  return (
    <div className="connector-template">
      <FormFieldDefinition
        fields={fields}
        values={currentValues}
        onChange={handleFieldChange}
        idPrefix={`connector${index}_`}
      />
    </div>
  );
};

export default BaseConnector;
