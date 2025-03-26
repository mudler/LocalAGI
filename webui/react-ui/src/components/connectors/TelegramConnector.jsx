import React from 'react';
import BaseConnector from './BaseConnector';

/**
 * Telegram connector template
 */
const TelegramConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  // Field definitions for Telegram connector
  const fields = [
    {
      name: 'token',
      label: 'Telegram Bot Token',
      type: 'text',
      defaultValue: '',
      placeholder: '123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11',
      helpText: 'Get this from @BotFather on Telegram',
      required: true,
    },
  ];

  return (
    <BaseConnector
      connector={connector}
      index={index}
      onConnectorConfigChange={onConnectorConfigChange}
      getConfigValue={getConfigValue}
      fields={fields}
    />
  );
};

export default TelegramConnector;
