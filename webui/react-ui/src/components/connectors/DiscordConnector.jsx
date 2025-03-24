import React from 'react';

/**
 * Discord connector template
 */
const DiscordConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  return (
    <div className="connector-template">
      <div className="form-group mb-3">
        <label htmlFor={`discordToken${index}`}>Discord Bot Token</label>
        <input
          type="text"
          id={`discordToken${index}`}
          value={getConfigValue(connector, 'token', '')}
          onChange={(e) => onConnectorConfigChange(index, 'token', e.target.value)}
          className="form-control"
          placeholder="Bot token from Discord Developer Portal"
        />
        <small className="form-text text-muted">Get this from the Discord Developer Portal</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`discordDefaultChannel${index}`}>Default Channel</label>
        <input
          type="text"
          id={`discordDefaultChannel${index}`}
          value={getConfigValue(connector, 'defaultChannel', '')}
          onChange={(e) => onConnectorConfigChange(index, 'defaultChannel', e.target.value)}
          className="form-control"
          placeholder="123456789012345678"
        />
        <small className="form-text text-muted">Channel ID to always answer even if not mentioned</small>
      </div>
    </div>
  );
};

export default DiscordConnector;
