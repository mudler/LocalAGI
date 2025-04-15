import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import hljs from 'highlight.js/lib/core';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/monokai.css';

hljs.registerLanguage('json', json);

function AgentStatus() {
  const [showStatus, setShowStatus] = useState(true);
  const { name } = useParams();
  const [statusData, setStatusData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [_eventSource, setEventSource] = useState(null);
  // Store all observables by id
  const [observableMap, setObservableMap] = useState({});
  const [observableTree, setObservableTree] = useState([]);
  const [expandedCards, setExpandedCards] = useState(new Map());

  // Update document title
  useEffect(() => {
    if (name) {
      document.title = `Agent Status: ${name} - LocalAGI`;
    }
    return () => {
      document.title = 'LocalAGI'; // Reset title when component unmounts
    };
  }, [name]);

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

    // Helper to build observable tree from map
    function buildObservableTree(map) {
      const nodes = Object.values(map);
      const nodeMap = {};
      nodes.forEach(node => { nodeMap[node.id] = { ...node, children: [] }; });
      const roots = [];
      nodes.forEach(node => {
        if (!node.parent_id) {
          roots.push(nodeMap[node.id]);
        } else if (nodeMap[node.parent_id]) {
          nodeMap[node.parent_id].children.push(nodeMap[node.id]);
        }
      });
      return roots;
    }

    // Fetch initial observable history
    const fetchObservables = async () => {
      try {
        const response = await fetch(`/api/agent/${name}/observables`);
        if (!response.ok) return;
        const data = await response.json();
        if (Array.isArray(data.History)) {
          const map = {};
          data.History.forEach(obs => {
            map[obs.id] = obs;
          });
          setObservableMap(map);
          setObservableTree(buildObservableTree(map));
        }
      } catch (err) {
        // Ignore errors for now
      }
    };
    fetchObservables();

    // Setup SSE connection for live updates
    const sse = new EventSource(`/sse/${name}`);
    setEventSource(sse);

    sse.addEventListener('observable_update', (event) => {
      const data = JSON.parse(event.data);
      console.log(data);
      setObservableMap(prevMap => {
        const prev = prevMap[data.id] || {};
        const updated = {
          ...prev,
          ...data,
          creation: data.creation,
          progress: data.progress,
          completion: data.completion,
          // children are always built client-side
        };
        const newMap = { ...prevMap, [data.id]: updated };
        setObservableTree(buildObservableTree(newMap));
        return newMap;
      });
    });

    // Listen for status events and append to statusData.History
    sse.addEventListener('status', (event) => {
      const status = event.data;
      setStatusData(prev => {
        // If prev is null, start a new object
        if (!prev || typeof prev !== 'object') {
          return { History: [status] };
        }
        // If History not present, add it
        if (!Array.isArray(prev.History)) {
          return { ...prev, History: [status] };
        }
        // Otherwise, append
        return { ...prev, History: [...prev.History, status] };
      });
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
      <div>
        <div></div>
        <p>Loading agent status...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <h2>Error</h2>
        <p>{error}</p>
        <Link to="/agents">
          <i className="fas fa-arrow-left"></i> Back to Agents
        </Link>
      </div>
    );
  }

  return (
    <div>
      <h1>Agent Status: {name}</h1>
      <div style={{ color: '#aaa', fontSize: 16, marginBottom: 18 }}>
        See what the agent is doing and thinking
      </div>
      {error && (
        <div>
          {error}
        </div>
      )}
      {loading && <div>Loading...</div>}
      {statusData && (
        <div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', userSelect: 'none' }}
              onClick={() => setShowStatus(prev => !prev)}>
              <h2 style={{ margin: 0 }}>Current Status</h2>
              <i
                className={`fas fa-chevron-${showStatus ? 'up' : 'down'}`}
                style={{ color: 'var(--primary)', marginLeft: 12 }}
                title={showStatus ? 'Collapse' : 'Expand'}
              />
            </div>
            <div style={{ color: '#aaa', fontSize: 14, margin: '5px 0 10px 2px' }}>
              Summary of the agent's thoughts and actions
            </div>
            {showStatus && (
              <div style={{ marginTop: 10 }}>
                {(Array.isArray(statusData?.History) && statusData.History.length === 0) && (
                  <div style={{ color: '#aaa' }}>No status history available.</div>
                )}
                {Array.isArray(statusData?.History) && statusData.History.map((item, idx) => (
                  <div key={idx} style={{
                    background: '#222',
                    border: '1px solid #444',
                    borderRadius: 8,
                    padding: '12px 16px',
                    marginBottom: 10,
                    whiteSpace: 'pre-line',
                    fontFamily: 'inherit',
                    fontSize: 15,
                    color: '#eee',
                  }}>
                    {/* Replace <br> tags with newlines, then render as pre-line */}
                    {typeof item === 'string'
                      ? item.replace(/<br\s*\/?>/gi, '\n')
                      : JSON.stringify(item)}
                  </div>
                ))}
              </div>
            )}
          </div>
          {observableTree.length > 0 && (
            <div>
              <h2>Observable Updates</h2>
              <div style={{ color: '#aaa', fontSize: 14, margin: '5px 0 10px 2px' }}>
                Drill down into what the agent is doing and thinking when activated by a connector
              </div>
              <div>
                {observableTree.map((container, idx) => (
                  <div key={container.id || idx} className='card' style={{ marginBottom: '1em' }}>
                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }}
                        onClick={() => {
                          const newExpanded = !expandedCards.get(container.id);
                          setExpandedCards(new Map(expandedCards).set(container.id, newExpanded));
                        }}
                      >
                        <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
  <i className={`fas fa-${container.icon || 'robot'}`} style={{ verticalAlign: '-0.125em' }}></i>
  <span>
    <span className='stat-label'>{container.name}</span>#<span className='stat-label'>{container.id}</span>
  </span>
</div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <i
                            className={`fas fa-chevron-${expandedCards.get(container.id) ? 'up' : 'down'}`}
                            style={{ color: 'var(--primary)' }}
                            title='Toggle details'
                          />
                          {!container.completion && (
                            <div className='spinner' />
                          )}
                        </div>
                      </div>
                      <div style={{ display: expandedCards.get(container.id) ? 'block' : 'none' }}>
                        {container.children && container.children.length > 0 && (

                          <div style={{ marginLeft: '2em', marginTop: '1em' }}>
                            <h4>Nested Observables</h4>
                            {container.children.map(child => {
                              const childKey = `child-${child.id}`;
                              const isExpanded = expandedCards.get(childKey);
                              return (
                                <div key={`${container.id}-child-${child.id}`} className='card' style={{ background: '#222', marginBottom: '0.5em' }}>
                                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }}
                                    onClick={() => {
                                      const newExpanded = !expandedCards.get(childKey);
                                      setExpandedCards(new Map(expandedCards).set(childKey, newExpanded));
                                    }}
                                  >
                                    <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
  <i className={`fas fa-${child.icon || 'robot'}`} style={{ verticalAlign: '-0.125em' }}></i>
  <span>
    <span className='stat-label'>{child.name}</span>#<span className='stat-label'>{child.id}</span>
  </span>
</div>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                      <i
                                        className={`fas fa-chevron-${isExpanded ? 'up' : 'down'}`}
                                        style={{ color: 'var(--primary)' }}
                                        title='Toggle details'
                                      />
                                      {!child.completion && (
                                        <div className='spinner' />
                                      )}
                                    </div>
                                  </div>
                                  <div style={{ display: isExpanded ? 'block' : 'none' }}>
                                    {child.creation && (
                                      <div>
                                        <h5>Creation:</h5>
                                        <pre className="hljs"><code>
                                          <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(child.creation || {}, null, 2), { language: 'json' }).value }}></div>
                                        </code></pre>
                                      </div>
                                    )}
                                    {child.progress && child.progress.length > 0 && (
                                      <div>
                                        <h5>Progress:</h5>
                                        <pre className="hljs"><code>
                                          <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(child.progress || {}, null, 2), { language: 'json' }).value }}></div>
                                        </code></pre>
                                      </div>
                                    )}
                                    {child.completion && (
                                      <div>
                                        <h5>Completion:</h5>
                                        <pre className="hljs"><code>
                                          <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(child.completion || {}, null, 2), { language: 'json' }).value }}></div>
                                        </code></pre>
                                      </div>
                                    )}
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        )}
                        {container.creation && (
                          <div>
                            <h4>Creation:</h4>
                            <pre className="hljs"><code>
                              <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(container.creation || {}, null, 2), { language: 'json' }).value }}></div>
                            </code></pre>
                          </div>
                        )}
                        {container.progress && container.progress.length > 0 && (
                          <div>
                            <h4>Progress:</h4>
                            <pre className="hljs"><code>
                              <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(container.progress || {}, null, 2), { language: 'json' }).value }}></div>
                            </code></pre>
                          </div>
                        )}
                        {container.completion && (
                          <div>
                            <h4>Completion:</h4>
                            <pre className="hljs"><code>
                              <div dangerouslySetInnerHTML={{ __html: hljs.highlight(JSON.stringify(container.completion || {}, null, 2), { language: 'json' }).value }}></div>
                            </code></pre>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default AgentStatus;
