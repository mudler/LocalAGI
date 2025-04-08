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
  name,
  label,
  type = 'text',
  value,
  onChange,
  placeholder = '',
  helpText = '',
  options = [],
  required = false,
  min = 0,
  max = 2**31,
  step = 1,
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
                name={name}
                checked={value === true || value === 'true'}
                onChange={onChange}
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
              name={name}
              value={value || ''}
              onChange={onChange}
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
              name={name}
              value={value || ''}
              onChange={onChange}
              className="form-control"
              placeholder={placeholder}
              required={required}
              rows={5}
            />
            {helpText && <small className="form-text text-muted">{helpText}</small>}
          </>
        );
      case 'number':
        return (
          <>
            <label htmlFor={id}>{labelWithIndicator}</label>
            <input
              type="number"
              id={id}
              name={name}
              value={value || ''}
              onChange={onChange}
              className="form-control"
              placeholder={placeholder}
              required={required}
              min={min}
              max={max}
              step={step}
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
              name={name}
              value={value || ''}
              onChange={onChange}
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
