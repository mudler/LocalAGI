import React, { useState } from 'react';
import FormFieldDefinition from './common/FormFieldDefinition';

/**
 * ConfigForm - A generic component for handling configuration forms based on FieldGroups
 * 
 * @param {Object} props Component props
 * @param {Array} props.items - Array of configuration items (actions, connectors, etc.)
 * @param {Array} props.fieldGroups - Array of FieldGroup objects that define the available types and their fields
 * @param {Function} props.onChange - Callback when an item changes
 * @param {Function} props.onRemove - Callback when an item is removed
 * @param {Function} props.onAdd - Callback when a new item is added
 * @param {String} props.itemType - Type of items being configured ('action', 'connector', etc.)
 * @param {String} props.typeField - The field name that determines the item's type (e.g., 'name' for actions, 'type' for connectors)
 * @param {String} props.addButtonText - Text for the add button
 * @param {String} props.saveAllFieldsAsString - Whether to save all fields as string or the appropriate JSON type
 */
const ConfigForm = ({ 
  items = [], 
  fieldGroups = [],
  onChange, 
  onRemove, 
  onAdd,
  itemType = 'item',
  typeField = 'type',
  addButtonText = 'Add Item',
  saveAllFieldsAsString = true,
}) => {
  // Generate options from fieldGroups
  const typeOptions = [
    { value: '', label: `Select a ${itemType} type` },
    ...fieldGroups.map(group => ({
      value: group.name,
      label: group.label
    }))
  ];
  
  // Parse the config JSON string to an object
  const parseConfig = (item) => {
    if (!item || !item.config) return {};
    
    try {
      return typeof item.config === 'string' 
        ? JSON.parse(item.config || '{}') 
        : item.config;
    } catch (error) {
      console.error(`Error parsing ${itemType} config:`, error);
      return {};
    }
  };

  // Handle item type change
  const handleTypeChange = (index, value) => {
    const item = items[index];
    onChange(index, {
      ...item,
      [typeField]: value
    });
  };

  // Handle config field change
  const handleConfigChange = (index, e) => {
    const { name: key, value, type, checked } = e.target;
    const item = items[index];
    const config = parseConfig(item);
    
    if (type === 'number' && !saveAllFieldsAsString)
      config[key] = Number(value);
    else if (type === 'checkbox')
      config[key] = saveAllFieldsAsString ? (checked ? 'true' : 'false') : checked;
    else
      config[key] = value;
    
    onChange(index, {
      ...item,
      config: JSON.stringify(config)
    });
  };

  // Render a specific item form
  const renderItemForm = (item, index) => {
    // Ensure item is an object with expected properties
    const safeItem = item || {};
    const itemTypeName = safeItem[typeField] || '';
    
    // Find the field group that matches this item's type
    const fieldGroup = fieldGroups.find(group => group.name === itemTypeName);
    const itemTypeLabel = itemType.charAt(0).toUpperCase() + itemType.slice(1).replace('_', ' ');
    
    return (
      <div key={index} className="config-item mb-4 card">
        <div className="config-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h4 style={{ margin: 0 }}>{itemTypeLabel} #{index + 1}</h4>
          <button 
            type="button" 
            className="action-btn delete-btn"
            onClick={() => onRemove(index)}
          >
            <i className="fas fa-times"></i>
          </button>
        </div>
        
        <div className="config-type mb-3">
          <label htmlFor={`${itemType}Type${index}`}>{itemTypeLabel} Type</label>
          <select
            id={`${itemType}Type${index}`}
            value={safeItem[typeField] || ''}
            onChange={(e) => handleTypeChange(index, e.target.value)}
            className="form-control"
          >
            {typeOptions.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>
        </div>
        
        {/* Render fields based on the selected type */}
        {fieldGroup && fieldGroup.fields && (
          <FormFieldDefinition
            fields={fieldGroup.fields}
            values={parseConfig(item)}
            onChange={(e) => handleConfigChange(index, e)}
            idPrefix={`${itemType}-${index}-`}
          />
        )}
      </div>
    );
  };

  return (
    <div className="config-container">
      {items && items.map((item, index) => (
        renderItemForm(item, index)
      ))}
      
      <button 
        type="button" 
        className="action-btn"
        onClick={onAdd}
      >
        <i className="fas fa-plus"></i> {addButtonText}
      </button>
    </div>
  );
};

export default ConfigForm;
