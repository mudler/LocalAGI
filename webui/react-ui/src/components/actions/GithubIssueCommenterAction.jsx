import React from 'react';
import BaseAction from './BaseAction';

/**
 * GitHub Issue Commenter action component
 */
const GithubIssueCommenterAction = ({ index, onActionConfigChange, getConfigValue }) => {
  // Field definitions for GitHub Issue Commenter action
  const fields = [
    {
      name: 'token',
      label: 'GitHub Token',
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
      helpText: 'Owner of the repository',
      required: true,
    },
    {
      name: 'repository',
      label: 'Repository Name',
      type: 'text',
      defaultValue: '',
      placeholder: 'repository-name',
      helpText: 'Name of the repository',
      required: true,
    },
    {
      name: 'customActionName',
      label: 'Custom Action Name (Optional)',
      type: 'text',
      defaultValue: '',
      placeholder: 'comment_on_github_issue',
      helpText: 'Custom name for this action (optional)',
      required: false,
    },
  ];

  return (
    <BaseAction
      index={index}
      onActionConfigChange={onActionConfigChange}
      getConfigValue={getConfigValue}
      fields={fields}
    />
  );
};

export default GithubIssueCommenterAction;
