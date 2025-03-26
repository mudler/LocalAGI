import React from 'react';
import BaseConnector from './BaseConnector';

/**
 * IRC connector template
 */
const IRCConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  // Field definitions for IRC connector
  const fields = [
    {
      name: 'server',
      label: 'IRC Server',
      type: 'text',
      defaultValue: '',
      placeholder: 'irc.libera.chat',
      helpText: 'IRC server address',
      required: true,
    },
    {
      name: 'port',
      label: 'Port',
      type: 'text',
      defaultValue: '6667',
      placeholder: '6667',
      helpText: 'IRC server port',
      required: true,
    },
    {
      name: 'nickname',
      label: 'Nickname',
      type: 'text',
      defaultValue: '',
      placeholder: 'MyAgentBot',
      helpText: 'Bot nickname',
      required: true,
    },
    {
      name: 'channel',
      label: 'Channel',
      type: 'text',
      defaultValue: '',
      placeholder: '#channel1',
      helpText: 'Channel to join',
      required: true,
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

export default IRCConnector;
