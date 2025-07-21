import React from 'react';
import FormField from './FormField';

/**
 * Component that renders a form based on a field definition
 * 
 * @param {Object} props Component props
 * @param {Array} props.fields Array of field definitions
 * @param {Object} props.values Current values for the fields
 * @param {Function} props.onChange Handler for field value changes
 * @param {string} props.idPrefix Prefix for field IDs
 */
const FormFieldDefinition = ({
  fields,
  values,
  onChange,
  idPrefix = '',
}) => {
  // Ensure values is an object
  const safeValues = values || {};

  return (
    <div className="form-fields">
      {fields.map((field) => (
        <div key={field.name} style={{ marginBottom: '16px' }}>
          <FormField
            id={`${idPrefix}${field.name}`}
            name={field.name}
            label={field.label}
            type={field.type}
            value={safeValues[field.name] != null ? safeValues[field.name] : field.defaultValue}
            onChange={onChange}
            placeholder={field.placeholder || ''}
            helpText={field.helpText || ''}
            options={field.options || []}
            required={field.required || false}
            min={field.min || 0}
            max={field.max || 2**31}
            step={field.step || 1}
          />
        </div>
      ))}
    </div>
  );
};

export default FormFieldDefinition;
