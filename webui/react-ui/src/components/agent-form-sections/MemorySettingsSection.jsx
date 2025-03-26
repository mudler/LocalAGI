import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Memory Settings section of the agent form
 */
const MemorySettingsSection = ({ formData, handleInputChange }) => {
  // Define field definitions for Memory Settings section
  const fields = [
    {
      name: 'memory_provider',
      label: 'Memory Provider',
      type: 'select',
      defaultValue: 'local',
      options: [
        { value: 'local', label: 'Local' },
        { value: 'redis', label: 'Redis' },
        { value: 'postgres', label: 'PostgreSQL' },
      ],
    },
    {
      name: 'memory_collection',
      label: 'Memory Collection',
      type: 'text',
      defaultValue: '',
      placeholder: 'agent_memories',
    },
    {
      name: 'memory_url',
      label: 'Memory URL',
      type: 'text',
      defaultValue: '',
      placeholder: 'redis://localhost:6379',
      helpText: 'Connection URL for Redis or PostgreSQL',
    },
    {
      name: 'memory_window_size',
      label: 'Memory Window Size',
      type: 'number',
      defaultValue: 10,
      helpText: 'Number of recent messages to include in context window',
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
