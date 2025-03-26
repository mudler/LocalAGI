import React from 'react';
import BaseConnector from './BaseConnector';

/**
 * Twitter connector template
 */
const TwitterConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  // Field definitions for Twitter connector
  const fields = [
    {
      name: 'apiKey',
      label: 'API Key',
      type: 'text',
      defaultValue: '',
      placeholder: 'Twitter API Key',
      helpText: '',
      required: true,
    },
    {
      name: 'apiSecret',
      label: 'API Secret',
      type: 'password',
      defaultValue: '',
      placeholder: 'Twitter API Secret',
      helpText: '',
      required: true,
    },
    {
      name: 'accessToken',
      label: 'Access Token',
      type: 'text',
      defaultValue: '',
      placeholder: 'Twitter Access Token',
      helpText: '',
      required: true,
    },
    {
      name: 'accessSecret',
      label: 'Access Token Secret',
      type: 'password',
      defaultValue: '',
      placeholder: 'Twitter Access Token Secret',
      helpText: '',
      required: true,
    },
    {
      name: 'bearerToken',
      label: 'Bearer Token',
      type: 'password',
      defaultValue: '',
      placeholder: 'Twitter Bearer Token',
      helpText: '',
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

export default TwitterConnector;
