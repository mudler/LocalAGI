import React from 'react';

/**
 * Reusable form field component that handles different input types
 * 
 * @param {Object} props Component props
 * @param {string} props.id Unique identifier for the input
 * @param {string} props.label Label text for the field
 * @param {string} props.type Input type (text, checkbox, select, textarea)
 * @param {string|boolean} props.value Current value of the field
 * @param {Function} props.onChange Handler for value changes
 * @param {string} props.placeholder Placeholder text
 * @param {string} props.helpText Help text to display below the field
 * @param {Array} props.options Options for select inputs
 * @param {boolean} props.required Whether the field is required
 */
const FormField = ({
  id,
  label,
  type = 'text',
  value,
  onChange,
  placeholder = '',
  helpText = '',
  options = [],
  required = false,
}) => {
  // Create label with required indicator
  const labelWithIndicator = required ? (
    <>{label} <span style={{ color: 'var(--danger)' }}>*</span></>
  ) : (
    label
  );

  // Render different input types
  const renderInput = () => {
    switch (type) {
      case 'checkbox':
        return (
          <div className="form-check">
            <label className="checkbox-label" htmlFor={id}>
              <input
                type="checkbox"
                id={id}
                checked={value === true || value === 'true'}
                onChange={(e) => onChange(e.target.checked ? 'true' : 'false')}
              />
              {labelWithIndicator}
            </label>
            {helpText && <small className="form-text text-muted d-block">{helpText}</small>}
          </div>
        );
      case 'select':
        return (
          <>
            <label htmlFor={id}>{labelWithIndicator}</label>
            <select
              id={id}
              value={value || ''}
              onChange={(e) => onChange(e.target.value)}
              className="form-control"
              required={required}
            >
              {options.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            {helpText && <small className="form-text text-muted">{helpText}</small>}
          </>
        );
      case 'textarea':
        return (
          <>
            <label htmlFor={id}>{labelWithIndicator}</label>
            <textarea
              id={id}
              value={value || ''}
              onChange={(e) => onChange(e.target.value)}
              className="form-control"
              placeholder={placeholder}
              required={required}
              rows={5}
            />
            {helpText && <small className="form-text text-muted">{helpText}</small>}
          </>
        );
      default:
        return (
          <>
            <label htmlFor={id}>{labelWithIndicator}</label>
            <input
              type={type}
              id={id}
              value={value || ''}
              onChange={(e) => onChange(e.target.value)}
              className="form-control"
              placeholder={placeholder}
              required={required}
            />
            {helpText && <small className="form-text text-muted">{helpText}</small>}
          </>
        );
    }
  };

  return (
    <div className="form-group mb-3">
      {renderInput()}
    </div>
  );
};

export default FormField;
