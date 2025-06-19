import { useState, useEffect } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { agentApi } from "../utils/api";
import AgentForm from "../components/AgentForm";
import Header from "../components/Header";

function CreateAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(false);
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({});

  useEffect(() => {
    document.title = "Create Agent - LocalAGI";
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
        // Continue without metadata, the form will use default fields
      }
    };

    fetchMetadata();
  }, []);

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

  // Handle form submission
  const handleSubmit = async (data) => {
    console.log("[CreateAgent] Submitting agent with full data:", data); // DEBUG LOG
    setLoading(true);
    try {
      await agentApi.createAgent(data);
      showToast && showToast("Agent created successfully!", "success");
      navigate("/agents");
    } catch (error) {
      showToast && showToast("Failed to create agent", "error");
      console.error("Error creating agent:", error);
    } finally {
      setLoading(false);
    }
  };

  const backButton = (
    <button
      className="action-btn pause-resume-btn"
      onClick={() => navigate("/agents")}
    >
      <i className="fas fa-arrow-left"></i> Back to Agents
    </button>
  );

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Create Agent"
            description="Fill out the form below to create a new agent. You can customize its configuration and capabilities."
          />
          <div className="header-right">{backButton}</div>
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
