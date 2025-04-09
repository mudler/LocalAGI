import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';
import DynamicPromptForm from '../DynamicPromptForm';

/**
 * Prompts & Goals section of the agent form
 * 
 * @param {Object} props Component props
 * @param {Object} props.formData Current form data values
 * @param {Function} props.handleInputChange Handler for input changes
 * @param {boolean} props.isGroupForm Whether the form is for a group
 * @param {Object} props.metadata Field metadata from the backend
 */
const PromptsGoalsSection = ({ 
  formData, 
  handleInputChange, 
  isGroupForm, 
  metadata,
  onAddPrompt,
  onRemovePrompt,
  handleDynamicPromptChange
}) => {
  // Get fields based on metadata and form context
  const getFields = () => {
    if (!metadata?.PromptsGoalsSection) {
      return [];
    }
    
    // If in group form, filter out system_prompt
    if (isGroupForm) {
      return metadata.PromptsGoalsSection.filter(field => field.name !== 'system_prompt');
    }
    
    return metadata.PromptsGoalsSection;
  };

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    const field = getFields().find(f => f.name === name);
    if (field && field.type === 'checkbox') {
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
    <div id="prompts-section">
      <h3 className="section-title">Prompts & Goals</h3>
      
      <FormFieldDefinition
        fields={getFields()}
        values={formData}
        onChange={handleFieldChange}
        idPrefix="prompts_"
      />

      <DynamicPromptForm
        prompts={formData.dynamicPrompts || []}
        onAddPrompt={onAddPrompt}
        onRemovePrompt={onRemovePrompt}
        onChange={handleDynamicPromptChange}
        fieldGroups={metadata?.dynamicPrompts || []}
      />
    </div>
  );
};

export default PromptsGoalsSection;
