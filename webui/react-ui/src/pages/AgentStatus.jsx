import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";

function AgentStatus() {
  const { name } = useParams();
  const [statusData, setStatusData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [_eventSource, setEventSource] = useState(null);
  const [liveUpdates, setLiveUpdates] = useState([]);

  // Update document title
  useEffect(() => {
    if (name) {
      document.title = `Agent Status: ${name} - LocalAGI`;
    }
    return () => {
      document.title = "LocalAGI";
    };
  }, [name]);

  // Fetch initial status data
  useEffect(() => {
    const fetchStatusData = async () => {
      try {
        const response = await fetch(`/api/agent/${name}/status`);
        if (!response.ok) {
          throw new Error(`Server responded with status: ${response.status}`);
        }
        const data = await response.json();
        setStatusData(data);
      } catch (err) {
        console.error("Error fetching agent status:", err);
        setError(`Failed to load status for agent \"${name}\": ${err.message}`);
      } finally {
        setLoading(false);
      }
    };
    fetchStatusData();
    // eslint-disable-next-line
  }, [name]);

  // Header status helpers
  const isActive = statusData?.active;
  const statusColor = isActive ? "#22c55e" : "#f59e0b";
  const statusText = isActive ? "Active" : "Paused";

  // Helper function to safely convert any value to a displayable string
  const formatValue = (value) => {
    if (value === null || value === undefined) {
      return "N/A";
    }

    if (typeof value === "object") {
      try {
        return JSON.stringify(value, null, 2);
      } catch (err) {
        return "[Complex Object]";
      }
    }

    return String(value);
  };

  // Combine live updates with history
  const allUpdates = [...liveUpdates, ...(statusData?.History || [])];

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        {/* Refreshed Header */}
        <div
          className="agent-status-header"
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
              className="fas fa-chart-bar"
              style={{ fontSize: 32, color: "var(--primary)" }}
            />
            <div>
              <div style={{ fontSize: "2rem", fontWeight: 700, color: "#222" }}>
                Agent Status <span style={{ color: "var(--primary)" }}>- {name}</span>
              </div>
              <div
                style={{
                  color: "var(--text-light)",
                  fontSize: "1.1rem",
                  marginTop: 2,
                }}
              >
                Live status, activity, and logs for this agent.
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
            <Link
              to={`/settings/${name}`}
              className="action-btn"
              style={{ background: "#f6f8fa", color: "var(--primary)" }}
            >
              <i className="fas fa-cogs"></i> Agent Settings
            </Link>
          </div>
        </div>

        {/* Main Content */}
        <div className="section-box">
          {loading ? (
            <div style={{ color: "var(--text-light)", padding: 24 }}>
              Loading agent status...
            </div>
          ) : error ? (
            <div style={{ color: "#dc2626", padding: 24 }}>{error}</div>
          ) : statusData ? (
            <div>
              <h3 style={{ fontWeight: 600, marginBottom: 12 }}>Agent Info</h3>
              <div style={{ marginBottom: 18 }}>
                <strong>Name:</strong> {statusData.name} <br />
                <strong>Model:</strong> {statusData.model || "-"} <br />
                <strong>Uptime:</strong> {statusData.uptime || "-"} <br />
                <strong>Status:</strong> {statusText}
              </div>
              {/* Activity log or live updates */}
              <h3 style={{ fontWeight: 600, marginBottom: 12 }}>Recent Activity</h3>
              {allUpdates.length === 0 ? (
                <div style={{ color: "var(--text-light)" }}>No recent activity.</div>
              ) : (
                <div className="chat-container bg-gray-800 shadow-lg rounded-lg">
                  {/* Chat Messages */}
                  <div className="chat-messages p-4">
                    {allUpdates.map((item, index) => (
                      <div key={index} className="status-item mb-4">
                        <div className="bg-gray-700 p-4 rounded-lg">
                          <h2 className="text-sm font-semibold mb-2">Agent Action:</h2>
                          <div className="status-details">
                            <div className="status-row">
                              <span className="status-label">{index}</span>
                              <span className="status-value">{formatValue(item)}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

export default AgentStatus;
