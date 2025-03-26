import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Prompts & Goals section of the agent form
 */
const PromptsGoalsSection = ({ formData, handleInputChange, isGroupForm }) => {
  // Define field definitions for Prompts & Goals section
  const getFields = () => {
    // Base fields that are always shown
    const baseFields = [
      {
        name: 'goals',
        label: 'Goals',
        type: 'textarea',
        defaultValue: '',
        helpText: 'Define the agent\'s goals (one per line)',
        rows: 5,
      },
      {
        name: 'constraints',
        label: 'Constraints',
        type: 'textarea',
        defaultValue: '',
        helpText: 'Define the agent\'s constraints (one per line)',
        rows: 5,
      },
      {
        name: 'tools',
        label: 'Tools',
        type: 'textarea',
        defaultValue: '',
        helpText: 'Define the agent\'s tools (one per line)',
        rows: 5,
      },
    ];

    // Only include system_prompt field if not in group form context
    if (!isGroupForm) {
      return [
        {
          name: 'system_prompt',
          label: 'System Prompt',
          type: 'textarea',
          defaultValue: '',
          helpText: 'Instructions that define the agent\'s behavior',
          rows: 5,
        },
        ...baseFields
      ];
    }

    return baseFields;
  };

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
    <div id="prompts-section">
      <h3 className="section-title">Prompts & Goals</h3>
      
      <FormFieldDefinition
        fields={getFields()}
        values={formData}
        onChange={handleFieldChange}
        idPrefix="prompts_"
      />
    </div>
  );
};

export default PromptsGoalsSection;
