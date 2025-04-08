import React from 'react';
import ConfigForm from './ConfigForm';

/**
 * PromptForm component
 * Renders prompt configuration forms based on field group metadata
 */
function PromptForm({ 
  prompts = [], 
  onAddPrompt, 
  onRemovePrompt, 
  onChange,
  fieldGroups = []
}) {
  return (
    <ConfigForm
      items={prompts}
      fieldGroups={fieldGroups}
      onChange={onChange}
      onRemove={onRemovePrompt}
      onAdd={onAddPrompt}
      itemType="dynamic_prompt"
      typeField="type"
      addButtonText="Add Dynamic Prompt"
    />
  );
}

export default PromptForm;

