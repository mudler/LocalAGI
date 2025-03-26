import React from 'react';
import BaseConnector from './BaseConnector';

/**
 * Slack connector template
 */
const SlackConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  // Field definitions for Slack connector
  const fields = [
    {
      name: 'appToken',
      label: 'Slack App Token',
      type: 'text',
      defaultValue: '',
      placeholder: 'xapp-...',
      helpText: 'App-level token starting with xapp-',
      required: true,
    },
    {
      name: 'botToken',
      label: 'Slack Bot Token',
      type: 'text',
      defaultValue: '',
      placeholder: 'xoxb-...',
      helpText: 'Bot token starting with xoxb-',
      required: true,
    },
    {
      name: 'channelID',
      label: 'Slack Channel ID',
      type: 'text',
      defaultValue: '',
      placeholder: 'C1234567890',
      helpText: 'Optional: Specific channel ID to join',
      required: false,
    },
    {
      name: 'alwaysReply',
      label: 'Always Reply',
      type: 'checkbox',
      defaultValue: 'false',
      helpText: 'If checked, the agent will reply to all messages in the channel',
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

export default SlackConnector;
