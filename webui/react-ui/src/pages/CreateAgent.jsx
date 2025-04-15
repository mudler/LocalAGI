import { useState, useEffect } from 'react';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { agentApi } from '../utils/api';
import AgentForm from '../components/AgentForm';

function CreateAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(false);
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({});

  useEffect(() => {
    document.title = 'Create Agent - LocalAGI';
    return () => {
      document.title = 'LocalAGI'; // Reset title when component unmounts
    };
  }, []);

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

  // Handle form submission
  const handleSubmit = async (data) => {
    setLoading(true);
    try {
      await agentApi.createAgent(data);
      showToast && showToast('Agent created successfully!', 'success');
      navigate('/agents');
    } catch (error) {
      showToast && showToast('Failed to create agent', 'error');
      console.error('Error creating agent:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="welcome-section" style={{ marginBottom: 24 }}>
          <h1 className="welcome-title" style={{ fontSize: 28, fontWeight: 700, marginBottom: 0 }}>Create Agent</h1>
          <p style={{ color: '#6b7a90', fontSize: 15, marginTop: 8, marginBottom: 0 }}>
            Fill out the form below to create a new agent. You can customize its configuration and capabilities.
          </p>
        </div>
        <div style={{ marginTop: 32 }}>
          <AgentForm
            metadata={metadata}
            formData={formData}
            setFormData={setFormData}
            onSubmit={handleSubmit}
            loading={loading}
          />
        </div>
      </div>
    </div>
  );
}

export default CreateAgent;
