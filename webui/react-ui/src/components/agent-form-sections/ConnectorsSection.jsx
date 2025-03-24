import React from 'react';
import ConnectorForm from '../ConnectorForm';

/**
 * Connectors section of the agent form
 */
const ConnectorsSection = ({ 
  formData, 
  handleAddConnector, 
  handleRemoveConnector, 
  handleConnectorNameChange, 
  handleConnectorConfigChange 
}) => {
  return (
    <div id="connectors-section">
      <h3 className="section-title">Connectors</h3>
      <p className="section-description">
        Configure the connectors that this agent will use to communicate with external services.
      </p>
      
      <ConnectorForm 
        connectors={formData.connectors || []} 
        onAddConnector={handleAddConnector}
        onRemoveConnector={handleRemoveConnector}
        onConnectorNameChange={handleConnectorNameChange}
        onConnectorConfigChange={handleConnectorConfigChange}
      />
    </div>
  );
};

export default ConnectorsSection;
