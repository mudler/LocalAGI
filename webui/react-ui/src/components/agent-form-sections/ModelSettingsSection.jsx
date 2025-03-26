import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Model Settings section of the agent form
 */
const ModelSettingsSection = ({ formData, handleInputChange }) => {
  // Define field definitions for Model Settings section
  const fields = [
    {
      name: 'model',
      label: 'Model',
      type: 'text',
      defaultValue: '',
    },
    {
      name: 'multimodal_model',
      label: 'Multimodal Model',
      type: 'text',
      defaultValue: '',
    },
    {
      name: 'api_url',
      label: 'API URL',
      type: 'text',
      defaultValue: '',
    },
    {
      name: 'api_key',
      label: 'API Key',
      type: 'password',
      defaultValue: '',
    },
    {
      name: 'temperature',
      label: 'Temperature',
      type: 'number',
      defaultValue: 0.7,
      min: 0,
      max: 2,
      step: 0.1,
    },
    {
      name: 'max_tokens',
      label: 'Max Tokens',
      type: 'number',
      defaultValue: 2000,
      min: 1,
    },
  ];

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    handleInputChange({
      target: {
        name,
        value
      }
    });
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
