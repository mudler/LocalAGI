import React from 'react';

/**
 * FallbackAction component for actions without specific configuration
 */
const FallbackAction = ({ index, onActionConfigChange, getConfigValue }) => {
  return (
    <div className="fallback-action">
      <p className="text-muted">
        This action doesn't require any additional configuration.
      </p>
    </div>
  );
};

export default FallbackAction;
