import React from 'react';
import BaseAction from './BaseAction';

/**
 * GitHub Repository action component for repository-related actions
 * Used for:
 * - github-repository-get-content
 * - github-repository-create-or-update-content
 * - github-readme
 */
const GithubRepositoryAction = ({ index, onActionConfigChange, getConfigValue }) => {
  // Field definitions for GitHub Repository action
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
      name: 'customActionName',
      label: 'Custom Action Name (Optional)',
      type: 'text',
      defaultValue: '',
      placeholder: 'github_repo_action',
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

export default GithubRepositoryAction;
