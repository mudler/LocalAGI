import React from 'react';

/**
 * Slack connector template
 */
const SlackConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  return (
    <div className="connector-template">
      <div className="form-group mb-3">
        <label htmlFor={`slackAppToken${index}`}>Slack App Token</label>
        <input
          type="text"
          id={`slackAppToken${index}`}
          value={getConfigValue(connector, 'appToken', '')}
          onChange={(e) => onConnectorConfigChange(index, 'appToken', e.target.value)}
          className="form-control"
          placeholder="xapp-..."
        />
        <small className="form-text text-muted">App-level token starting with xapp-</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`slackBotToken${index}`}>Slack Bot Token</label>
        <input
          type="text"
          id={`slackBotToken${index}`}
          value={getConfigValue(connector, 'botToken', '')}
          onChange={(e) => onConnectorConfigChange(index, 'botToken', e.target.value)}
          className="form-control"
          placeholder="xoxb-..."
        />
        <small className="form-text text-muted">Bot token starting with xoxb-</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`slackChannelID${index}`}>Slack Channel ID</label>
        <input
          type="text"
          id={`slackChannelID${index}`}
          value={getConfigValue(connector, 'channelID', '')}
          onChange={(e) => onConnectorConfigChange(index, 'channelID', e.target.value)}
          className="form-control"
          placeholder="C1234567890"
        />
        <small className="form-text text-muted">Optional: Specific channel ID to join</small>
      </div>

      <div className="form-group mb-3">
        <div className="form-check">
          <input
            type="checkbox"
            id={`slackAlwaysReply${index}`}
            checked={getConfigValue(connector, 'alwaysReply', '') === 'true'}
            onChange={(e) => onConnectorConfigChange(index, 'alwaysReply', e.target.checked ? 'true' : 'false')}
            className="form-check-input"
          />
          <label className="form-check-label" htmlFor={`slackAlwaysReply${index}`}>
            Always Reply
          </label>
          <small className="form-text text-muted d-block">If checked, the agent will reply to all messages in the channel</small>
        </div>
      </div>
    </div>
  );
};

export default SlackConnector;
