import React from 'react';
import ActionForm from '../ActionForm';

/**
 * ActionsSection component for the agent form
 */
const ActionsSection = ({ formData, setFormData }) => {
  // Handle action change
  const handleActionChange = (index, updatedAction) => {
    const updatedActions = [...(formData.actions || [])];
    updatedActions[index] = updatedAction;
    setFormData({
      ...formData,
      actions: updatedActions
    });
  };

  // Handle action removal
  const handleActionRemove = (index) => {
    const updatedActions = [...(formData.actions || [])].filter((_, i) => i !== index);
    setFormData({
      ...formData,
      actions: updatedActions
    });
  };

  // Handle adding an action
  const handleAddAction = () => {
    setFormData({
      ...formData,
      actions: [
        ...(formData.actions || []),
        { name: '', config: '{}' }
      ]
    });
  };

  return (
    <div className="actions-section">
      <h3>Actions</h3>
      <p className="text-muted">
        Configure actions that the agent can perform.
      </p>

      <ActionForm
        actions={formData.actions || []}
        onChange={handleActionChange}
        onRemove={handleActionRemove}
        onAdd={handleAddAction}
      />
    </div>
  );
};

export default ActionsSection;
