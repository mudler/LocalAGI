import { useState, useEffect } from "react";
import { useOutletContext } from "react-router-dom";
import { actionApi } from "../utils/api";

function ActionsPlayground() {
  const { showToast } = useOutletContext();
  const [actions, setActions] = useState([]);
  const [selectedAction, setSelectedAction] = useState("");
  const [configJson, setConfigJson] = useState("{}");
  const [paramsJson, setParamsJson] = useState("{}");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [loadingActions, setLoadingActions] = useState(true);

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
  }, [showToast]);

  const handleActionChange = (e) => {
    setSelectedAction(e.target.value);
    setResult(null);
  };
  const handleConfigChange = (e) => setConfigJson(e.target.value);
  const handleParamsChange = (e) => setParamsJson(e.target.value);

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
        showToast("Invalid configuration JSON", "error");
        setLoading(false);
        return;
      }
      try {
        params = JSON.parse(paramsJson);
      } catch (err) {
        showToast("Invalid parameters JSON", "error");
        setLoading(false);
        return;
      }
      const actionData = { action: selectedAction, config, params };
      const response = await actionApi.executeAction(selectedAction, actionData);
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
        <div className="section-title" style={{ marginBottom: "2.5rem" }}>
          <h1 style={{ margin: 0, fontSize: "2rem" }}>Actions Playground</h1>
          <div style={{ color: "var(--text-light)", fontSize: "1.1rem", marginTop: 8 }}>
            Test and execute actions directly from the UI.
          </div>
        </div>

        <div className="agent-form-container" style={{ gap: 40 }}>
          {/* Left column: Action selection and config */}
          <div style={{ flex: 1, minWidth: 320 }}>
            <div className="section-box" style={{ marginBottom: 32 }}>
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
              <div className="section-box" style={{ marginBottom: 32 }}>
                <h2 className="section-title" style={{ fontSize: "1.2rem", marginBottom: 18 }}>Action Configuration</h2>
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
                    <small className="form-text text-muted">Enter JSON configuration for the action</small>
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
                    <small className="form-text text-muted">Enter JSON parameters for the action</small>
                  </div>
                  <div className="form-actions">
                    <button
                      type="submit"
                      className="action-btn"
                      disabled={loading}
                      aria-label="Execute Action"
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
          </div>

          {/* Right column: Results */}
          <div style={{ flex: 1, minWidth: 320 }}>
            {result && (
              <div className="section-box" style={{ minHeight: 220 }}>
                <h2 className="section-title" style={{ fontSize: "1.2rem", marginBottom: 18 }}>Action Results</h2>
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
                    <pre style={{ margin: 0, whiteSpace: "pre-wrap", wordBreak: "break-word" }}>
                      {JSON.stringify(result, null, 2)}
                    </pre>
                  ) : (
                    <pre style={{ margin: 0, whiteSpace: "pre-wrap", wordBreak: "break-word" }}>{result}</pre>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default ActionsPlayground;
