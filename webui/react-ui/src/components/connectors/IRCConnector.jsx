import React from 'react';

/**
 * IRC connector template
 */
const IRCConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  return (
    <div className="connector-template">
      <div className="form-group mb-3">
        <label htmlFor={`ircServer${index}`}>IRC Server</label>
        <input
          type="text"
          id={`ircServer${index}`}
          value={getConfigValue(connector, 'server', '')}
          onChange={(e) => onConnectorConfigChange(index, 'server', e.target.value)}
          className="form-control"
          placeholder="irc.libera.chat"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`ircPort${index}`}>Port</label>
        <input
          type="text"
          id={`ircPort${index}`}
          value={getConfigValue(connector, 'port', '6667')}
          onChange={(e) => onConnectorConfigChange(index, 'port', e.target.value)}
          className="form-control"
          placeholder="6667"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`ircNick${index}`}>Nickname</label>
        <input
          type="text"
          id={`ircNick${index}`}
          value={getConfigValue(connector, 'nickname', '')}
          onChange={(e) => onConnectorConfigChange(index, 'nickname', e.target.value)}
          className="form-control"
          placeholder="MyAgentBot"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`ircChannels${index}`}>Channel</label>
        <input
          type="text"
          id={`ircChannels${index}`}
          value={getConfigValue(connector, 'channel', '')}
          onChange={(e) => onConnectorConfigChange(index, 'channel', e.target.value)}
          className="form-control"
          placeholder="#channel1"
        />
        <small className="form-text text-muted">Channel to join</small>
      </div>

      <div className="form-group mb-3">
        <div className="form-check">
          <label className="checkbox-label" htmlFor={`ircAlwaysReply${index}`}>
            <input
              type="checkbox"
              id={`ircAlwaysReply${index}`}
              checked={getConfigValue(connector, 'alwaysReply', '') === 'true'}
              onChange={(e) => onConnectorConfigChange(index, 'alwaysReply', e.target.checked ? 'true' : 'false')}
          />
            Always Reply
          </label>
          <small className="form-text text-muted d-block">If checked, the agent will reply to all messages in the channel</small>
        </div>
      </div>
    </div>
  );
};

export default IRCConnector;
