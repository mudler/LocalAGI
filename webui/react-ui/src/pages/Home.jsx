import { useState, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { agentApi } from '../utils/api';

function Home() {
  const [stats, setStats] = useState({
    agents: [],
    agentCount: 0,
    actions: 0,
    connectors: 0,
    status: {},
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const location = useLocation();

  // Update document title
  useEffect(() => {
    document.title = 'Agent Dashboard - LocalAGI';
    return () => {
      document.title = 'LocalAGI'; // Reset title when component unmounts
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
        console.error('Error fetching dashboard data:', err);
        setError('Failed to load dashboard data');
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

  return (
    <div>
      {/* Dashboard Stats */}
      <div className="dashboard-stats">
        <div className="stat-item">
          <div className="stat-count">{stats.actions}</div>
          <div className="stat-label">Available Actions</div>
        </div>
        <div className="stat-item">
          <div className="stat-count">{stats.connectors}</div>
          <div className="stat-label">Available Connectors</div>
        </div>
        <div className="stat-item">
          <div className="stat-count">{stats.agentCount}</div>
          <div className="stat-label">Agents</div>
        </div>
      </div>

      {/* Cards Container */}
      <div className="cards-container">
        {/* Card for Agent List Page */}
        <Link to="/agents" className="card-link">
          <div className="card">
            <h2><i className="fas fa-robot"></i> Agent List</h2>
            <p>View and manage your list of agents, including detailed profiles and statistics.</p>
          </div>
        </Link>

        {/* Card for Create Agent */}
        <Link to="/create" className="card-link">
          <div className="card">
            <h2><i className="fas fa-plus-circle"></i> Create Agent</h2>
            <p>Create a new intelligent agent with custom behaviors, connectors, and actions.</p>
          </div>
        </Link>

        {/* Card for Actions Playground */}
        <Link to="/actions-playground" className="card-link">
          <div className="card">
            <h2><i className="fas fa-code"></i> Actions Playground</h2>
            <p>Explore and test available actions for your agents.</p>
          </div>
        </Link>

        {/* Card for Group Create */}
        <Link to="/group-create" className="card-link">
          <div className="card">
            <h2><i className="fas fa-users"></i> Create Group</h2>
            <p>Create a group of agents with shared configurations and behaviors.</p>
          </div>
        </Link>

        {/* Card for Import Agent */}
        <Link to="/import" className="card-link">
          <div className="card">
            <h2><i className="fas fa-upload"></i> Import Agent</h2>
            <p>Import an existing agent configuration from a file.</p>
          </div>
        </Link>

      </div>

    </div>
  );
}

export default Home;
