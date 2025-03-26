import React from 'react';
import BaseAction from './BaseAction';

/**
 * SendMail action component
 */
const SendMailAction = ({ index, onActionConfigChange, getConfigValue }) => {
  // Field definitions for SendMail action
  const fields = [
    {
      name: 'email',
      label: 'Email',
      type: 'email',
      defaultValue: '',
      placeholder: 'your-email@example.com',
      helpText: 'Email address to send from',
      required: true,
    },
    {
      name: 'username',
      label: 'Username',
      type: 'text',
      defaultValue: '',
      placeholder: 'SMTP username (often same as email)',
      helpText: '',
      required: true,
    },
    {
      name: 'password',
      label: 'Password',
      type: 'password',
      defaultValue: '',
      placeholder: 'SMTP password or app password',
      helpText: 'For Gmail, use an app password',
      required: true,
    },
    {
      name: 'smtpHost',
      label: 'SMTP Host',
      type: 'text',
      defaultValue: '',
      placeholder: 'smtp.gmail.com',
      helpText: '',
      required: true,
    },
    {
      name: 'smtpPort',
      label: 'SMTP Port',
      type: 'text',
      defaultValue: '587',
      placeholder: '587',
      helpText: 'Common ports: 587 (TLS), 465 (SSL)',
      required: true,
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

export default SendMailAction;
