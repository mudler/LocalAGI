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
      const response = await fetch('/agents');
      const html = await response.text();
      
      // Create a temporary element to parse the HTML
      const tempDiv = document.createElement('div');
      tempDiv.innerHTML = html;
      
      // Extract agent names and statuses from the HTML
      const agentElements = tempDiv.querySelectorAll('[data-agent]');
      const agentList = [];
      const statusMap = {};
      
      agentElements.forEach(el => {
        const name = el.getAttribute('data-agent');
        const status = el.getAttribute('data-active') === 'true';
        if (name) {
          agentList.push(name);
          statusMap[name] = status;
        }
      });
      
      setAgents(agentList);
      setStatuses(statusMap);
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
      const endpoint = isActive ? `/pause/${name}` : `/start/${name}`;
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
      } else {
        throw new Error(`Server responded with status: ${response.status}`);
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
      const response = await fetch(`/delete/${name}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
      });
      
      if (response.ok) {
        // Remove from local state
        setAgents(prev => prev.filter(agent => agent !== name));
        
        // Show success toast
        showToast(`Agent "${name}" deleted successfully`, 'success');
      } else {
        throw new Error(`Server responded with status: ${response.status}`);
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
        <Link to="/create" className="create-btn">
          <i className="fas fa-plus"></i> Create New Agent
        </Link>
      </header>

      {agents.length > 0 ? (
        <div className="agents-grid">
          {agents.map(name => (
            <div key={name} className="agent-card" data-agent={name} data-active={statuses[name]}>
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
                <Link to={`/settings/${name}`} className="action-btn settings-btn">
                  <i className="fas fa-cog"></i> Settings
                </Link>
                <Link to={`/status/${name}`} className="action-btn status-btn">
                  <i className="fas fa-chart-line"></i> Status
                </Link>
                
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
          ))}
        </div>
      ) : (
        <div className="no-agents">
          <h2>No Agents Found</h2>
          <p>Get started by creating your first agent</p>
          <Link to="/create" className="create-agent-btn">
            Create Agent
          </Link>
        </div>
      )}
    </div>
  );
}

export default AgentsList;
