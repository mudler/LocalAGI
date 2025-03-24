import { useState } from 'react';

// Import connector components
import TelegramConnector from './connectors/TelegramConnector';
import SlackConnector from './connectors/SlackConnector';
import DiscordConnector from './connectors/DiscordConnector';
import GithubIssuesConnector from './connectors/GithubIssuesConnector';
import GithubPRsConnector from './connectors/GithubPRsConnector';
import IRCConnector from './connectors/IRCConnector';
import TwitterConnector from './connectors/TwitterConnector';
import FallbackConnector from './connectors/FallbackConnector';

/**
 * ConnectorForm component
 * Provides specific form templates for different connector types
 */
function ConnectorForm({ 
  connectors = [], 
  onAddConnector, 
  onRemoveConnector, 
  onConnectorNameChange, 
  onConnectorConfigChange 
}) {
  const [newConfigKey, setNewConfigKey] = useState('');

  // Render a specific connector form based on its type
  const renderConnectorForm = (connector, index) => {
    // Ensure connector is an object with expected properties
    const safeConnector = connector || {};
    
    return (
      <div key={index} className="connector-item mb-4">
        <div className="connector-header">
          <h4>Connector #{index + 1}</h4>
          <button 
            type="button" 
            className="remove-btn"
            onClick={() => onRemoveConnector(index)}
          >
            <i className="fas fa-times"></i>
          </button>
        </div>
        
        <div className="connector-type mb-3">
          <label htmlFor={`connectorName${index}`}>Connector Type</label>
          <select
            id={`connectorName${index}`}
            value={safeConnector.type || ''}
            onChange={(e) => onConnectorNameChange(index, e.target.value)}
            className="form-control"
          >
            <option value="">Select a connector type</option>
            <option value="telegram">Telegram</option>
            <option value="slack">Slack</option>
            <option value="discord">Discord</option>
            <option value="github-issues">GitHub Issues</option>
            <option value="github-prs">GitHub PRs</option>
            <option value="irc">IRC</option>
            <option value="twitter">Twitter</option>
            <option value="custom">Custom</option>
          </select>
        </div>
        
        {/* Render specific connector template based on type */}
        {renderConnectorTemplate(safeConnector, index)}
      </div>
    );
  };

  // Get the appropriate form template based on connector type
  const renderConnectorTemplate = (connector, index) => {
    // Check if connector.type exists, if not use empty string to avoid errors
    const connectorType = (connector.type || '').toLowerCase();
    
    // Common props for all connector components
    const connectorProps = {
      connector,
      index,
      onConnectorConfigChange,
      getConfigValue
    };
    
    switch (connectorType) {
      case 'telegram':
        return <TelegramConnector {...connectorProps} />;
      case 'slack':
        return <SlackConnector {...connectorProps} />;
      case 'discord':
        return <DiscordConnector {...connectorProps} />;
      case 'github-issues':
        return <GithubIssuesConnector {...connectorProps} />;
      case 'github-prs':
        return <GithubPRsConnector {...connectorProps} />;
      case 'irc':
        return <IRCConnector {...connectorProps} />;
      case 'twitter':
        return <TwitterConnector {...connectorProps} />;
      default:
        return <FallbackConnector {...connectorProps} />;
    }
  };

  // Helper function to safely get config values
  const getConfigValue = (connector, key, defaultValue = '') => {
    if (!connector || !connector.config) return defaultValue;
    
    // If config is a string (JSON), try to parse it
    let config = connector.config;
    if (typeof config === 'string') {
      try {
        config = JSON.parse(config);
      } catch (err) {
        console.error('Error parsing config:', err);
        return defaultValue;
      }
    }
    
    return config[key] !== undefined ? config[key] : defaultValue;
  };

  return (
    <div className="connectors-container">
      {connectors && connectors.map((connector, index) => (
        renderConnectorForm(connector, index)
      ))}
      
      <button 
        type="button" 
        className="add-btn"
        onClick={onAddConnector}
      >
        <i className="fas fa-plus"></i> Add Connector
      </button>
    </div>
  );
}

export default ConnectorForm;
