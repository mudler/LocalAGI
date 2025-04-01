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
        <div className="agents-grid">
          {agents.map(name => (
            <div key={name} className="agent-card" data-agent={name} data-active={statuses[name]}>
              <div className="agent-content text-center">
                <div className="avatar-container mb-4">
                  <img 
                    src={`/avatars/${name}.png`} 
                    alt={name} 
                    className="w-24 h-24 rounded-full" 
                    style={{
                      border: '2px solid var(--primary)', 
                      boxShadow: 'var(--neon-glow)', 
                      display: 'none',
                      margin: '0 auto'
                    }}
                    onLoad={(e) => {
                      e.target.style.display = 'block';
                      e.target.nextElementSibling.style.display = 'none';
                    }}
                    onError={(e) => {
                      e.target.style.display = 'none';
                      e.target.nextElementSibling.style.display = 'flex';
                    }}
                  />
                  <div className="avatar-placeholder" style={{margin: '0 auto'}}>
                    <span className="placeholder-text"><i className="fas fa-sync fa-spin"></i></span>
                  </div>
                </div>
                
                <div className="agent-header">
                  <h2>{name}</h2>
                  <span className={`status-badge ${statuses[name] ? 'active' : 'inactive'}`}>
                    {statuses[name] ? 'Active' : 'Paused'}
                  </span>
                </div>
              
                <div className="agent-actions">
                  <Link to={`/talk/${name}`} className="action-btn chat-btn">
                    <i className="fas fa-comment"></i> Chat
                  </Link>
                  <Link to={`/status/${name}`} className="action-btn status-btn">
                    <i className="fas fa-chart-line"></i> Status
                  </Link>
                  <Link to={`/settings/${name}`} className="action-btn settings-btn">
                    <i className="fas fa-cog"></i> Settings
                  </Link>
                </div>
                
                <div className="agent-actions mt-2">
                  <button 
                    className="action-btn toggle-btn"
                    onClick={() => toggleAgentStatus(name, statuses[name])}
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
                  >
                    <i className="fas fa-trash-alt"></i> Delete
                  </button>
                </div>
              </div>
            </div>
          ))}
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
