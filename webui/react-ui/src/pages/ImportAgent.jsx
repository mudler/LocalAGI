import { useState, useEffect } from 'react';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { agentApi } from '../utils/api';
import AgentForm from '../components/AgentForm';

function ImportAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [loading, setLoading] = useState(false);
  const [metadata, setMetadata] = useState(null);
  const [formData, setFormData] = useState({});
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    document.title = 'Import Agent - LocalAGI';
    return () => {
      document.title = 'LocalAGI';
    };
  }, []);

  // Fetch metadata on mount (needed for AgentForm)
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

  const handleFileSelected = (file) => {
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const parsed = JSON.parse(e.target.result);
        setFormData(parsed);
        setShowForm(true);
      } catch (err) {
        showToast('Failed to parse JSON file: ' + err.message, 'error');
      }
    };
    reader.onerror = () => {
      showToast('Failed to read file', 'error');
    };
    reader.readAsText(file);
  };

  const handleFileChange = (e) => {
    handleFileSelected(e.target.files[0]);
  };

  const handleDrop = (e) => {
    e.preventDefault();
    handleFileSelected(e.dataTransfer.files[0]);
  };

  const handleBack = () => {
    setShowForm(false);
    setFormData({});
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!formData.name || !formData.name.trim()) {
      showToast('Agent name is required', 'error');
      return;
    }

    setLoading(true);
    try {
      await agentApi.createAgent(formData);
      showToast(`Agent "${formData.name}" imported successfully`, 'success');
      navigate(`/settings/${formData.name}`);
    } catch (err) {
      showToast(`Error importing agent: ${err.message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  if (showForm) {
    return (
      <div className="create-agent-container">
        <header className="page-header">
          <h1>
            <i className="fas fa-upload"></i> Import Agent
          </h1>
        </header>

        <div className="create-agent-content">
          <div className="section-box">
            <div style={{ marginBottom: '1rem' }}>
              <button className="action-btn" onClick={handleBack}>
                <i className="fas fa-arrow-left"></i> Back to File Selection
              </button>
            </div>
            <h2>
              <i className="fas fa-robot"></i> Review & Edit Agent Configuration
            </h2>

            <AgentForm
              formData={formData}
              setFormData={setFormData}
              onSubmit={handleSubmit}
              loading={loading}
              submitButtonText="Import Agent"
              isEdit={false}
              metadata={metadata}
            />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="import-agent-container">
      <header className="page-header">
        <h1>
          <i className="fas fa-upload"></i> Import Agent
        </h1>
      </header>

      <div className="import-agent-content">
        <div className="section-box">
          <div className="file-dropzone" onDrop={handleDrop} onDragOver={(e) => e.preventDefault()}>
            <div className="dropzone-content">
              <i className="fas fa-cloud-upload-alt"></i>
              <h2>Drop your agent JSON file here</h2>
              <p>or</p>
              <label htmlFor="fileInput" className="action-btn">
                <i className="fas fa-folder-open"></i> Select File
              </label>
              <input
                type="file"
                id="fileInput"
                accept=".json"
                onChange={handleFileChange}
                style={{ display: 'none' }}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ImportAgent;
