import { useState, useEffect } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { usePrivy, useLogin } from "@privy-io/react-auth";
import { agentApi } from "../utils/api";
import Header from "../components/Header";
import FeatureCard from "../components/FeatureCard";

function Home() {
  const { ready, authenticated } = usePrivy();

  const { login } = useLogin();

  const { showToast } = useOutletContext();
  const [stats, setStats] = useState({
    agents: [],
    agentCount: 0,
    actions: 32,
    connectors: 9,
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
    if (!ready || !authenticated) {
      setLoading(false);
      return;
    }

    const fetchData = async () => {
      setLoading(true);
      try {
        const agents = await agentApi.getAgents();
        setStats({
          agents: agents.agents || [],
          agentCount: agents.agentCount || 0,
          actions: agents.actions || 32,
          connectors: agents.connectors || 9,
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
  }, [ready, authenticated, showToast]);

  if (!ready || loading) {
    return (
      <div className="loading-container">
        <div className="spinner"></div>
      </div>
    );
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
        {/* <div className="header-container">
          <Header title="Welcome back" description={currentDate} />
        </div> */}

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
              <FeatureCard
                to="/create"
                imageSrc="/app/features/duplicate-plus.svg"
                imageAlt="Duplicate Plus"
                title="Create Agent"
                description="Agent with custom behaviors, connectors, and actions."
                authenticated={authenticated}
                onLogin={login}
              />

              <FeatureCard
                to="/group-create"
                imageSrc="/app/features/user-group.svg"
                imageAlt="User Group"
                title="Create Group"
                description="Group agents with shared configs and behaviors."
                authenticated={authenticated}
                onLogin={login}
              />

              <FeatureCard
                to="/import"
                imageSrc="/app/features/dashed-upload.svg"
                imageAlt="Dashed Upload"
                title="Import Agent"
                description="Import an existing agent configuration from a file."
                authenticated={authenticated}
                onLogin={login}
              />

              <FeatureCard
                to="/agents"
                imageSrc="/app/features/robot.svg"
                imageAlt="Robot"
                title="Agent List"
                description="Manage agents, including detailed profiles and statistics."
                authenticated={authenticated}
                onLogin={login}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export default Home;
