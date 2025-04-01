import { useState } from 'react';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { agentApi } from '../utils/api';

function ImportAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [file, setFile] = useState(null);
  const [loading, setLoading] = useState(false);

  const handleFileChange = (e) => {
    const selectedFile = e.target.files[0];
    if (selectedFile) {
      setFile(selectedFile);
    }
  };

  const handleImport = async () => {
    if (!file) {
      showToast('Please select a file to import', 'error');
      return;
    }

    setLoading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);

      await agentApi.importAgentConfig(formData);
      showToast('Agent imported successfully', 'success');
      navigate('/agents');
    } catch (err) {
      showToast(`Error importing agent: ${err.message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="import-agent-container">
      <header className="page-header">
        <h1>
          <i className="fas fa-upload"></i> Import Agent
        </h1>
      </header>

      <div className="import-agent-content">
        <div className="section-box">
          <div className="file-dropzone" onDrop={(e) => {
            e.preventDefault();
            const droppedFile = e.dataTransfer.files[0];
            if (droppedFile) {
              setFile(droppedFile);
            }
          }}
          onDragOver={(e) => e.preventDefault()}>
            <div className="dropzone-content">
              <i className="fas fa-cloud-upload-alt"></i>
              <h2>Drop your agent file here</h2>
              <p>or</p>
              <label htmlFor="fileInput" className="file-button">
                <i className="fas fa-folder-open"></i> Select File
              </label>
              <input
                type="file"
                id="fileInput"
                accept=".json,.yaml,.yml"
                onChange={handleFileChange}
                style={{ display: 'none' }}
              />
            </div>
          </div>

          {file && (
            <div className="selected-file-info">
              <p>Selected file: {file.name}</p>
              <button
                className="import-button"
                onClick={handleImport}
                disabled={loading}
              >
                {loading ? (
                  <>
                    <i className="fas fa-spinner fa-spin"></i>
                    Importing...
                  </>
                ) : (
                  <>
                    <i className="fas fa-upload"></i>
                    Import Agent
                  </>
                )}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default ImportAgent;
