import { useState, useEffect } from 'react';
import { useOutletContext, useNavigate } from 'react-router-dom';
import { actionApi } from '../utils/api';
import FormFieldDefinition from '../components/common/FormFieldDefinition';

function ActionsPlayground() {
  const { showToast } = useOutletContext();
  const [actions, setActions] = useState([]);
  const [selectedAction, setSelectedAction] = useState('');
  const [actionMeta, setActionMeta] = useState(null);
  const [configValues, setConfigValues] = useState({});
  const [paramsValues, setParamsValues] = useState({});
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [loadingActions, setLoadingActions] = useState(true);

  // Update document title
  useEffect(() => {
    document.title = 'Actions Playground - LocalAgent';
    return () => {
      document.title = 'LocalAgent'; // Reset title when component unmounts
    };
  }, []);

  // Fetch available actions
  useEffect(() => {
    const fetchActions = async () => {
      const response = await actionApi.listActions();
      setActions(response);
      setLoadingActions(false);
    };

    fetchActions();
  }, []);

  // Fetch action metadata when an action is selected
  useEffect(() => {
    if (selectedAction) {
      const fetchActionMeta = async () => {
        const response = await actionApi.getAgentConfigMeta();
        const meta = response.actions.find(a => a.name === selectedAction);
        setActionMeta(meta);
        // Reset values when action changes
        setConfigValues({});
        setParamsValues({});
      };

      fetchActionMeta();
    }
  }, [selectedAction]);

  // Handle action selection
  const handleActionChange = (e) => {
    setSelectedAction(e.target.value);
    setResult(null);
  };

  // Handle config field changes
  const handleConfigChange = (fieldName, value) => {
    setConfigValues(prev => ({
      ...prev,
      [fieldName]: value
    }));
  };

  // Handle params field changes
  const handleParamsChange = (fieldName, value) => {
    setParamsValues(prev => ({
      ...prev,
      [fieldName]: value
    }));
  };

  // Execute the selected action
  const handleExecuteAction = async (e) => {
    e.preventDefault();
    
    if (!selectedAction) {
      showToast('Please select an action', 'warning');
      return;
    }
    
    setLoading(true);
    setResult(null);
    
    try {
      // Prepare action data
      const actionData = {
        action: selectedAction,
        config: configValues,
        params: paramsValues
      };
      
      // Execute action
      const response = await actionApi.executeAction(selectedAction, actionData);
      setResult(response);
      showToast('Action executed successfully', 'success');
    } catch (err) {
      showToast('Failed to execute action', 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="actions-playground">
      <h1>Actions Playground</h1>
      
      <form onSubmit={handleExecuteAction}>
        <div className="form-group">
          <label htmlFor="actionSelect">Select Action</label>
          <select
            id="actionSelect"
            value={selectedAction}
            onChange={handleActionChange}
            disabled={loadingActions}
          >
            <option value="">Select an action...</option>
            {actions.map(action => (
              <option key={action} value={action}>{action}</option>
            ))}
          </select>
        </div>

        {actionMeta && (
          <>
            <h2>Configuration</h2>
            <FormFieldDefinition
              fields={actionMeta.configFields || []}
              values={configValues}
              onChange={handleConfigChange}
              idPrefix="config_"
            />

            <h2>Parameters</h2>
            <FormFieldDefinition
              fields={actionMeta.paramFields || []}
              values={paramsValues}
              onChange={handleParamsChange}
              idPrefix="param_"
            />
          </>
        )}

        <div className="form-group">
          <button type="submit" disabled={loading || !selectedAction}>
            {loading ? 'Executing...' : 'Execute Action'}
          </button>
        </div>
      </form>

      {result && (
        <div className="result-section">
          <h2>Result</h2>
          <pre>{JSON.stringify(result, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}

export default ActionsPlayground;
