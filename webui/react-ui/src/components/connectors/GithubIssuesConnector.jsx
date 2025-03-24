import React from 'react';

/**
 * GitHub Issues connector template
 */
const GithubIssuesConnector = ({ connector, index, onConnectorConfigChange, getConfigValue }) => {
  return (
    <div className="connector-template">
      <div className="form-group mb-3">
        <label htmlFor={`githubToken${index}`}>GitHub Personal Access Token</label>
        <input
          type="text"
          id={`githubToken${index}`}
          value={getConfigValue(connector, 'token', '')}
          onChange={(e) => onConnectorConfigChange(index, 'token', e.target.value)}
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
          value={getConfigValue(connector, 'owner', '')}
          onChange={(e) => onConnectorConfigChange(index, 'owner', e.target.value)}
          className="form-control"
          placeholder="username or organization"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`githubRepo${index}`}>Repository Name</label>
        <input
          type="text"
          id={`githubRepo${index}`}
          value={getConfigValue(connector, 'repository', '')}
          onChange={(e) => onConnectorConfigChange(index, 'repository', e.target.value)}
          className="form-control"
          placeholder="repository-name"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`replyIfNoReplies${index}`}>Reply Behavior</label>
        <select
          id={`replyIfNoReplies${index}`}
          value={getConfigValue(connector, 'replyIfNoReplies', 'false')}
          onChange={(e) => onConnectorConfigChange(index, 'replyIfNoReplies', e.target.value)}
          className="form-control"
        >
          <option value="false">Reply to all issues</option>
          <option value="true">Only reply to issues with no comments</option>
        </select>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`pollInterval${index}`}>Poll Interval</label>
        <input
          type="text"
          id={`pollInterval${index}`}
          value={getConfigValue(connector, 'pollInterval', '10m')}
          onChange={(e) => onConnectorConfigChange(index, 'pollInterval', e.target.value)}
          className="form-control"
          placeholder="10m"
        />
        <small className="form-text text-muted">How often to check for new issues (e.g., 10m, 1h)</small>
      </div>
    </div>
  );
};

export default GithubIssuesConnector;
