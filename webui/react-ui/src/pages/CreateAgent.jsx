import { useState } from 'react';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { agentApi } from '../utils/api';
import AgentForm from '../components/AgentForm';

function CreateAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(false);
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

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!formData.name.trim()) {
      showToast('Agent name is required', 'error');
      return;
    }
    
    setLoading(true);
    
    try {
      const response = await agentApi.createAgent(formData);
      showToast(`Agent "${formData.name}" created successfully`, 'success');
      navigate(`/settings/${formData.name}`);
    } catch (err) {
      showToast(`Error creating agent: ${err.message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="create-agent-container">
      <header className="page-header">
        <h1>
          <i className="fas fa-plus-circle"></i> Create New Agent
        </h1>
      </header>
      
      <div className="create-agent-content">
        <div className="section-box">
          <h2>
            <i className="fas fa-robot"></i> Agent Configuration
          </h2>
          
          <AgentForm
            formData={formData}
            setFormData={setFormData}
            onSubmit={handleSubmit}
            loading={loading}
            submitButtonText="Create Agent"
            isEdit={false}
          />
        </div>
      </div>
    </div>
  );
}

export default CreateAgent;
