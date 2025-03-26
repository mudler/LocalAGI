import React from 'react';
import FormFieldDefinition from '../common/FormFieldDefinition';

/**
 * Basic Information section of the agent form
 */
const BasicInfoSection = ({ formData, handleInputChange, isEdit, isGroupForm }) => {
  // In group form context, we hide the basic info section entirely
  if (isGroupForm) {
    return null;
  }
  
  // Define field definitions for Basic Information section
  const fields = [
    {
      name: 'name',
      label: 'Name',
      type: 'text',
      defaultValue: '',
      required: true,
      helpText: isEdit ? 'Agent name cannot be changed after creation' : '',
      disabled: isEdit, // This will be handled in the component
    },
    {
      name: 'description',
      label: 'Description',
      type: 'textarea',
      defaultValue: '',
    },
    {
      name: 'identity_guidance',
      label: 'Identity Guidance',
      type: 'textarea',
      defaultValue: '',
    },
    {
      name: 'random_identity',
      label: 'Random Identity',
      type: 'checkbox',
      defaultValue: false,
    },
    {
      name: 'hud',
      label: 'HUD',
      type: 'checkbox',
      defaultValue: false,
    }
  ];

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    // For checkboxes, convert string 'true'/'false' to boolean
    if (name === 'random_identity' || name === 'hud') {
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
    <div id="basic-section">
      <h3 className="section-title">Basic Information</h3>
      
      <FormFieldDefinition
        fields={fields}
        values={formData}
        onChange={handleFieldChange}
        idPrefix="basic_"
      />
    </div>
  );
};

export default BasicInfoSection;
