import React from 'react';

/**
 * Twitter Post action component
 */
const TwitterPostAction = ({ index, onActionConfigChange, getConfigValue }) => {
  return (
    <div className="twitter-post-action">
      <div className="form-group mb-3">
        <label htmlFor={`twitterToken${index}`}>Twitter API Token</label>
        <input
          type="text"
          id={`twitterToken${index}`}
          value={getConfigValue('token', '')}
          onChange={(e) => onActionConfigChange('token', e.target.value)}
          className="form-control"
          placeholder="Twitter API token"
        />
        <small className="form-text text-muted">Twitter API token with posting permissions</small>
      </div>
      
      <div className="form-group mb-3">
        <div className="form-check">
          <input
            type="checkbox"
            id={`noCharacterLimits${index}`}
            checked={getConfigValue('noCharacterLimits', '') === 'true'}
            onChange={(e) => onActionConfigChange('noCharacterLimits', e.target.checked ? 'true' : 'false')}
            className="form-check-input"
          />
          <label className="form-check-label" htmlFor={`noCharacterLimits${index}`}>
            Disable character limit (280 characters)
          </label>
          <small className="form-text text-muted d-block">Enable to bypass the 280 character limit check</small>
        </div>
      </div>
    </div>
  );
};

export default TwitterPostAction;
