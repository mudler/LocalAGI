import { useState, useEffect } from 'react';
import { useOutletContext, useNavigate } from 'react-router-dom';
import { actionApi } from '../utils/api';

function ActionsPlayground() {
  const { showToast } = useOutletContext();
  const navigate = useNavigate();
  const [actions, setActions] = useState([]);
  const [selectedAction, setSelectedAction] = useState('');
  const [configJson, setConfigJson] = useState('{}');
  const [paramsJson, setParamsJson] = useState('{}');
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [loadingActions, setLoadingActions] = useState(true);

  // Update document title
  useEffect(() => {
    document.title = 'Actions Playground - LocalAGI';
    return () => {
      document.title = 'LocalAGI'; // Reset title when component unmounts
    };
  }, []);

  // Fetch available actions
  useEffect(() => {
    const fetchActions = async () => {
      try {
        const response = await actionApi.listActions();
        setActions(response);
      } catch (err) {
        console.error('Error fetching actions:', err);
        showToast('Failed to load actions', 'error');
      } finally {
        setLoadingActions(false);
      }
    };

    fetchActions();
  }, [showToast]);

  // Handle action selection
  const handleActionChange = (e) => {
    setSelectedAction(e.target.value);
    setResult(null);
  };

  // Handle JSON input changes
  const handleConfigChange = (e) => {
    setConfigJson(e.target.value);
  };

  const handleParamsChange = (e) => {
    setParamsJson(e.target.value);
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
      // Parse JSON inputs
      let config = {};
      let params = {};
      
      try {
        config = JSON.parse(configJson);
      } catch (err) {
        showToast('Invalid configuration JSON', 'error');
        setLoading(false);
        return;
      }
      
      try {
        params = JSON.parse(paramsJson);
      } catch (err) {
        showToast('Invalid parameters JSON', 'error');
        setLoading(false);
        return;
      }
      
      // Prepare action data
      const actionData = {
        action: selectedAction,
        config: config,
        params: params
      };
      
      // Execute action
      const response = await actionApi.executeAction(selectedAction, actionData);
      setResult(response);
      showToast('Action executed successfully', 'success');
    } catch (err) {
      console.error('Error executing action:', err);
      showToast(`Failed to execute action: ${err.message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="actions-playground-container">
      <header className="page-header">
        <h1>Actions Playground</h1>
        <p>Test and execute actions directly from the UI</p>
      </header>
      
      <div className="actions-playground-content">
        <div className="section-box">
          <h2>Select an Action</h2>
          
          <div className="form-group mb-4">
            <label htmlFor="action-select">Available Actions:</label>
            <select 
              id="action-select" 
              value={selectedAction}
              onChange={handleActionChange}
              className="form-control"
              disabled={loadingActions}
            >
              <option value="">-- Select an action --</option>
              {actions.map((action) => (
                <option key={action} value={action}>{action}</option>
              ))}
            </select>
          </div>
        </div>
        
        {selectedAction && (
          <div className="section-box">
            <h2>Action Configuration</h2>
            
            <form onSubmit={handleExecuteAction}>
              <div className="form-group mb-6">
                <label htmlFor="config-json">Configuration (JSON):</label>
                <textarea 
                  id="config-json"
                  value={configJson}
                  onChange={handleConfigChange}
                  className="form-control"
                  rows="5"
                  placeholder='{"key": "value"}'
                />
                <p className="text-xs text-gray-400 mt-1">Enter JSON configuration for the action</p>
              </div>
              
              <div className="form-group mb-6">
                <label htmlFor="params-json">Parameters (JSON):</label>
                <textarea 
                  id="params-json"
                  value={paramsJson}
                  onChange={handleParamsChange}
                  className="form-control"
                  rows="5"
                  placeholder='{"key": "value"}'
                />
                <p className="text-xs text-gray-400 mt-1">Enter JSON parameters for the action</p>
              </div>
              
              <div className="flex justify-end">
                <button 
                  type="submit" 
                  className="action-btn"
                  disabled={loading}
                >
                  {loading ? (
                    <><i className="fas fa-spinner fa-spin"></i> Executing...</>
                  ) : (
                    <><i className="fas fa-play"></i> Execute Action</>
                  )}
                </button>
              </div>
            </form>
          </div>
        )}
        
        {result && (
          <div className="section-box">
            <h2>Action Results</h2>
            
            <div className="result-container" style={{ 
              maxHeight: '400px', 
              overflow: 'auto', 
              border: '1px solid rgba(94, 0, 255, 0.2)',
              borderRadius: '4px',
              padding: '10px',
              backgroundColor: 'rgba(30, 30, 30, 0.7)'
            }}>
              {typeof result === 'object' ? (
                <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {JSON.stringify(result, null, 2)}
                </pre>
              ) : (
                <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {result}
                </pre>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ActionsPlayground;
