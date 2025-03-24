import React from 'react';

/**
 * Twitter connector template
 */
const TwitterConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  return (
    <div className="connector-template">
      <div className="form-group mb-3">
        <label htmlFor={`twitterApiKey${index}`}>API Key</label>
        <input
          type="text"
          id={`twitterApiKey${index}`}
          value={getConfigValue(connector, 'apiKey', '')}
          onChange={(e) => onConnectorConfigChange(index, 'apiKey', e.target.value)}
          className="form-control"
          placeholder="Twitter API Key"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`twitterApiSecret${index}`}>API Secret</label>
        <input
          type="password"
          id={`twitterApiSecret${index}`}
          value={getConfigValue(connector, 'apiSecret', '')}
          onChange={(e) => onConnectorConfigChange(index, 'apiSecret', e.target.value)}
          className="form-control"
          placeholder="Twitter API Secret"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`twitterAccessToken${index}`}>Access Token</label>
        <input
          type="text"
          id={`twitterAccessToken${index}`}
          value={getConfigValue(connector, 'accessToken', '')}
          onChange={(e) => onConnectorConfigChange(index, 'accessToken', e.target.value)}
          className="form-control"
          placeholder="Twitter Access Token"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`twitterAccessSecret${index}`}>Access Token Secret</label>
        <input
          type="password"
          id={`twitterAccessSecret${index}`}
          value={getConfigValue(connector, 'accessSecret', '')}
          onChange={(e) => onConnectorConfigChange(index, 'accessSecret', e.target.value)}
          className="form-control"
          placeholder="Twitter Access Token Secret"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`twitterBearerToken${index}`}>Bearer Token</label>
        <input
          type="password"
          id={`twitterBearerToken${index}`}
          value={getConfigValue(connector, 'bearerToken', '')}
          onChange={(e) => onConnectorConfigChange(index, 'bearerToken', e.target.value)}
          className="form-control"
          placeholder="Twitter Bearer Token"
        />
      </div>
    </div>
  );
};

export default TwitterConnector;
