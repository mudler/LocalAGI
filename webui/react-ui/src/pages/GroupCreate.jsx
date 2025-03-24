import { useState } from 'react';
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
  const [formData, setFormData] = useState({
    description: '',
    model: '',
    api_url: '',
    api_key: '',
    connectors: [],
    actions: [],
    profiles: []
  });

  // Handle form field changes
  const handleInputChange = (e) => {
    const { name, value, type } = e.target;
    setFormData({
      ...formData,
      [name]: type === 'number' ? parseInt(value, 10) : value
    });
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

  // Create agent group
  const handleCreateGroup = async (e) => {
    e.preventDefault();
    
    if (selectedProfiles.length === 0) {
      showToast('Please select at least one agent profile', 'warning');
      return;
    }
    
    // Filter profiles to only include selected ones
    const selectedProfilesData = selectedProfiles.map(index => formData.profiles[index]);
    
    setLoading(true);
    
    try {
      // Structure the data according to what the server expects
      const groupData = {
        agents: selectedProfilesData,
        agent_config: {
          // Don't set name/description as they'll be overridden by each agent's values
          model: formData.model,
          api_url: formData.api_url,
          api_key: formData.api_key,
          connectors: formData.connectors,
          actions: formData.actions
        }
      };
      
      const response = await agentApi.createGroup(groupData);
      showToast(`Agent group "${formData.group_name}" created successfully`, 'success');
      navigate('/agents');
    } catch (err) {
      console.error('Error creating group:', err);
      showToast(`Failed to create group: ${err.message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="group-create-container">
      <div className="section-box">
        <h1>Create Agent Group</h1>
        
        {/* Progress Bar */}
        <div className="progress-container">
          <div className={`progress-step ${activeStep === 1 ? 'step-active' : ''}`}>
            <div className="step-circle">1</div>
            <div className="step-label">Generate Profiles</div>
          </div>
          <div className={`progress-step ${activeStep === 2 ? 'step-active' : ''}`}>
            <div className="step-circle">2</div>
            <div className="step-label">Review & Select</div>
          </div>
          <div className={`progress-step ${activeStep === 3 ? 'step-active' : ''}`}>
            <div className="step-circle">3</div>
            <div className="step-label">Configure Settings</div>
          </div>
        </div>
        
        {/* Step 1: Generate Profiles */}
        <div className={`page-section ${activeStep === 1 ? 'section-active' : ''}`}>
          <h2>Generate Agent Profiles</h2>
          <p>Describe the group of agents you want to create. Be specific about their roles, relationships, and purpose.</p>
          
          <div className="prompt-container">
            <textarea 
              id="description" 
              name="description"
              value={formData.description}
              onChange={handleInputChange}
              placeholder="Example: Create a team of agents for a software development project including a project manager, developer, tester, and designer. They should collaborate to build web applications."
              rows="5"
            />
          </div>
          
          <div className="action-buttons">
            <button 
              type="button" 
              className="action-btn"
              onClick={handleGenerateProfiles}
              disabled={generatingProfiles || !formData.description}
            >
              {generatingProfiles ? (
                <><i className="fas fa-spinner fa-spin"></i> Generating Profiles...</>
              ) : (
                <><i className="fas fa-magic"></i> Generate Profiles</>
              )}
            </button>
          </div>
        </div>
        
        {/* Loader */}
        {generatingProfiles && (
          <div className="loader" style={{ display: 'block' }}>
            <i className="fas fa-spinner fa-spin"></i>
            <p>Generating agent profiles...</p>
          </div>
        )}
        
        {/* Step 2: Review & Select Profiles */}
        <div className={`page-section ${activeStep === 2 ? 'section-active' : ''}`}>
          <h2>Review & Select Agent Profiles</h2>
          <p>Select the agents you want to create. You can customize their details before creation.</p>
          
          <div className="select-all-container">
            <label htmlFor="select-all" className="checkbox-label">
              <input 
                type="checkbox" 
                id="select-all"
                checked={selectedProfiles.length === formData.profiles.length}
                onChange={handleSelectAll}
              />
              <span>Select All</span>
            </label>
          </div>
          
          <div className="agent-profiles-container">
            {formData.profiles.map((profile, index) => (
              <div 
                key={index} 
                className={`agent-profile ${selectedProfiles.includes(index) ? 'selected' : ''}`}
                onClick={() => handleProfileSelection(index)}
              >
                <div className="select-checkbox">
                  <input 
                    type="checkbox"
                    checked={selectedProfiles.includes(index)}
                    onChange={() => handleProfileSelection(index)}
                  />
                </div>
                <h3>{profile.name || `Agent ${index + 1}`}</h3>
                <div className="description">{profile.description || 'No description available.'}</div>
                <div className="system-prompt">{profile.system_prompt || 'No system prompt defined.'}</div>
              </div>
            ))}
          </div>
          
          <div className="action-buttons">
            <button type="button" className="nav-btn" onClick={prevStep}>
              <i className="fas fa-arrow-left"></i> Back
            </button>
            <button 
              type="button" 
              className="action-btn"
              onClick={nextStep}
              disabled={selectedProfiles.length === 0}
            >
              Continue <i className="fas fa-arrow-right"></i>
            </button>
          </div>
        </div>
        
        {/* Step 3: Common Settings */}
        <div className={`page-section ${activeStep === 3 ? 'section-active' : ''}`}>
          <h2>Configure Common Settings</h2>
          <p>Configure common settings for all selected agents. These settings will be applied to each agent.</p>
          
          <form id="group-settings-form" onSubmit={handleCreateGroup}>
            {/* Informative message about profile data */}
            <div className="info-message">
              <i className="fas fa-info-circle"></i>
              <span>
                Each agent will be created with its own name, description, and system prompt from the selected profiles.
                The settings below will be applied to all agents.
              </span>
            </div>
            
            {/* Use AgentForm for common settings */}
            <div className="agent-form-wrapper">
              <AgentForm 
                formData={formData}
                setFormData={setFormData}
                onSubmit={handleCreateGroup}
                loading={loading}
                submitButtonText="Create Group"
                isGroupForm={true}
                noFormWrapper={true}
              />
            </div>
            
            <div className="action-buttons">
              <button type="button" className="nav-btn" onClick={prevStep}>
                <i className="fas fa-arrow-left"></i> Back
              </button>
              <button 
                type="submit" 
                className="action-btn"
                disabled={loading}
              >
                {loading ? (
                  <><i className="fas fa-spinner fa-spin"></i> Creating Group...</>
                ) : (
                  <><i className="fas fa-users"></i> Create Group</>
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
      <style>{`
        .progress-container {
          display: flex;
          justify-content: center;
          margin-bottom: 30px;
        }
        .progress-step {
          display: flex;
          flex-direction: column;
          align-items: center;
          position: relative;
          padding: 0 20px;
        }
        .progress-step:not(:last-child)::after {
          content: '';
          position: absolute;
          top: 12px;
          right: -30px;
          width: 60px;
          height: 3px;
          background-color: var(--medium-bg);
        }
        .progress-step.step-active:not(:last-child)::after {
          background-color: var(--primary);
        }
        .step-circle {
          width: 28px;
          height: 28px;
          border-radius: 50%;
          background-color: var(--medium-bg);
          display: flex;
          justify-content: center;
          align-items: center;
          color: var(--text);
          margin-bottom: 8px;
          transition: all 0.3s ease;
        }
        .progress-step.step-active .step-circle {
          background-color: var(--primary);
          box-shadow: 0 0 10px var(--primary);
        }
        .step-label {
          font-size: 0.9rem;
          color: var(--muted-text);
          transition: all 0.3s ease;
        }
        .progress-step.step-active .step-label {
          color: var(--primary);
          font-weight: bold;
        }
        .page-section {
          display: none;
          animation: fadeIn 0.5s;
        }
        .page-section.section-active {
          display: block;
        }
        .prompt-container {
          margin-bottom: 30px;
        }
        .prompt-container textarea {
          width: 100%;
          min-height: 120px;
          padding: 15px;
          border-radius: 6px;
          background-color: var(--lighter-bg);
          border: 1px solid var(--medium-bg);
          color: var(--text);
          font-size: 1rem;
          resize: vertical;
        }
        .action-buttons {
          display: flex;
          justify-content: space-between;
          margin-top: 30px;
        }
        .select-all-container {
          display: flex;
          align-items: center;
          margin-bottom: 20px;
        }
        .loader {
          text-align: center;
          margin: 40px 0;
        }
        .loader i {
          color: var(--primary);
          font-size: 2rem;
        }
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        .agent-profile {
          border: 1px solid var(--medium-bg);
          border-radius: 8px;
          padding: 15px;
          margin-bottom: 20px;
          background-color: var(--lighter-bg);
          position: relative;
          transition: all 0.3s ease;
          cursor: pointer;
        }
        .agent-profile:hover {
          transform: translateY(-3px);
          box-shadow: 0 10px 20px rgba(0, 0, 0, 0.2);
        }
        .agent-profile h3 {
          color: var(--primary);
          text-shadow: var(--neon-glow);
          margin-top: 0;
          margin-bottom: 15px;
          border-bottom: 1px solid var(--medium-bg);
          padding-bottom: 10px;
        }
        .agent-profile .description {
          color: var(--text);
          font-size: 0.9rem;
          margin-bottom: 15px;
        }
        .agent-profile .system-prompt {
          background-color: var(--darker-bg);
          border-radius: 6px;
          padding: 10px;
          font-size: 0.85rem;
          max-height: 150px;
          overflow-y: auto;
          margin-bottom: 10px;
          white-space: pre-wrap;
        }
        .agent-profile.selected {
          border: 2px solid var(--primary);
          background-color: rgba(94, 0, 255, 0.1);
        }
        .agent-profile .select-checkbox {
          position: absolute;
          top: 10px;
          right: 10px;
        }
        .info-message {
          background-color: rgba(94, 0, 255, 0.1);
          border-left: 4px solid var(--primary);
          padding: 15px;
          margin: 20px 0;
          border-radius: 0 8px 8px 0;
          display: flex;
          align-items: center;
        }
        .info-message i {
          font-size: 1.5rem;
          color: var(--primary);
          margin-right: 15px;
        }
        .info-message-content {
          flex: 1;
        }
        .info-message-content h4 {
          margin-top: 0;
          margin-bottom: 5px;
          color: var(--primary);
        }
        .info-message-content p {
          margin-bottom: 0;
        }
        .nav-btn {
          background-color: var(--medium-bg);
          color: var(--text);
          border: none;
          border-radius: 4px;
          padding: 8px 16px;
          cursor: pointer;
          transition: all 0.3s ease;
          display: flex;
          align-items: center;
          gap: 8px;
        }
        .nav-btn:hover {
          background-color: var(--lighter-bg);
        }
      `}</style>
    </div>
  );
}

export default GroupCreate;
