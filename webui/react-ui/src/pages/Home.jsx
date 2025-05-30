import { useState, useEffect } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { agentApi } from "../utils/api";
import Header from "../components/Header";

function Home() {
  const { showToast } = useOutletContext();
  const [stats, setStats] = useState({
    agents: [],
    agentCount: 0,
    actions: 0,
    connectors: 0,
    status: {},
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Update document title
  useEffect(() => {
    document.title = "Agent Dashboard - LocalAGI";
    return () => {
      document.title = "LocalAGI"; // Reset title when component unmounts
    };
  }, []);

  // Fetch dashboard data
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const agents = await agentApi.getAgents();
        setStats({
          agents: agents.agents || [],
          agentCount: agents.agentCount || 0,
          actions: agents.actions || 0,
          connectors: agents.connectors || 0,
          status: agents.statuses || {},
        });
      } catch (err) {
        console.error("Error fetching dashboard data:", err);
        setError("Failed to load dashboard data");
        showToast && showToast("Failed to load dashboard data", "error");
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) {
    return <div className="loading">Loading dashboard data...</div>;
  }

  if (error) {
    return <div className="error">{error}</div>;
  }

  const currentDate = new Date().toLocaleDateString("en-US", {
    month: "long",
    day: "numeric",
    year: "numeric",
  });

  return (
    <div className="dashboard-container">
      <div className="sidebar">
        <div className="logo-container sidebar-logo-container">
          <div className="dots-background">
            <img src="/app/dots.png" alt="dots" className="dots-image" />
          </div>
          <img
            src="/app/logo_1.png"
            alt="BitGPT Network"
            className="sidebar-logo"
          />
        </div>
        <h2 className="sidebar-title">BitGPT Network</h2>
        <p className="sidebar-subtitle">
          Start by creating your agent or exploring available actions.
        </p>
      </div>

      <div className="main-content-area">
        <div className="header-container">
          <Header title="Welcome back" description={currentDate} />
        </div>

        {/* Dashboard Stats */}
        <div className="dashboard-stats">
          <div className="stat-card-outer">
            <div className="stat-header">
              <i className="fas fa-bolt"></i> Available Actions
            </div>
            <div className="stat-card-inner">
              <div className="stat-value">{stats.actions}</div>
            </div>
          </div>

          <div className="stat-card-outer">
            <div className="stat-header">
              <i className="fas fa-plug"></i> Available Connectors
            </div>
            <div className="stat-card-inner">
              <div className="stat-value">{stats.connectors}</div>
            </div>
          </div>

          <div className="stat-card-outer">
            <div className="stat-header">
              <i className="fas fa-robot"></i> Agents
            </div>
            <div className="stat-card-inner">
              <div className="stat-value">{stats.agentCount}</div>
            </div>
          </div>
        </div>

        {stats.agents.length > 0 ? (
          <div className="agents-section">
            <h2>Your Agents</h2>
            <div className="agents-grid">
              {stats.agents.map((agent) => (
                <div key={agent.id} className="agent-card">
                  <div className="agent-header">
                    <h3>
                      <i className="fas fa-robot"></i> {agent.name}
                    </h3>
                    <div
                      className={`status-badge ${
                        stats.status[agent.id]
                          ? "status-active"
                          : "status-paused"
                      }`}
                    >
                      {stats.status[agent.id] ? "Active" : "Paused"}
                    </div>
                  </div>
                  <div className="agent-actions">
                    <Link
                      to={`/talk/${agent.id}`}
                      className="agent-action-btn chat-btn"
                    >
                      <i className="fas fa-comment"></i> Chat
                    </Link>
                    <Link
                      to={`/settings/${agent.id}`}
                      className="agent-action-btn settings-btn"
                    >
                      <i className="fas fa-cog"></i> Settings
                    </Link>
                    <Link
                      to={`/status/${agent.id}`}
                      className="agent-action-btn status-btn"
                    >
                      <i className="fas fa-chart-line"></i> Status
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <>
            <div className="section-title">
              <h2>Manage Agents</h2>
              <p>
                Easily manage, access, and interact with all your agents from
                one place.
              </p>
            </div>
            <div className="features-grid">
              {/* Card for Create Agent */}
              <Link to="/create" className="feature-card">
                <img
                  src="/app/features/duplicate-plus.svg"
                  alt="Duplicate Plus"
                />
                <div className="feature-content">
                  <h3>Create Agent</h3>
                  <p>Agent with custom behaviors, connectors, and actions.</p>
                </div>
              </Link>

              {/* Card for Create Group */}
              <Link to="/group-create" className="feature-card">
                <img src="/app/features/user-group.svg" alt="User Group" />
                <div className="feature-content">
                  <h3>Create Group</h3>
                  <p>Group agents with shared configs and behaviors.</p>
                </div>
              </Link>

              {/* Card for Import Agent */}
              <Link to="/import" className="feature-card">
                <img
                  src="/app/features/dashed-upload.svg"
                  alt="Dashed Upload"
                />
                <div className="feature-content">
                  <h3>Import Agent</h3>
                  <p>Import an existing agent configuration from a file.</p>
                </div>
              </Link>

              {/* Card for Agent List */}
              <Link to="/agents" className="feature-card">
                <img src="/app/features/robot.svg" alt="Robot" />
                <div className="feature-content">
                  <h3>Agent List</h3>
                  <p>
                    Manage agents, including detailed profiles and statistics.
                  </p>
                </div>
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export default Home;
