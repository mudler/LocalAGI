import { useState, useEffect } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { agentApi } from "../utils/api";
import Header from "../components/Header";

function ImportAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [file, setFile] = useState(null);
  const [loading, setLoading] = useState(false);
  const [dragOver, setDragOver] = useState(false);

  useEffect(() => {
    document.title = "Import Agent - LocalAGI";
    return () => {
      document.title = "LocalAGI";
    };
  }, []);

  const handleFileChange = (selectedFile) => {
    if (selectedFile && selectedFile.type === "application/json") {
      setFile(selectedFile);
    } else {
      showToast("Please select a valid JSON file", "error");
    }
  };

  const handleInputChange = (e) => {
    const selectedFile = e.target.files[0];
    if (selectedFile) {
      handleFileChange(selectedFile);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    setDragOver(false);
    const droppedFile = e.dataTransfer.files[0];
    if (droppedFile) {
      handleFileChange(droppedFile);
    }
  };

  const handleDragOver = (e) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = (e) => {
    e.preventDefault();
    setDragOver(false);
  };

  const handleImport = async () => {
    if (!file) {
      showToast("Please select a file to import", "error");
      return;
    }

    setLoading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      await agentApi.importAgent(formData);
      showToast("Agent imported successfully", "success");
      navigate("/agents");
    } catch (err) {
      console.error("Error importing agent:", err);
      showToast("Failed to import agent", "error");
    } finally {
      setLoading(false);
    }
  };

  const backButton = (
    <button
      className="action-btn pause-resume-btn"
      onClick={() => navigate("/agents")}
      disabled={loading}
    >
      <i className="fas fa-arrow-left"></i> Back to Agents
    </button>
  );

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Import Agent"
            description="Upload a previously exported agent configuration file to restore or transfer an agent."
          />
          <div className="header-right">{backButton}</div>
        </div>

        <div className="section-box" style={{ maxWidth: 720 }}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleImport();
            }}
            style={{ display: "flex", flexDirection: "column", gap: 24 }}
          >
            <label style={{ fontWeight: 500, marginBottom: 8 }}>
              Select agent file (.json)
            </label>
            
            <div
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => document.getElementById('file-input').click()}
              style={{
                border: `2px dashed ${dragOver ? 'var(--primary)' : 'var(--border)'}`,
                borderRadius: 12,
                padding: '2rem',
                textAlign: 'center',
                cursor: 'pointer',
                backgroundColor: dragOver ? 'rgba(30, 84, 191, 0.05)' : '#f9fafb',
                transition: 'all 0.2s ease',
                position: 'relative',
                minHeight: 160,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 16
              }}
            >
              <img 
                src="/app/features/dashed-upload.svg" 
                alt="Upload" 
                style={{ 
                  width: 48, 
                  height: 48, 
                  opacity: dragOver ? 0.9 : 0.7,
                  transition: 'opacity 0.2s ease'
                }} 
              />
              
              {file ? (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
                  <div style={{ 
                    fontSize: '1rem', 
                    fontWeight: 500, 
                    color: 'var(--primary)' 
                  }}>
                    <i className="fas fa-file-check" style={{ marginRight: 8 }}></i>
                    {file.name}
                  </div>
                  <div style={{ 
                    fontSize: '0.875rem', 
                    color: 'var(--text-light)' 
                  }}>
                    {(file.size / 1024).toFixed(1)} KB
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setFile(null);
                    }}
                    style={{
                      background: 'none',
                      border: 'none',
                      color: 'var(--danger)',
                      cursor: 'pointer',
                      fontSize: '0.875rem',
                      marginTop: 4
                    }}
                  >
                    <i className="fas fa-times"></i> Remove
                  </button>
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
                  <div style={{ 
                    fontSize: '1.1rem', 
                    fontWeight: 500, 
                    color: dragOver ? 'var(--primary)' : 'var(--text)' 
                  }}>
                    {dragOver ? 'Drop your file here' : 'Drag and drop your agent file'}
                  </div>
                  <div style={{ 
                    fontSize: '0.9rem', 
                    color: 'var(--text-light)' 
                  }}>
                    or click to browse
                  </div>
                  <div style={{
                    fontSize: '0.8rem',
                    color: 'var(--text-lighter)',
                    marginTop: 4
                  }}>
                    Supports JSON files only
                  </div>
                </div>
              )}
              
              <input
                id="file-input"
                type="file"
                accept=".json"
                onChange={handleInputChange}
                disabled={loading}
                style={{ display: 'none' }}
              />
            </div>

            <div
              style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}
            >
              <button
                type="submit"
                className="action-btn"
                disabled={loading || !file}
              >
                {loading ? (
                  <>
                    <i className="fas fa-spinner fa-spin"></i> Importing...
                  </>
                ) : (
                  <>
                    <i className="fas fa-file-import"></i> Import Agent
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

export default ImportAgent;
