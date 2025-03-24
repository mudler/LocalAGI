import React from 'react';

/**
 * Telegram connector template
 */
const TelegramConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  return (
    <div className="connector-template">
      <div className="form-group mb-3">
        <label htmlFor={`telegramToken${index}`}>Telegram Bot Token</label>
        <input
          type="text"
          id={`telegramToken${index}`}
          value={getConfigValue(connector, 'token', '')}
          onChange={(e) => onConnectorConfigChange(index, 'token', e.target.value)}
          className="form-control"
          placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
        />
        <small className="form-text text-muted">Get this from @BotFather on Telegram</small>
      </div>
    </div>
  );
};

export default TelegramConnector;
