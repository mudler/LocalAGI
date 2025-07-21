import React from "react";
import FormFieldDefinition from "../common/FormFieldDefinition";

/**
 * Basic Information section of the agent form
 *
 * @param {Object} props Component props
 * @param {Object} props.formData Current form data values
 * @param {Function} props.handleInputChange Handler for input changes
 * @param {boolean} props.isEdit Whether the form is in edit mode
 * @param {boolean} props.isGroupForm Whether the form is for a group
 * @param {Object} props.metadata Field metadata from the backend
 */
const BasicInfoSection = ({
  formData,
  handleInputChange,
  isEdit,
  isGroupForm,
  metadata,
}) => {
  // In group form context, we hide the basic info section entirely
  if (isGroupForm) {
    return null;
  }

  // Get fields from metadata and apply any client-side overrides
  const fields =
    metadata?.BasicInfoSection?.map((field) => {
      // Special case for name field in edit mode
      if (field.name === "name" && isEdit) {
        return {
          ...field,
          disabled: true,
          helpText: "Agent name cannot be changed after creation",
        };
      }
      return field;
    }) || [];

  // Handle field value changes
  const handleFieldChange = (name, value) => {
    const field = fields.find((f) => f.name === name);
    if (field && field.type === "checkbox") {
      handleInputChange({
        target: {
          name,
          type: "checkbox",
          checked: value === "true",
        },
      });
    } else {
      handleInputChange({
        target: {
          name,
          value,
        },
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
