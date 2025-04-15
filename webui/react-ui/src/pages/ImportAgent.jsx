import { useState, useEffect } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { agentApi } from "../utils/api";

function ImportAgent() {
  const navigate = useNavigate();
  const { showToast } = useOutletContext();
  const [file, setFile] = useState(null);
  const [loading, setLoading] = useState(false);

  // Update document title
  useEffect(() => {
    document.title = "Import Agent - LocalAGI";
    return () => {
      document.title = "LocalAGI";
    };
  }, []);

  const handleFileChange = (e) => {
    const selectedFile = e.target.files[0];
    if (selectedFile) {
      setFile(selectedFile);
    }
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
      showToast("Failed to import agent", "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        {/* Header */}
        <div
          className="import-agent-header"
          style={{
            display: "flex",
            alignItems: "center",
            gap: 18,
            marginBottom: "2.5rem",
          }}
        >
          <i
            className="fas fa-file-import"
            style={{ fontSize: 32, color: "var(--primary)" }}
          />
          <div>
            <div style={{ fontSize: "2rem", fontWeight: 700, color: "#222" }}>
              Import Agent
            </div>
            <div
              className="section-description"
              style={{
                color: "var(--text-light)",
                fontSize: "1.1rem",
                marginTop: 2,
              }}
            >
              Upload a previously exported agent configuration file to restore
              or transfer an agent.
            </div>
          </div>
        </div>

        {/* Import Form */}
        <div className="section-box" style={{ maxWidth: 720 }}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleImport();
            }}
            style={{ display: "flex", flexDirection: "column", gap: 24 }}
          >
            <label htmlFor="import-file" style={{ fontWeight: 500 }}>
              Select agent file (.json)
            </label>
            <input
              id="import-file"
              type="file"
              accept=".json"
              onChange={handleFileChange}
              disabled={loading}
              style={{
                padding: 10,
                border: "1px solid var(--border-color)",
                borderRadius: 5,
                fontSize: "1rem",
                background: "#fff",
              }}
            />
            <div
              style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}
            >
              <button
                type="button"
                className="action-btn"
                style={{ background: "#f6f8fa", color: "var(--primary)" }}
                onClick={() => navigate("/agents")}
                disabled={loading}
              >
                <i className="fas fa-times"></i> Cancel
              </button>
              <button type="submit" className="action-btn" disabled={loading}>
                <i className="fas fa-file-import"></i> Import Agent
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

export default ImportAgent;
