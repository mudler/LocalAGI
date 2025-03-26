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
  onConnectorNameChange, 
  onConnectorConfigChange,
  fieldGroups = []
}) {
  // Debug logging
  console.log('ConnectorForm:', { connectors, fieldGroups });
  
  // Handle connector change
  const handleConnectorChange = (index, updatedConnector) => {
    console.log('Connector change:', { index, updatedConnector });
    if (updatedConnector.type !== connectors[index].type) {
      onConnectorNameChange(index, updatedConnector.type);
    } else {
      onConnectorConfigChange(index, updatedConnector.config);
    }
  };

  // Handle adding a new connector
  const handleAddConnector = () => {
    console.log('Adding new connector');
    onAddConnector();
  };

  return (
    <ConfigForm
      items={connectors}
      fieldGroups={fieldGroups}
      onChange={handleConnectorChange}
      onRemove={onRemoveConnector}
      onAdd={handleAddConnector}
      itemType="connector"
      typeField="type"
      addButtonText="Add Connector"
    />
  );
}

export default ConnectorForm;
