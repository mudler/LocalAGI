import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Advanced Settings section of the agent form
 */
const AdvancedSettingsSection = ({ formData, handleInputChange }) => {
  // Define field definitions for Advanced Settings section
  const fields = [
    {
      name: 'max_steps',
      label: 'Max Steps',
      type: 'number',
      defaultValue: 10,
      helpText: 'Maximum number of steps the agent can take',
      required: true,
    },
    {
      name: 'max_iterations',
      label: 'Max Iterations',
      type: 'number',
      defaultValue: 5,
      helpText: 'Maximum number of iterations for each step',
      required: true,
    },
    {
      name: 'autonomous',
      label: 'Autonomous Mode',
      type: 'checkbox',
      defaultValue: false,
      helpText: 'Allow the agent to operate autonomously',
    },
    {
      name: 'verbose',
      label: 'Verbose Mode',
      type: 'checkbox',
      defaultValue: false,
      helpText: 'Enable detailed logging',
    },
    {
      name: 'allow_code_execution',
      label: 'Allow Code Execution',
      type: 'checkbox',
      defaultValue: false,
      helpText: 'Allow the agent to execute code (use with caution)',
    },
  ];

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    // For checkboxes, convert string 'true'/'false' to boolean
    if (['autonomous', 'verbose', 'allow_code_execution'].includes(name)) {
      handleInputChange({
        target: {
          name,
          type: 'checkbox',
          checked: value === 'true'
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
    <div id="advanced-section">
      <h3 className="section-title">Advanced Settings</h3>
      
      <FormFieldDefinition
        fields={fields}
        values={formData}
        onChange={handleFieldChange}
        idPrefix="advanced_"
      />
    </div>
  );
};

export default AdvancedSettingsSection;
