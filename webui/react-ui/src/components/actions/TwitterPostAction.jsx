import React from 'react';
import BaseAction from './BaseAction';

/**
 * Twitter Post action component
 */
const TwitterPostAction = ({ index, onActionConfigChange, getConfigValue }) => {
  // Field definitions for Twitter Post action
  const fields = [
    {
      name: 'token',
      label: 'Twitter API Token',
      type: 'text',
      defaultValue: '',
      placeholder: 'Twitter API token',
      helpText: 'Twitter API token with posting permissions',
      required: true,
    },
    {
      name: 'noCharacterLimits',
      label: 'Disable character limit (280 characters)',
      type: 'checkbox',
      defaultValue: 'false',
      helpText: 'Enable to bypass the 280 character limit check',
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

export default TwitterPostAction;
