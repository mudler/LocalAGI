import { useState, useEffect } from "react";
import { useParams, useOutletContext, useNavigate } from "react-router-dom";
import { useAgent } from "../hooks/useAgent";
import { agentApi } from "../utils/api";
import AgentForm from "../components/AgentForm";

function AgentSettings() {
  const { name } = useParams();
  const { showToast } = useOutletContext();
  const navigate = useNavigate();
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({});

  // Update document title
  useEffect(() => {
    if (name) {
      document.title = `Agent Settings: ${name} - LocalAGI`;
    }
    return () => {
      document.title = "LocalAGI";
    };
  }, [name]);

  // Use our custom agent hook
  const { agent, loading, updateAgent, toggleAgentStatus, deleteAgent } =
    useAgent(name);

  // Fetch metadata on component mount
  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        const response = await agentApi.getAgentConfigMetadata();
        setMetadata(response);
      } catch (err) {
        showToast("Failed to load agent metadata", "error");
      }
    };
    fetchMetadata();
  }, [showToast]);

  useEffect(() => {
    if (agent) {
      setFormData(agent);
    }
  }, [agent]);

  // Header action handlers
  const handlePauseResume = async () => {
    try {
      await toggleAgentStatus();
      showToast(agent?.active ? "Agent paused" : "Agent resumed", "success");
    } catch (err) {
      console.error("Error toggling agent status:", err);
      showToast("Failed to update agent status", "error");
    }
  };

  const handleDelete = async () => {
    if (!window.confirm("Are you sure you want to delete this agent?")) return;
    try {
      await deleteAgent();
      showToast("Agent deleted", "success");
      navigate("/agents");
    } catch (err) {
      console.error("Error deleting agent:", err);
      showToast("Failed to delete agent", "error");
    }
  };

  // Header status
  const statusColor = agent?.active ? "#22c55e" : "#f59e0b";
  const statusText = agent?.active ? "Active" : "Paused";

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        {/* Refreshed Header */}
        <div
          className="agent-settings-header"
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: "2.5rem",
            gap: 18,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 18 }}>
            <i
              className="fas fa-cogs"
              style={{ fontSize: 32, color: "var(--primary)" }}
            />
            <div>
              <div style={{ fontSize: "2rem", fontWeight: 700, color: "#222" }}>
                Agent Settings{" "}
                <span style={{ color: "var(--primary)" }}>- {name}</span>
              </div>
              <div
                style={{
                  color: "var(--text-light)",
                  fontSize: "1.1rem",
                  marginTop: 2,
                }}
              >
                Configure and manage the agent’s settings, connectors, and
                capabilities.
              </div>
            </div>
          </div>
          <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                fontWeight: 500,
                color: statusColor,
                fontSize: "1rem",
                background: "rgba(34,197,94,0.09)",
                borderRadius: 16,
                padding: "4px 14px",
                marginRight: 8,
              }}
            >
              <span
                style={{
                  display: "inline-block",
                  width: 9,
                  height: 9,
                  borderRadius: "50%",
                  background: statusColor,
                  marginRight: 8,
                }}
              ></span>
              {statusText}
            </span>
            <button
              className="action-btn"
              style={{ background: "#f6f8fa", color: "var(--primary)" }}
              onClick={handlePauseResume}
              disabled={loading}
            >
              <i
                className={`fas ${agent?.active ? "fa-pause" : "fa-play"}`}
              ></i>{" "}
              {agent?.active ? "Pause Agent" : "Resume Agent"}
            </button>
            <button
              className="action-btn"
              style={{
                background: "#fff0f0",
                color: "#dc2626",
                border: "1px solid #fca5a5",
              }}
              onClick={handleDelete}
              disabled={loading}
            >
              <i className="fas fa-trash"></i> Delete Agent
            </button>
          </div>
        </div>

        {/* Agent Form */}
        <div className="section-box">
          {metadata && formData ? (
            <AgentForm
              isEdit
              formData={formData}
              setFormData={setFormData}
              onSubmit={updateAgent}
              loading={loading}
              submitButtonText="Save Changes"
              metadata={metadata}
            />
          ) : (
            <div style={{ color: "var(--text-light)", padding: 24 }}>
              Loading agent configuration...
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default AgentSettings;
