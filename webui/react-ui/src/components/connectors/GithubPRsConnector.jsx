import React from 'react';
import BaseConnector from './BaseConnector';

/**
 * GitHub PRs connector template
 */
const GithubPRsConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  // Field definitions for GitHub PRs connector
  const fields = [
    {
      name: 'token',
      label: 'GitHub Personal Access Token',
      type: 'text',
      defaultValue: '',
      placeholder: 'ghp_...',
      helpText: 'Personal access token with repo scope',
      required: true,
    },
    {
      name: 'owner',
      label: 'Repository Owner',
      type: 'text',
      defaultValue: '',
      placeholder: 'username or organization',
      helpText: '',
      required: true,
    },
    {
      name: 'repository',
      label: 'Repository Name',
      type: 'text',
      defaultValue: '',
      placeholder: 'repository-name',
      helpText: '',
      required: true,
    },
    {
      name: 'replyIfNoReplies',
      label: 'Reply Behavior',
      type: 'select',
      defaultValue: 'false',
      options: [
        { value: 'false', label: 'Reply to all PRs' },
        { value: 'true', label: 'Only reply to PRs with no comments' },
      ],
      helpText: '',
      required: false,
    },
    {
      name: 'pollInterval',
      label: 'Poll Interval',
      type: 'text',
      defaultValue: '10m',
      placeholder: '10m',
      helpText: 'How often to check for new PRs (e.g., 10m, 1h)',
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

export default GithubPRsConnector;
