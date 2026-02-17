import { useState, useEffect } from 'react';
import { useParams, useOutletContext, useNavigate } from 'react-router-dom';
import { useAgent } from '../hooks/useAgent';
import { agentApi } from '../utils/api';
import AgentForm from '../components/AgentForm';

function AgentSettings() {
  const { name } = useParams();
  const { showToast } = useOutletContext();
  const navigate = useNavigate();
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({});

  // Update document title
  useEffect(() => {
    if (name) {
      document.title = `${name} - Settings - LocalAGI`;
    }
    return () => {
      document.title = 'LocalAGI';
    };
  }, [name]);

  // Use our custom agent hook
  const { 
    agent, 
    loading, 
    error, 
    updateAgent, 
    toggleAgentStatus, 
    deleteAgent 
  } = useAgent(name);

  // Fetch metadata on component mount
  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        const response = await agentApi.getAgentConfigMetadata();
        if (response) {
          setMetadata(response);
        }
      } catch (error) {
        console.error('Error fetching metadata:', error);
      }
    };

    fetchMetadata();
  }, []);

  // Load agent data when component mounts
  useEffect(() => {
    if (agent) {
      setFormData({
        ...formData,
        ...agent,
        name: name
      });
    }
  }, [agent]);

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    try {
      const success = await updateAgent(formData);
      if (success) {
        showToast('Agent updated successfully', 'success');
      }
    } catch (err) {
      showToast(`Error updating agent: ${err.message}`, 'error');
    }
  };

  // Handle agent toggle (pause/start)
  const handleToggleStatus = async () => {
    const isActive = agent?.active || false;
    try {
      const success = await toggleAgentStatus(isActive);
      if (success) {
        const action = isActive ? 'paused' : 'started';
        showToast(`Agent "${name}" ${action} successfully`, 'success');
      }
    } catch (err) {
      showToast(`Error toggling agent status: ${err.message}`, 'error');
    }
  };

  // Handle agent deletion
  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete agent "${name}"? This action cannot be undone.`)) {
      return;
    }
    
    try {
      const success = await deleteAgent();
      if (success) {
        showToast(`Agent "${name}" deleted successfully`, 'success');
        navigate('/agents');
      }
    } catch (err) {
      showToast(`Error deleting agent: ${err.message}`, 'error');
    }
  };

  if (loading && !agent) {
    return (
      <div className="settings-container">
        <div className="loading">
          <div className="loader" />
          <p>Loading agent settings...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="settings-container">
        <div className="error">
          <i className="fas fa-exclamation-triangle" />
          <p>{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="settings-container">
      {/* Page Header */}
      <header className="page-header">
        <div className="header-title-section">
          <div className="agent-title-wrapper">
            <h1 className="agent-name">{name}</h1>
            <span className={`status-badge ${agent?.active ? 'status-active' : 'status-paused'}`}>
              {agent?.active ? 'Active' : 'Paused'}
            </span>
          </div>
          <p className="agent-subtitle">Configure agent behavior, models, and connections</p>
        </div>
        
        <div className="header-actions">
          <button 
            className={`action-btn ${agent?.active ? 'warning' : 'success'}`}
            onClick={handleToggleStatus}
          >
            <i className={`fas ${agent?.active ? 'fa-pause' : 'fa-play'}`} />
            {agent?.active ? 'Pause Agent' : 'Start Agent'}
          </button>
          <button 
            className="action-btn delete-btn"
            onClick={handleDelete}
          >
            <i className="fas fa-trash" />
            Delete
          </button>
        </div>
      </header>
      
      {/* Settings Content */}
      <div className="settings-content">
        <AgentForm 
          isEdit={true}
          formData={formData}
          setFormData={setFormData}
          onSubmit={handleSubmit}
          loading={loading}
          submitButtonText="Save Changes"
          metadata={metadata}
        />
      </div>
    </div>
  );
}

export default AgentSettings;
