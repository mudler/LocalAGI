import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';

function AgentStatus() {
  const { name } = useParams();
  const [statusData, setStatusData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [eventSource, setEventSource] = useState(null);
  const [liveUpdates, setLiveUpdates] = useState([]);

  // Fetch initial status data
  useEffect(() => {
    const fetchStatusData = async () => {
      try {
        const response = await fetch(`/api/agent/${name}/status`);
        if (!response.ok) {
          throw new Error(`Server responded with status: ${response.status}`);
        }
        const data = await response.json();
        setStatusData(data);
      } catch (err) {
        console.error('Error fetching agent status:', err);
        setError(`Failed to load status for agent "${name}": ${err.message}`);
      } finally {
        setLoading(false);
      }
    };

    fetchStatusData();

    // Setup SSE connection for live updates
    const sse = new EventSource(`/sse/${name}`);
    setEventSource(sse);

    sse.addEventListener('status', (event) => {
      try {
        const data = JSON.parse(event.data);
        setLiveUpdates(prev => [data, ...prev.slice(0, 19)]); // Keep last 20 updates
      } catch (err) {
        console.error('Error parsing SSE data:', err);
      }
    });

    sse.onerror = (err) => {
      console.error('SSE connection error:', err);
    };

    // Cleanup on unmount
    return () => {
      if (sse) {
        sse.close();
      }
    };
  }, [name]);

  // Helper function to safely convert any value to a displayable string
  const formatValue = (value) => {
    if (value === null || value === undefined) {
      return 'N/A';
    }
    
    if (typeof value === 'object') {
      try {
        return JSON.stringify(value, null, 2);
      } catch (err) {
        return '[Complex Object]';
      }
    }
    
    return String(value);
  };

  if (loading) {
    return (
      <div className="loading-container">
        <div className="loader"></div>
        <p>Loading agent status...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="error-container">
        <h2>Error</h2>
        <p>{error}</p>
        <Link to="/agents" className="back-btn">
          <i className="fas fa-arrow-left"></i> Back to Agents
        </Link>
      </div>
    );
  }

  // Combine live updates with history
  const allUpdates = [...liveUpdates, ...(statusData?.History || [])];

  return (
    <div className="agent-status-container">
      <header className="page-header">
        <div className="header-content">
          <h1>
            <Link to="/agents" className="back-link">
              <i className="fas fa-arrow-left"></i>
            </Link>
            Agent Status: {name}
          </h1>
        </div>
      </header>

      <div className="chat-container bg-gray-800 shadow-lg rounded-lg">
        {/* Chat Messages */}
        <div className="chat-messages p-4">
          {allUpdates.length > 0 ? (
            allUpdates.map((item, index) => (
              <div key={index} className="status-item mb-4">
                <div className="bg-gray-700 p-4 rounded-lg">
                  <h2 className="text-sm font-semibold mb-2">Agent Action:</h2>
                  <div className="status-details">
                    <div className="status-row">
                      <span className="status-label">Result:</span>
                      <span className="status-value">{formatValue(item.Result)}</span>
                    </div>
                    <div className="status-row">
                      <span className="status-label">Action:</span>
                      <span className="status-value">{formatValue(item.Action)}</span>
                    </div>
                    <div className="status-row">
                      <span className="status-label">Parameters:</span>
                      <span className="status-value pre-wrap">{formatValue(item.Params)}</span>
                    </div>
                    {item.Reasoning && (
                      <div className="status-row">
                        <span className="status-label">Reasoning:</span>
                        <span className="status-value reasoning">{formatValue(item.Reasoning)}</span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))
          ) : (
            <div className="no-status-data">
              <p>No status data available for this agent.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default AgentStatus;
