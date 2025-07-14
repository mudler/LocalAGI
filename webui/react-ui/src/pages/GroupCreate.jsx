import { useState, useEffect } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { agentApi } from "../utils/api";
import AgentForm from "../components/AgentForm";
import Header from "../components/Header";

function GroupCreate() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(false);
  const [generatingProfiles, setGeneratingProfiles] = useState(false);
  const [activeStep, setActiveStep] = useState(1);
  const [selectedProfiles, setSelectedProfiles] = useState([]);
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({
    description: "",
    model: "",
    api_url: "",
    api_key: "",
    connectors: [],
    actions: [],
    profiles: [],
  });

  // Update document title
  useEffect(() => {
    document.title = "Create Agent Group - LocalAGI";
    return () => {
      document.title = "LocalAGI"; // Reset title when component unmounts
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
        console.error("Error fetching metadata:", error);
        showToast("Failed to load agent group metadata", "error");
      }
    };

    fetchMetadata();
  }, [showToast]);

  // Handle form field changes
  const handleFormChange = (changes) => {
    setFormData((prev) => ({ ...prev, ...changes }));
  };

  // Handle input changes for textarea and other inputs
  const handleInputChange = (e) => {
    const { name, value, type } = e.target;
    setFormData({
      ...formData,
      [name]: type === "number" ? parseInt(value, 10) : value,
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
      showToast("Please enter a description", "warning");
      return;
    }

    setGeneratingProfiles(true);

    try {
      const response = await agentApi.generateGroupProfiles({
        description: formData.description,
      });

      // The API returns an array of agent profiles directly
      const profiles = Array.isArray(response) ? response : [];

      setFormData({
        ...formData,
        profiles: profiles,
      });

      // Auto-select all profiles
      setSelectedProfiles(profiles.map((_, index) => index));

      // Move to next step
      nextStep();

      showToast("Agent profiles generated successfully", "success");
    } catch (err) {
      console.error("Error generating profiles:", err);
      showToast(`Failed to generate profiles: ${err.message}`, "error");
    } finally {
      setGeneratingProfiles(false);
    }
  };

  // Back button for the header
  const backButton = (
    <button
      className="action-btn pause-resume-btn"
      onClick={() => navigate("/agents")}
    >
      <i className="fas fa-arrow-left"></i> Back to Agents
    </button>
  );

  // Initialize formData with default values when metadata is loaded
  useEffect(() => {
    if (metadata && Object.keys(formData).length === 0) {
      const defaultFormData = {
        // Initialize arrays for complex fields
        connectors: [],
        actions: [],
        dynamic_prompts: [],
        mcp_servers: [],
      };

      // Process all field sections to extract default values
      // const sections = [
      //   'BasicInfoSection',
      //   'ModelSettingsSection', 
      //   'MemorySettingsSection',
      //   'PromptsGoalsSection',
      //   'AdvancedSettingsSection'
      // ];

      const sections = [
        'ModelSettingsSection', 
      ];

      sections.forEach((sectionKey) => {
        if (metadata[sectionKey] && Array.isArray(metadata[sectionKey])) {
          metadata[sectionKey].forEach((field) => {
            if (field.name) {
              let defaultValue = field.defaultValue;
              
              // If field has options array, use the first option's value
              if (field.options && Array.isArray(field.options) && field.options.length > 0) {
                defaultValue = field.options[0].value;
              } else if (field.hasOwnProperty('defaultValue')) {
                defaultValue = field.defaultValue;
              }
              
              defaultFormData[field.name] = defaultValue;
            }
          });
        }
      });

      setFormData(defaultFormData);
    }
  }, [metadata, formData]);

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
      const { name, description, ...restFormData } = formData;
      const groupData = {
        agents: selectedProfilesData,
        agent_config: {
          // Don't set name/description as they'll be overridden by each agent's values
          ...restFormData
        }
      };
      
      const response = await agentApi.createGroup(groupData);
      showToast(`Agent group "${formData.group_name}" created successfully`, 'success');
      navigate('/agents');
    } catch (err) {
      if(error?.message){
        showToast && showToast(error.message.charAt(0).toUpperCase() + error.message.slice(1), "error");
      } else {
        showToast && showToast("Failed to create agent", "error");
      }
      console.error('Error creating group:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Create Agent Group"
            description="Organize agents by creating a new group with shared configuration and profiles."
          />
          <div className="header-right">{backButton}</div>
        </div>

        <div className="agent-form-container" style={{ gap: 40 }}>
          <div style={{ flex: 1, minWidth: 340 }}>
            <div className="section-box" style={{ marginBottom: 32 }}>
              {/* Progress Bar */}
              <div className="progress-container">
                <div className="progress-step">
                  <div className={`step-circle ${activeStep >= 1 ? (activeStep === 1 ? 'active' : 'completed') : 'inactive'}`}>
                    1
                  </div>
                  <div className={`step-label ${activeStep === 1 ? 'active' : 'inactive'}`}>
                    Generate Profiles
                  </div>
                </div>
                
                {/* Line between step 1 and 2 */}
                <div className={`progress-line line-1-2 ${activeStep > 1 ? 'completed' : 'inactive'}`}></div>
                
                <div className="progress-step">
                  <div className={`step-circle ${activeStep >= 2 ? (activeStep === 2 ? 'active' : 'completed') : 'inactive'}`}>
                    2
                  </div>
                  <div className={`step-label ${activeStep === 2 ? 'active' : 'inactive'}`}>
                    Review & Select
                  </div>
                </div>
                
                {/* Line between step 2 and 3 */}
                <div className={`progress-line line-2-3 ${activeStep > 2 ? 'completed' : 'inactive'}`}></div>
                
                <div className="progress-step">
                  <div className={`step-circle ${activeStep >= 3 ? (activeStep === 3 ? 'active' : 'completed') : 'inactive'}`}>
                    3
                  </div>
                  <div className={`step-label ${activeStep === 3 ? 'active' : 'inactive'}`}>
                    Configure Settings
                  </div>
                </div>
              </div>

              {/* Step 1: Generate Profiles */}
              {activeStep === 1 && (
                <div className="page-section">
                  <h2>Generate Agent Profiles</h2>
                  <p>
                    Describe the group of agents you want to create. Be specific about their roles, relationships, and purpose.
                  </p>
                  
                  <div className="form-field">
                    <textarea 
                      id="description" 
                      name="description"
                      value={formData.description}
                      onChange={handleInputChange}
                      placeholder="Example: Create a team of agents for a software development project including a project manager, developer, tester, and designer. They should collaborate to build web applications."
                      rows="5"
                    />
                  </div>
                  
                  <div className="button-container-end">
                    <button 
                      type="button" 
                      className="action-btn create-btn"
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
              )}

              {/* Step 2: Review & Select Profiles */}
              {activeStep === 2 && (
                <div className="page-section">
                  <h2>Review & Select Agent Profiles</h2>
                  <p>
                    Select the agents you want to create. You can customize their details before creation.
                  </p>
                  
                  <div className="select-all-container">
                    <label htmlFor="select-all" className="checkbox-label">
                      <input 
                        type="checkbox" 
                        id="select-all"
                        checked={selectedProfiles.length === formData.profiles.length}
                        onChange={handleSelectAll}
                        className="select-all-checkbox"
                      />
                      <span>Select All</span>
                    </label>
                  </div>
                  
                  <div className="agent-profiles-container">
                    {formData.profiles.map((profile, index) => (
                      <div 
                        key={index} 
                        className={`agent-profile-card ${selectedProfiles.includes(index) ? 'selected' : 'unselected'}`}
                        onClick={() => handleProfileSelection(index)}
                      >
                        <div className="profile-checkbox">
                          <input 
                            type="checkbox"
                            checked={selectedProfiles.includes(index)}
                            onChange={() => handleProfileSelection(index)}
                          />
                        </div>
                        <h3 className="profile-title">
                          {profile.name || `Agent ${index + 1}`}
                        </h3>
                        <div className="profile-description">
                          {profile.description || 'No description available.'}
                        </div>
                        <div className="profile-system-prompt">
                          {profile.system_prompt || 'No system prompt defined.'}
                        </div>
                      </div>
                    ))}
                  </div>
                  
                  <div className="button-container-between">
                    <button 
                      type="button" 
                      className="action-btn pause-resume-btn" 
                      onClick={prevStep}
                    >
                      <i className="fas fa-arrow-left"></i> Back
                    </button>
                    <button 
                      type="button" 
                      className="action-btn create-btn"
                      onClick={nextStep}
                      disabled={selectedProfiles.length === 0}
                    >
                      Continue <i className="fas fa-arrow-right"></i>
                    </button>
                  </div>
                </div>
              )}

              {/* Step 3: Configure Settings */}
              {activeStep === 3 && (
                <div className="page-section">
                  <h2>Configure Common Settings</h2>
                  <p>
                    Configure common settings for all selected agents. These settings will be applied to each agent.
                  </p>

                  <form id="group-settings-form" onSubmit={handleCreateGroup}>
                    <div className="info-message">
                      <i className="fas fa-info-circle info-message-icon"></i>
                      <span className="info-message-text">
                        Each agent will be created with its own name, description, and system prompt from the selected profiles.
                        The settings below will be applied to all agents.
                      </span>
                    </div>

                    {metadata ? (
                      <div>
                        <AgentForm
                          formData={formData}
                          setFormData={setFormData}
                          onSubmit={handleCreateGroup}
                          loading={loading}
                          submitButtonText="Create Group"
                          isGroupForm={true}
                          metadata={metadata}
                          noFormWrapper={true}
                        />
                        
                        <div className="button-container-between">
                          <button 
                            type="button" 
                            className="action-btn pause-resume-btn" 
                            onClick={prevStep}
                          >
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
                      </div>
                    ) : (
                      <div className="loading-container">
                        <div className="spinner"></div>
                      </div>
                    )}
                  </form>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default GroupCreate;
