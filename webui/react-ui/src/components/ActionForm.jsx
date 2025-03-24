import React from 'react';
import FallbackAction from './actions/FallbackAction';
import GithubIssueLabelerAction from './actions/GithubIssueLabelerAction';
import GithubIssueOpenerAction from './actions/GithubIssueOpenerAction';
import GithubIssueCloserAction from './actions/GithubIssueCloserAction';
import GithubIssueCommenterAction from './actions/GithubIssueCommenterAction';
import GithubRepositoryAction from './actions/GithubRepositoryAction';
import TwitterPostAction from './actions/TwitterPostAction';
import SendMailAction from './actions/SendMailAction';

/**
 * ActionForm component for configuring an action
 */
const ActionForm = ({ actions = [], onChange, onRemove, onAdd }) => {
  // Available action types
  const actionTypes = [
    { value: '', label: 'Select an action type' },
    { value: 'github-issue-labeler', label: 'GitHub Issue Labeler' },
    { value: 'github-issue-opener', label: 'GitHub Issue Opener' },
    { value: 'github-issue-closer', label: 'GitHub Issue Closer' },
    { value: 'github-issue-commenter', label: 'GitHub Issue Commenter' },
    { value: 'github-repository-get-content', label: 'GitHub Repository Get Content' },
    { value: 'github-repository-create-or-update-content', label: 'GitHub Repository Create/Update Content' },
    { value: 'github-readme', label: 'GitHub Readme' },
    { value: 'twitter-post', label: 'Twitter Post' },
    { value: 'send-mail', label: 'Send Email' },
    { value: 'search', label: 'Search' },
    { value: 'github-issue-searcher', label: 'GitHub Issue Searcher' },
    { value: 'github-issue-reader', label: 'GitHub Issue Reader' },
    { value: 'scraper', label: 'Web Scraper' },
    { value: 'wikipedia', label: 'Wikipedia' },
    { value: 'browse', label: 'Browse' },
    { value: 'generate_image', label: 'Generate Image' },
    { value: 'counter', label: 'Counter' },
    { value: 'call_agents', label: 'Call Agents' },
    { value: 'shell-command', label: 'Shell Command' },
    { value: 'custom', label: 'Custom' }
  ];

  // Parse the config JSON string to an object
  const parseConfig = (action) => {
    if (!action || !action.config) return {};
    
    try {
      return JSON.parse(action.config || '{}');
    } catch (error) {
      console.error('Error parsing action config:', error);
      return {};
    }
  };

  // Get a value from the config object
  const getConfigValue = (action, key, defaultValue = '') => {
    const config = parseConfig(action);
    return config[key] !== undefined ? config[key] : defaultValue;
  };

  // Update a value in the config object
  const onActionConfigChange = (index, key, value) => {
    const action = actions[index];
    const config = parseConfig(action);
    config[key] = value;
    
    onChange(index, {
      ...action,
      config: JSON.stringify(config)
    });
  };

  // Handle action type change
  const handleActionTypeChange = (index, value) => {
    const action = actions[index];
    onChange(index, {
      ...action,
      name: value
    });
  };

  // Render the appropriate action component based on the action type
  const renderActionComponent = (action, index) => {
    // Common props for all action components
    const actionProps = {
      index,
      onActionConfigChange: (key, value) => onActionConfigChange(index, key, value),
      getConfigValue: (key, defaultValue) => getConfigValue(action, key, defaultValue)
    };
    
    switch (action.name) {
      case 'github-issue-labeler':
        return <GithubIssueLabelerAction {...actionProps} />;
      case 'github-issue-opener':
        return <GithubIssueOpenerAction {...actionProps} />;
      case 'github-issue-closer':
        return <GithubIssueCloserAction {...actionProps} />;
      case 'github-issue-commenter':
        return <GithubIssueCommenterAction {...actionProps} />;
      case 'github-repository-get-content':
      case 'github-repository-create-or-update-content':
      case 'github-readme':
        return <GithubRepositoryAction {...actionProps} />;
      case 'twitter-post':
        return <TwitterPostAction {...actionProps} />;
      case 'send-mail':
        return <SendMailAction {...actionProps} />;
      case 'generate_image':
        return (
          <div className="generate-image-action">
            <div className="form-group mb-3">
              <label htmlFor={`apiKey${index}`}>OpenAI API Key</label>
              <input
                type="text"
                id={`apiKey${index}`}
                value={getConfigValue(action, 'apiKey', '')}
                onChange={(e) => onActionConfigChange(index, 'apiKey', e.target.value)}
                className="form-control"
                placeholder="sk-..."
              />
            </div>
            
            <div className="form-group mb-3">
              <label htmlFor={`apiURL${index}`}>API URL (Optional)</label>
              <input
                type="text"
                id={`apiURL${index}`}
                value={getConfigValue(action, 'apiURL', 'https://api.openai.com/v1')}
                onChange={(e) => onActionConfigChange(index, 'apiURL', e.target.value)}
                className="form-control"
                placeholder="https://api.openai.com/v1"
              />
            </div>
            
            <div className="form-group mb-3">
              <label htmlFor={`model${index}`}>Model</label>
              <input
                type="text"
                id={`model${index}`}
                value={getConfigValue(action, 'model', 'dall-e-3')}
                onChange={(e) => onActionConfigChange(index, 'model', e.target.value)}
                className="form-control"
                placeholder="dall-e-3"
              />
              <small className="form-text text-muted">Image generation model (e.g., dall-e-3)</small>
            </div>
          </div>
        );
      default:
        return <FallbackAction {...actionProps} />;
    }
  };

  // Render a specific action form
  const renderActionForm = (action, index) => {
    // Ensure action is an object with expected properties
    const safeAction = action || {};
    
    return (
      <div key={index} className="connector-item mb-4">
        <div className="connector-header">
          <h4>Action #{index + 1}</h4>
          <button 
            type="button" 
            className="remove-btn"
            onClick={() => onRemove(index)}
          >
            <i className="fas fa-times"></i>
          </button>
        </div>
        
        <div className="connector-type mb-3">
          <label htmlFor={`actionType${index}`}>Action Type</label>
          <select
            id={`actionType${index}`}
            value={safeAction.name || ''}
            onChange={(e) => handleActionTypeChange(index, e.target.value)}
            className="form-control"
          >
            {actionTypes.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>
        </div>
        
        {/* Render specific action template based on type */}
        {renderActionComponent(safeAction, index)}
      </div>
    );
  };

  return (
    <div className="connectors-container">
      {actions && actions.map((action, index) => (
        renderActionForm(action, index)
      ))}
      
      <button 
        type="button" 
        className="add-btn"
        onClick={onAdd}
      >
        <i className="fas fa-plus"></i> Add Action
      </button>
    </div>
  );
};

export default ActionForm;
