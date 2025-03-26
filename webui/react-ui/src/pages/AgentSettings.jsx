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
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    identity_guidance: '',
    random_identity: false,
    hud: false,
    model: '',
    multimodal_model: '',
    api_url: '',
    api_key: '',
    local_rag_url: '',
    local_rag_api_key: '',
    enable_reasoning: false,
    enable_kb: false,
    kb_results: 3,
    long_term_memory: false,
    summary_long_term_memory: false,
    connectors: [],
    actions: [],
    mcp_servers: [],
    system_prompt: '',
    user_prompt: '',
    goals: '',
    standalone_job: false,
    standalone_job_interval: 60,
    avatar: '',
    avatar_seed: '',
    avatar_style: 'default',
  });
  
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
        // Fetch metadata from the dedicated endpoint
        const response = await agentApi.getAgentConfigMetadata();
        if (response) {
          setMetadata(response);
        }
      } catch (error) {
        console.error('Error fetching metadata:', error);
        // Continue without metadata, the form will use default fields
      }
    };

    fetchMetadata();
  }, []);

  // Load agent data when component mounts
  useEffect(() => {
    if (agent) {
      // Set form data from agent config
      setFormData({
        ...formData,
        ...agent,
        name: name // Ensure name is set correctly
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
          <i className="fas fa-spinner fa-spin"></i>
          <p>Loading agent settings...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="settings-container">
        <div className="error">
          <i className="fas fa-exclamation-triangle"></i>
          <p>{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="settings-container">
      <header className="page-header">
        <h1>
          <i className="fas fa-cog"></i> Agent Settings - {name}
        </h1>
        <div className="header-actions">
          <button 
            className={`action-btn ${agent?.active ? 'warning' : 'success'}`}
            onClick={handleToggleStatus}
          >
            {agent?.active ? (
              <><i className="fas fa-pause"></i> Pause Agent</>
            ) : (
              <><i className="fas fa-play"></i> Start Agent</>
            )}
          </button>
          <button 
            className="action-btn delete-btn"
            onClick={handleDelete}
          >
            <i className="fas fa-trash"></i> Delete Agent
          </button>
        </div>
      </header>
      
      <div className="settings-content">
        {/* Agent Configuration Form Section */}
        <div className="section-box">
          
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
    </div>
  );
}

export default AgentSettings;
