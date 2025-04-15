import { useState, useEffect } from 'react';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { agentApi } from '../utils/api';
import AgentForm from '../components/AgentForm';

function GroupCreate() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(false);
  const [generatingProfiles, setGeneratingProfiles] = useState(false);
  const [activeStep, setActiveStep] = useState(1);
  const [selectedProfiles, setSelectedProfiles] = useState([]);
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({
    description: '',
    model: '',
    api_url: '',
    api_key: '',
    connectors: [],
    actions: [],
    profiles: []
  });

  // Update document title
  useEffect(() => {
    document.title = 'Create Agent Group - LocalAGI';
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
        showToast('Failed to load agent group metadata', 'error');
      }
    };

    fetchMetadata();
  }, [showToast]);

  // Handle form field changes
  const handleFormChange = (changes) => {
    setFormData((prev) => ({ ...prev, ...changes }));
  };

  // Handle form submit
  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      // Structure the data according to what the server expects
      const groupData = {
        agents: selectedProfiles.map(index => formData.profiles[index]),
        agent_config: {
          // Don't set name/description as they'll be overridden by each agent's values
          model: formData.model,
          api_url: formData.api_url,
          api_key: formData.api_key,
          connectors: formData.connectors,
          actions: formData.actions
        }
      };

      // API call to create agent group
      await agentApi.createGroup(groupData);
      showToast('Agent group created successfully!', 'success');
      navigate('/agents');
    } catch (err) {
      console.error('Error creating group:', err);
      showToast(`Failed to create group: ${err.message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  // Handle profile selection
  const handleProfileSelection = (index) => {
    const newSelectedProfiles = [...selectedProfiles];
    
    if (newSelectedProfiles.includes(index)) {
      // Remove from selection
      const profileIndex = newSelectedProfiles.indexOf(index);
      newSelectedProfiles.splice(profileIndex, 1);
    } else {
      // Add to selection
      newSelectedProfiles.push(index);
    }
    
    setSelectedProfiles(newSelectedProfiles);
  };

  // Handle select all profiles
  const handleSelectAll = (e) => {
    if (e.target.checked) {
      // Select all profiles
      setSelectedProfiles(formData.profiles.map((_, index) => index));
    } else {
      // Deselect all profiles
      setSelectedProfiles([]);
    }
  };

  // Navigate to next step
  const nextStep = () => {
    setActiveStep(activeStep + 1);
  };

  // Navigate to previous step
  const prevStep = () => {
    setActiveStep(activeStep - 1);
  };

  // Generate agent profiles
  const handleGenerateProfiles = async () => {
    if (!formData.description.trim()) {
      showToast('Please enter a description', 'warning');
      return;
    }
    
    setGeneratingProfiles(true);
    
    try {
      const response = await agentApi.generateGroupProfiles({
        description: formData.description
      });
      
      // The API returns an array of agent profiles directly
      const profiles = Array.isArray(response) ? response : [];
      
      setFormData({
        ...formData,
        profiles: profiles
      });
      
      // Auto-select all profiles
      setSelectedProfiles(profiles.map((_, index) => index));
      
      // Move to next step
      nextStep();
      
      showToast('Agent profiles generated successfully', 'success');
    } catch (err) {
      console.error('Error generating profiles:', err);
      showToast(`Failed to generate profiles: ${err.message}`, 'error');
    } finally {
      setGeneratingProfiles(false);
    }
  };

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="section-title" style={{ marginBottom: "2.5rem" }}>
          <h1 style={{ margin: 0, fontSize: "2rem" }}>Create Agent Group</h1>
          <div style={{ color: "var(--text-light)", fontSize: "1.1rem", marginTop: 8 }}>
            Organize agents by creating a new group with shared configuration and profiles.
          </div>
        </div>

        <div className="agent-form-container" style={{ gap: 40 }}>
          <div style={{ flex: 1, minWidth: 340 }}>
            <div className="section-box" style={{ marginBottom: 32 }}>
              {metadata ? (
                <AgentForm
                  metadata={metadata}
                  formData={formData}
                  onChange={handleFormChange}
                  loading={loading}
                  generatingProfiles={generatingProfiles}
                  activeStep={activeStep}
                  setActiveStep={setActiveStep}
                  selectedProfiles={selectedProfiles}
                  setSelectedProfiles={setSelectedProfiles}
                  isGroupForm={true}
                  handleGenerateProfiles={handleGenerateProfiles}
                  handleProfileSelection={handleProfileSelection}
                  handleSelectAll={handleSelectAll}
                  handleSubmit={handleSubmit}
                />
              ) : (
                <div style={{ color: "var(--text-light)", padding: 24 }}>Loading group configuration...</div>
              )}
            </div>
          </div>
          {/* Optionally, add a sidebar or info panel here if needed */}
        </div>
      </div>
    </div>
  );
}

export default GroupCreate;
