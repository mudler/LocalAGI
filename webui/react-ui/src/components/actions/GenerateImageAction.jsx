import React from 'react';
import BaseAction from './BaseAction';

/**
 * Generate Image action component
 */
const GenerateImageAction = ({ index, onActionConfigChange, getConfigValue }) => {
  // Field definitions for Generate Image action
  const fields = [
    {
      name: 'apiKey',
      label: 'OpenAI API Key',
      type: 'text',
      defaultValue: '',
      placeholder: 'sk-...',
      helpText: 'Your OpenAI API key for image generation',
      required: false,
    },
    {
      name: 'apiURL',
      label: 'API URL',
      type: 'text',
      defaultValue: '',
      placeholder: 'http://localai:8081',
      helpText: 'OpenAI compatible API endpoint URL',
      required: false,
    },
    {
      name: 'model',
      label: 'Model',
      type: 'text',
      defaultValue: '',
      placeholder: 'dall-e-3',
      helpText: 'Image generation model',
      required: false,
    }
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

export default GenerateImageAction;
