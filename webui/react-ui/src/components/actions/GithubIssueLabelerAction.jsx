import React from 'react';

/**
 * GitHub Issue Labeler action component
 */
const GithubIssueLabelerAction = ({ index, onActionConfigChange, getConfigValue }) => {
  return (
    <div className="github-issue-labeler-action">
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
        <label htmlFor={`availableLabels${index}`}>Available Labels</label>
        <input
          type="text"
          id={`availableLabels${index}`}
          value={getConfigValue('availableLabels', 'bug,enhancement')}
          onChange={(e) => onActionConfigChange('availableLabels', e.target.value)}
          className="form-control"
          placeholder="bug,enhancement,documentation"
        />
        <small className="form-text text-muted">Comma-separated list of available labels</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`customActionName${index}`}>Custom Action Name (Optional)</label>
        <input
          type="text"
          id={`customActionName${index}`}
          value={getConfigValue('customActionName', '')}
          onChange={(e) => onActionConfigChange('customActionName', e.target.value)}
          className="form-control"
          placeholder="add_label_to_issue"
        />
        <small className="form-text text-muted">Custom name for this action (optional)</small>
      </div>
    </div>
  );
};

export default GithubIssueLabelerAction;
