import { useState, useEffect } from "react";
import { useOutletContext } from "react-router-dom";
import { actionApi } from "../utils/api";
import Header from "../components/Header";
import { useState, useEffect } from 'react';
import { useOutletContext, useNavigate } from 'react-router-dom';
import { actionApi, agentApi } from '../utils/api';
import FormFieldDefinition from '../components/common/FormFieldDefinition';
import hljs from 'highlight.js/lib/core';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/monokai.css';
hljs.registerLanguage('json', json);

function ActionsPlayground() {
  const { showToast } = useOutletContext();
  const [actions, setActions] = useState([]);
  const [selectedAction, setSelectedAction] = useState("");
  const [configJson, setConfigJson] = useState("{}");
  const [paramsJson, setParamsJson] = useState("{}");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [loadingActions, setLoadingActions] = useState(true);
  const [actionMetadata, setActionMetadata] = useState(null);
  const [agentMetadata, setAgentMetadata] = useState(null);
  const [configFields, setConfigFields] = useState([]);
  const [paramFields, setParamFields] = useState([]);

  // Update document title
  useEffect(() => {
    document.title = "Actions Playground - LocalAGI";
    return () => {
      document.title = "LocalAGI";
    };
  }, []);

  // Fetch available actions
  useEffect(() => {
    const fetchActions = async () => {
      try {
        const response = await actionApi.listActions();
        setActions(response);
      } catch (err) {
        console.error("Error fetching actions:", err);
        showToast("Failed to load actions", "error");
      } finally {
        setLoadingActions(false);
      }
    };
    fetchActions();
  }, []);

  // Fetch agent metadata on mount
  useEffect(() => {
    const fetchAgentMetadata = async () => {
      try {
        const metadata = await agentApi.getAgentConfigMetadata();
        setAgentMetadata(metadata);
      } catch (err) {
        console.error('Error fetching agent metadata:', err);
        showToast('Failed to load agent metadata', 'error');
      }
    };

    fetchAgentMetadata();
  }, []);

  // Fetch action definition when action is selected or config changes
  useEffect(() => {
    if (!selectedAction) return;

    const fetchActionDefinition = async () => {
      try {
        // Get config fields from agent metadata
        const actionMeta = agentMetadata?.actions?.find(action => action.name === selectedAction);
        const configFields = actionMeta?.fields || [];
        console.debug('Config fields:', configFields);
        setConfigFields(configFields);

        // Parse current config to pass to action definition
        let currentConfig = {};
        try {
          currentConfig = JSON.parse(configJson);
        } catch (err) {
          console.error('Error parsing current config:', err);
        }

        // Get parameter fields from action definition
        const paramFields = await actionApi.getActionDefinition(selectedAction, currentConfig);
        console.debug('Parameter fields:', paramFields);
        setParamFields(paramFields);

        // Reset JSON to match the new fields
        setConfigJson(JSON.stringify(currentConfig, null, 2));
        setParamsJson(JSON.stringify({}, null, 2));
        setResult(null);
      } catch (err) {
        console.error('Error fetching action definition:', err);
        showToast('Failed to load action definition', 'error');
      }
    };

    fetchActionDefinition();
  }, [selectedAction, agentMetadata]);

  const handleActionChange = (e) => {
    setSelectedAction(e.target.value);
    setConfigJson('{}');
    setParamsJson('{}');
    setResult(null);
  };

  // Helper to generate onChange handlers for form fields
  const makeFieldChangeHandler = (fields, updateFn) => (e) => {
    let value;
    if (e && e.target) {
      const fieldName = e.target.name;
      const fieldDef = fields.find(f => f.name === fieldName);
      const fieldType = fieldDef ? fieldDef.type : undefined;
      if (fieldType === 'checkbox') {
        value = e.target.checked;
      } else if (fieldType === 'number') {
        value = e.target.value === '' ? '' : String(e.target.value);
      } else {
        value = e.target.value;
      }
      updateFn(fieldName, value);
    }
  };

  // Handle form field changes
  const handleConfigChange = (field, value) => {
    try {
      const config = JSON.parse(configJson);
      config[field] = value;
      setConfigJson(JSON.stringify(config, null, 2));
    } catch (err) {
      console.error('Error updating config:', err);
    }
  };

  const handleParamsChange = (field, value) => {
    try {
      const params = JSON.parse(paramsJson);
      params[field] = value;
      setParamsJson(JSON.stringify(params, null, 2));
    } catch (err) {
      console.error('Error updating params:', err);
    }
  };

  // Execute the selected action
  const handleExecuteAction = async (e) => {
    e.preventDefault();
    if (!selectedAction) {
      showToast("Please select an action", "warning");
      return;
    }
    setLoading(true);
    setResult(null);
    try {
      let config = {};
      let params = {};
      try {
        config = JSON.parse(configJson);
      } catch (err) {
        console.error("Error parsing configuration JSON:", err);
        showToast("Invalid configuration JSON", "error");
        setLoading(false);
        return;
      }
      try {
        params = JSON.parse(paramsJson);
      } catch (err) {
        console.error("Error parsing parameters JSON:", err);
        showToast("Invalid parameters JSON", "error");
        setLoading(false);
        return;
      }
      const actionData = { action: selectedAction, config, params };
      const response = await actionApi.executeAction(
        selectedAction,
        actionData
      );
      setResult(response);
      showToast("Action executed successfully", "success");
    } catch (err) {
      console.error("Error executing action:", err);
      showToast(`Failed to execute action: ${err.message}`, "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Actions Playground"
            description="Test and execute actions directly from the UI."
          />
        </div>

        <div
          className="agent-form-container"
          style={{ gap: 40, display: "flex", width: "100%" }}
        >
          {/* Left column: Action selection and config */}
          <div style={{ width: "100%" }}>
            <div
              className="section-box"
              style={{
                marginBottom: 32,
                width: "100%",
                maxWidth: "none",
                marginLeft: 0,
                marginRight: 0,
              }}
            >
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
                    <option key={action} value={action}>
                      {action}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {selectedAction && (
              <div
                className="section-box"
                style={{
                  marginBottom: 32,
                  width: "100%",
                  maxWidth: "none",
                  marginLeft: 0,
                  marginRight: 0,
                }}
              >
                <h2
                  className="section-title"
                  style={{ fontSize: "1.2rem", marginBottom: 18 }}
                >
                  Action Configuration
                </h2>
                <form onSubmit={handleExecuteAction} autoComplete="off">
                  <div className="form-group mb-4">
                    <label htmlFor="config-json">Configuration (JSON):</label>
                    <textarea
                      id="config-json"
                      value={configJson}
                      onChange={handleConfigChange}
                      className="form-control"
                      rows={4}
                      placeholder='{"key": "value"}'
                      spellCheck={false}
                    />
                    <small className="form-text text-muted">
                      Enter JSON configuration for the action
                    </small>
                  </div>
                  <div className="form-group mb-4">
                    <label htmlFor="params-json">Parameters (JSON):</label>
                    <textarea
                      id="params-json"
                      value={paramsJson}
                      onChange={handleParamsChange}
                      className="form-control"
                      rows={4}
                      placeholder='{"key": "value"}'
                      spellCheck={false}
                    />
                    <small className="form-text text-muted">
                      Enter JSON parameters for the action
                    </small>
                  </div>
                  <div className="form-actions">
                    <button
                      type="submit"
                      className="action-btn"
                      disabled={loading}
                      aria-label="Execute Action"
                    >
                      {loading ? (
                        <>
                          <i className="fas fa-spinner fa-spin"></i>{" "}
                          Executing...
                        </>
                      ) : (
                        <>
                          <i className="fas fa-play"></i> Execute Action
                        </>
                      )}
                    </button>
                  </div>
                </form>
              </div>
            )}
          </div>

          {/* Right column: Results */}
          <div>
            {result && (
              <div
                className="section-box"
                style={{
                  minWidth: 420,
                  minHeight: 220,
                  width: "100%",
                  maxWidth: "none",
                  marginLeft: 0,
                  marginRight: 0,
                }}
              >
                <h2
                  className="section-title"
                  style={{ fontSize: "1.2rem", marginBottom: 18 }}
                >
                  Action Results
                </h2>
                <div
                  className="result-container"
                  style={{
                    maxHeight: 400,
                    overflow: "auto",
                    border: "1px solid var(--border)",
                    borderRadius: 6,
                    padding: 14,
                    background: "#f9fafb",
                    fontFamily: "Menlo, Monaco, Consolas, monospace",
                    fontSize: 15,
                    color: "#1f2937",
                  }}
                >
                  {typeof result === "object" ? (
                    <pre
                      style={{
                        margin: 0,
                        whiteSpace: "pre-wrap",
                        wordBreak: "break-word",
                      }}
                    >
                      {JSON.stringify(result, null, 2)}
                    </pre>
                  ) : (
                    <pre
                      style={{
                        margin: 0,
                        whiteSpace: "pre-wrap",
                        wordBreak: "break-word",
                      }}
                    >
                      {result}
                    </pre>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
        
        {selectedAction && (
          <div className="section-box">
            
            <form onSubmit={handleExecuteAction}>
              {configFields.length > 0 && (
                <>
                <h2>Configuration</h2>
                <FormFieldDefinition
                  fields={configFields}
                  values={JSON.parse(configJson)}
                  onChange={makeFieldChangeHandler(configFields, handleConfigChange)}
                  idPrefix="config_"
                />
                </>
              )}

              {paramFields.length > 0 && (
                <>
                <h2>Parameters</h2>
                <FormFieldDefinition
                  fields={paramFields}
                  values={JSON.parse(paramsJson)}
                  onChange={makeFieldChangeHandler(paramFields, handleParamsChange)}
                  idPrefix="param_"
                />
                </>
              )}
              
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
                <pre className="hljs"><code>
                  <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(result, null, 2), { language: 'json' }).value }}></div>
                </code></pre>
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
