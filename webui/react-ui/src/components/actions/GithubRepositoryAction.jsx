import React from 'react';

/**
 * GitHub Repository action component for repository-related actions
 * Used for:
 * - github-repository-get-content
 * - github-repository-create-or-update-content
 * - github-readme
 */
const GithubRepositoryAction = ({ index, onActionConfigChange, getConfigValue }) => {
  return (
    <div className="github-repository-action">
      <div className="form-group mb-3">
        <label htmlFor={`githubToken${index}`}>GitHub Token</label>
        <input
          type="text"
          id={`githubToken${index}`}
          value={getConfigValue('token', '')}
          onChange={(e) => onActionConfigChange('token', e.target.value)}
          className="form-control"
          placeholder="ghp_..."
        />
        <small className="form-text text-muted">Personal access token with repo scope</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`githubOwner${index}`}>Repository Owner</label>
        <input
          type="text"
          id={`githubOwner${index}`}
          value={getConfigValue('owner', '')}
          onChange={(e) => onActionConfigChange('owner', e.target.value)}
          className="form-control"
          placeholder="username or organization"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`githubRepo${index}`}>Repository Name</label>
        <input
          type="text"
          id={`githubRepo${index}`}
          value={getConfigValue('repository', '')}
          onChange={(e) => onActionConfigChange('repository', e.target.value)}
          className="form-control"
          placeholder="repository-name"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`customActionName${index}`}>Custom Action Name (Optional)</label>
        <input
          type="text"
          id={`customActionName${index}`}
          value={getConfigValue('customActionName', '')}
          onChange={(e) => onActionConfigChange('customActionName', e.target.value)}
          className="form-control"
          placeholder="github_repo_action"
        />
        <small className="form-text text-muted">Custom name for this action (optional)</small>
      </div>
    </div>
  );
};

export default GithubRepositoryAction;
