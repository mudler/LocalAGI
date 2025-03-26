import React from 'react';
import BaseConnector from './BaseConnector';

/**
 * Discord connector template
 */
const DiscordConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  // Field definitions for Discord connector
  const fields = [
    {
      name: 'token',
      label: 'Discord Bot Token',
      type: 'text',
      defaultValue: '',
      placeholder: 'Bot token from Discord Developer Portal',
      helpText: 'Get this from the Discord Developer Portal',
      required: true,
    },
    {
      name: 'defaultChannel',
      label: 'Default Channel',
      type: 'text',
      defaultValue: '',
      placeholder: '123456789012345678',
      helpText: 'Channel ID to always answer even if not mentioned',
      required: false,
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

export default DiscordConnector;
