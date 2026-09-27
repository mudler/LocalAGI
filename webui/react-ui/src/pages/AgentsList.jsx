import { useState, useEffect } from 'react';
import { Link, useOutletContext } from 'react-router-dom';
import { agentApi } from '../utils/api';

function AgentsList() {
  const [agents, setAgents] = useState([]);
  const [statuses, setStatuses] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const { showToast } = useOutletContext();

  // Fetch agents data
  const fetchAgents = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/agents');
      if (!response.ok) {
        throw new Error(`Server responded with status: ${response.status}`);
      }

      const data = await response.json();
      setAgents(data.agents || []);
      setStatuses(data.statuses || {});
    } catch (err) {
      console.error('Error fetching agents:', err);
      setError('Failed to load agents');
    } finally {
      setLoading(false);
    }
  };

  // Toggle agent status (pause/start)
  const toggleAgentStatus = async (name, isActive) => {
    try {
      const endpoint = isActive ? `/api/agent/${name}/pause` : `/api/agent/${name}/start`;
      const response = await fetch(endpoint, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });

      if (response.ok) {
        // Update local state
        setStatuses(prev => ({
          ...prev,
          [name]: !isActive
        }));

        // Show success toast
        const action = isActive ? 'paused' : 'started';
        showToast(`Agent "${name}" ${action} successfully`, 'success');

        // Refresh the agents list to ensure we have the latest data
        fetchAgents();
      } else {
        const errorData = await response.json().catch(() => null);
        throw new Error(errorData?.error || `Server responded with status: ${response.status}`);
      }
    } catch (err) {
      console.error(`Error toggling agent status:`, err);
      showToast(`Failed to update agent status: ${err.message}`, 'error');
    }
  };

  // Delete an agent
  const deleteAgent = async (name) => {
    if (!confirm(`Are you sure you want to delete agent "${name}"? This action cannot be undone.`)) {
      return;
    }

    try {
      const response = await fetch(`/api/agent/${name}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
      });

      if (response.ok) {
        // Remove from local state
        setAgents(prev => prev.filter(agent => agent !== name));
        setStatuses(prev => {
          const newStatuses = { ...prev };
          delete newStatuses[name];
          return newStatuses;
        });

        // Show success toast
        showToast(`Agent "${name}" deleted successfully`, 'success');
      } else {
        const errorData = await response.json().catch(() => null);
        throw new Error(errorData?.error || `Server responded with status: ${response.status}`);
      }
    } catch (err) {
      console.error(`Error deleting agent:`, err);
      showToast(`Failed to delete agent: ${err.message}`, 'error');
    }
  };

  useEffect(() => {
    document.title = 'Agents - LocalAGI';
    return () => {
      document.title = 'LocalAGI'; // Reset title when component unmounts
    };
  }, []);

  // Load agents on mount
  useEffect(() => {
    fetchAgents();
  }, []);

  if (loading) {
    return <div className="loading">Loading agents...</div>;
  }

  if (error) {
    return <div className="error">{error}</div>;
  }

  return (
    <div className="agents-container">
      <header className="page-header">
        <h1>Manage Agents</h1>
        <div className="agent-actions">
          <Link to="/create" className="action-btn">
            <i className="fas fa-plus-circle"></i> Create Agent
          </Link>
          <Link to="/import" className="action-btn">
            <i className="fas fa-upload"></i> Import Agent
          </Link>
        </div>
      </header>

      {agents.length > 0 ? (
        <div className="agents-table-container">
          <table className="agents-table">
            <thead>
              <tr>
                <th>Agent Name</th>
                <th>Status</th>
                <th>Quick Actions</th>
                <th>Management</th>
              </tr>
            </thead>
            <tbody>
              {agents.map(name => (
                <tr key={name} data-agent={name} data-active={statuses[name]}>
                  <td>
                    <div className="agent-info">
                      <span className="agent-name-main">{name}</span>
                    </div>
                  </td>
                  <td>
                    <span className={`status-badge ${statuses[name] ? 'active' : 'inactive'}`}>
                      {statuses[name] ? 'Active' : 'Paused'}
                    </span>
                  </td>
                  <td>
                    <div className="agent-table-actions">
                      <Link to={`/talk/${name}`} className="action-btn chat-btn" title="Chat">
                        <i className="fas fa-comment"></i> Chat
                      </Link>
                      <Link to={`/status/${name}`} className="action-btn status-btn" title="Status">
                        <i className="fas fa-chart-line"></i> Status
                      </Link>
                      <Link to={`/settings/${name}`} className="action-btn settings-btn" title="Settings">
                        <i className="fas fa-cog"></i> Settings
                      </Link>
                    </div>
                  </td>
                  <td>
                    <div className="agent-table-actions">
                      <button
                        className="action-btn toggle-btn"
                        onClick={() => toggleAgentStatus(name, statuses[name])}
                        title={statuses[name] ? "Pause Agent" : "Start Agent"}
                      >
                        {statuses[name] ? (
                          <><i className="fas fa-pause"></i> Pause</>
                        ) : (
                          <><i className="fas fa-play"></i> Start</>
                        )}
                      </button>

                      <button
                        className="action-btn delete-btn"
                        onClick={() => deleteAgent(name)}
                        title="Delete Agent"
                      >
                        <i className="fas fa-trash-alt"></i> Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="no-agents">
          <h2>No Agents Found</h2>
          <p>Get started by creating your first agent</p>
          <Link to="/create" className="action-btn">
            <i className="fas fa-plus"></i> Create Agent
          </Link>
        </div>
      )}
    </div>
  );
}

export default AgentsList;
