import React from 'react';
import ConfigForm from './ConfigForm';

/**
 * ActionForm component for configuring an action
 * Renders action configuration forms based on field group metadata
 */
const ActionForm = ({ actions = [], onChange, onRemove, onAdd, onPlay, fieldGroups = [] }) => {
  const handleActionChange = (index, updatedAction) => {
    onChange(index, updatedAction);
  };
  
  return (
    <ConfigForm
      items={actions}
      fieldGroups={fieldGroups}
      onChange={handleActionChange}
      onRemove={onRemove}
      onAdd={onAdd}
      onPlay={onPlay}
      itemType="action"
      typeField="name"
      addButtonText="Add Action"
    />
  );
};

export default ActionForm;
