import React from 'react';
import ConfigForm from './ConfigForm';

/**
 * ConnectorForm component
 * Renders connector configuration forms based on field group metadata
 */
function ConnectorForm({ 
  connectors = [], 
  onAddConnector, 
  onRemoveConnector, 
  onChange,
  fieldGroups = []
}) {
  return (
    <ConfigForm
      items={connectors}
      fieldGroups={fieldGroups}
      onChange={onChange}
      onRemove={onRemoveConnector}
      onAdd={onAddConnector}
      itemType="connector"
      typeField="type"
      addButtonText="Add Connector"
    />
  );
}

export default ConnectorForm;
