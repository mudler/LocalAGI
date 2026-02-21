import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import hljs from 'highlight.js/lib/core';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/monokai.css';
import CollapsibleRawSections from '../components/CollapsibleRawSections';

hljs.registerLanguage('json', json);

function ObservableSummary({ observable }) {
  const creation = observable?.creation || {};
  const completion = observable?.completion || {};
  
  // Chat message summary
  let creationChatMsg = '';
  if (creation?.chat_completion_message && creation.chat_completion_message.content) {
    creationChatMsg = creation.chat_completion_message.content;
  } else {
    const messages = creation?.chat_completion_request?.messages;
    if (Array.isArray(messages) && messages.length > 0) {
      const lastMsg = messages[messages.length - 1];
      creationChatMsg = lastMsg?.content || '';
    }
  }
  
  if (typeof creationChatMsg === 'object') {
    creationChatMsg = 'Multimedia message';
  }
  
  // Function definition summary
  let creationFunctionDef = '';
  if (creation?.function_definition?.name) {
    creationFunctionDef = `Function: ${creation.function_definition.name}`;
  }
  
  // Function params summary
  let creationFunctionParams = '';
  if (creation?.function_params && Object.keys(creation.function_params).length > 0) {
    creationFunctionParams = `Params: ${JSON.stringify(creation.function_params)}`;
  }
  
  // Completion summary
  let completionChatMsg = '';
  let chatCompletion = completion?.chat_completion_response;
  
  if (!chatCompletion && Array.isArray(completion?.conversation) && completion.conversation.length > 0) {
    chatCompletion = { 
      choices: completion.conversation.map(m => ({ message: m }))
    };
  }
  
  if (chatCompletion && Array.isArray(chatCompletion.choices) && chatCompletion.choices.length > 0) {
    const lastChoice = chatCompletion.choices[chatCompletion.choices.length - 1];
    const toolCalls = lastChoice?.message?.tool_calls;
    
    if (Array.isArray(toolCalls) && toolCalls.length > 0) {
      const toolCallSummary = toolCalls.map(tc => {
        let args = '';
        if (tc.function && tc.function.arguments) {
          try {
            args = typeof tc.function.arguments === 'string' 
              ? tc.function.arguments 
              : JSON.stringify(tc.function.arguments);
          } catch (e) {
            args = '[Unserializable]';
          }
        }
        const toolName = tc.function?.name || tc.name || 'unknown';
        return `${toolName}(${args})`;
      }).join(', ');
      completionChatMsg = { toolCallSummary, message: lastChoice?.message?.content || '' };
    } else {
      completionChatMsg = lastChoice?.message?.content || '';
    }
  }
  
  // Action result summary
  let completionActionResult = '';
  if (completion?.action_result) {
    completionActionResult = String(completion.action_result).slice(0, 100);
  }
  
  // Agent state summary
  let completionAgentState = '';
  if (completion?.agent_state) {
    completionAgentState = JSON.stringify(completion.agent_state);
  }
  
  // Error summary
  let completionError = '';
  if (completion?.error) {
    completionError = completion.error;
  }
  
  // Filter result summary
  let completionFilter = '';
  if (completion?.filter_result) {
    if (completion.filter_result?.has_triggers && !completion.filter_result?.triggered_by) {
      completionFilter = 'Failed to match triggers';
    } else if (completion.filter_result?.triggered_by) {
      completionFilter = `Triggered by ${completion.filter_result.triggered_by}`;
    }
    if (completion?.filter_result?.failed_by) {
      completionFilter += `${completionFilter ? ', ' : ''}Failed by ${completion.filter_result.failed_by}`;
    }
  }
  
  // Check if any summary exists
  if (!creationChatMsg && !creationFunctionDef && !creationFunctionParams &&
      !completionChatMsg && !completionActionResult && 
      !completionAgentState && !completionError && !completionFilter) {
    return null;
  }
  
  return (
    <div className="observable-summary">
      {creationChatMsg && (
        <div className="observable-summary-item creation" title={creationChatMsg}>
          <i className="fas fa-comment-dots" />
          <span>{creationChatMsg}</span>
        </div>
      )}
      {creationFunctionDef && (
        <div className="observable-summary-item creation" title={creationFunctionDef}>
          <i className="fas fa-code" />
          <span>{creationFunctionDef}</span>
        </div>
      )}
      {creationFunctionParams && (
        <div className="observable-summary-item creation" title={creationFunctionParams}>
          <i className="fas fa-sliders-h" />
          <span>{creationFunctionParams}</span>
        </div>
      )}
      {completionChatMsg && typeof completionChatMsg === 'object' && completionChatMsg.toolCallSummary && (
        <div className="observable-summary-item tool-call" title={completionChatMsg.toolCallSummary}>
          <i className="fas fa-tools" />
          <span>{completionChatMsg.toolCallSummary}</span>
        </div>
      )}
      {completionChatMsg && (typeof completionChatMsg === 'string' || completionChatMsg.message) && (
        <div className="observable-summary-item completion" title={typeof completionChatMsg === 'string' ? completionChatMsg : completionChatMsg.message}>
          <i className="fas fa-robot" />
          <span>{typeof completionChatMsg === 'string' ? completionChatMsg : completionChatMsg.message}</span>
        </div>
      )}
      {completionActionResult && (
        <div className="observable-summary-item tool-call" title={completionActionResult}>
          <i className="fas fa-bolt" />
          <span>{completionActionResult}</span>
        </div>
      )}
      {completionAgentState && (
        <div className="observable-summary-item completion" title={completionAgentState}>
          <i className="fas fa-brain" />
          <span>{completionAgentState}</span>
        </div>
      )}
      {completionError && (
        <div className="observable-summary-item error" title={completionError}>
          <i className="fas fa-exclamation-triangle" />
          <span>{completionError}</span>
        </div>
      )}
      {completionFilter && (
        <div className="observable-summary-item completion" title={completionFilter}>
          <i className="fas fa-shield-alt" />
          <span>{completionFilter}</span>
        </div>
      )}
    </div>
  );
}

function ObservableCard({ observable, isNested = false }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [expandedChildren, setExpandedChildren] = useState(new Map());
  const childKey = isNested ? `child-${observable.id}` : observable.id;
  
  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };
  
  const toggleChild = (childId) => {
    setExpandedChildren(prev => {
      const newMap = new Map(prev);
      newMap.set(childId, !prev.get(childId));
      return newMap;
    });
  };
  
  const isComplete = !!observable.completion;
  const hasChildren = observable.children && observable.children.length > 0;
  
  return (
    <div className={`observable-card ${isNested ? 'nested' : ''}`}>
      <div className="observable-header" onClick={toggleExpand}>
        <div className="observable-title">
          <div className="observable-icon">
            <i className={`fas fa-${observable.icon || 'robot'}`} />
          </div>
          <div className="observable-info">
            <div className="observable-name">
              {observable.name}
              <span className="observable-id">#{observable.id}</span>
            </div>
            <ObservableSummary observable={observable} />
          </div>
        </div>
        <div className="observable-actions">
          {!isComplete && <div className="spinner" />}
          <i className={`fas fa-chevron-down observable-toggle ${isExpanded ? 'expanded' : ''}`} />
        </div>
      </div>
      
      {isExpanded && (
        <div className="observable-content">
          {hasChildren && (
            <div className="observable-children">
              <h4 style={{ marginBottom: '0.75rem', color: 'var(--color-text-secondary)', fontSize: '0.9rem' }}>
                Nested Observables
              </h4>
              {observable.children.map(child => (
                <div key={child.id} className="observable-card nested">
                  <div className="observable-header" onClick={() => toggleChild(child.id)}>
                    <div className="observable-title">
                      <div className="observable-icon">
                        <i className={`fas fa-${child.icon || 'robot'}`} />
                      </div>
                      <div className="observable-info">
                        <div className="observable-name">
                          {child.name}
                          <span className="observable-id">#{child.id}</span>
                        </div>
                        <ObservableSummary observable={child} />
                      </div>
                    </div>
                    <div className="observable-actions">
                      {!child.completion && <div className="spinner" />}
                      <i className={`fas fa-chevron-down observable-toggle ${expandedChildren.get(child.id) ? 'expanded' : ''}`} />
                    </div>
                  </div>
                  {expandedChildren.get(child.id) && (
                    <div className="observable-content">
                      <CollapsibleRawSections container={child} />
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
          <CollapsibleRawSections container={observable} />
        </div>
      )}
    </div>
  );
}

function AgentStatus() {
  const { name } = useParams();
  const [showStatus, setShowStatus] = useState(false);
  const [statusData, setStatusData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [observableMap, setObservableMap] = useState({});
  const [observableTree, setObservableTree] = useState([]);
  const [clearLoading, setClearLoading] = useState(false);

  // Update document title
  useEffect(() => {
    if (name) {
      document.title = `${name} - Status - LocalAGI`;
    }
    return () => {
      document.title = 'LocalAGI';
    };
  }, [name]);

  // Fetch initial status data and setup SSE
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

    const fetchObservables = async () => {
      try {
        const response = await fetch(`/api/agent/${name}/observables`);
        if (!response.ok) return;
        const data = await response.json();
        if (Array.isArray(data.History)) {
          const map = {};
          data.History.forEach(obs => { map[obs.id] = obs; });
          setObservableMap(map);
          setObservableTree(buildObservableTree(map));
        }
      } catch (err) {
        // Ignore errors for now
      }
    };
    fetchObservables();

    // Setup SSE connection
    const sse = new EventSource(`/sse/${name}`);

    sse.addEventListener('observable_update', (event) => {
      const data = JSON.parse(event.data);
      setObservableMap(prevMap => {
        const prev = prevMap[data.id] || {};
        const updated = { ...data, ...prev };
        if (data.creation) updated.creation = data.creation;
        if (data.completion) updated.completion = data.completion;
        if ((data.progress?.length ?? 0) > (prev.progress?.length ?? 0))
          updated.progress = data.progress;
        if (data.parent_id && !prevMap[data.parent_id])
          prevMap[data.parent_id] = { id: data.parent_id, name: "unknown" };
        const newMap = { ...prevMap, [data.id]: updated };
        setObservableTree(buildObservableTree(newMap));
        return newMap;
      });
    });

    sse.addEventListener('status', (event) => {
      const status = event.data;
      setStatusData(prev => {
        if (!prev || typeof prev !== 'object') {
          return { History: [status] };
        }
        if (!Array.isArray(prev.History)) {
          return { ...prev, History: [status] };
        }
        return { ...prev, History: [...prev.History, status] };
      });
    });

    sse.onerror = (err) => {
      console.error('SSE connection error:', err);
    };

    return () => {
      sse.close();
    };
  }, [name]);

  const handleClearObservables = async () => {
    if (clearLoading) return;
    setClearLoading(true);
    try {
      const resp = await fetch(`/api/agent/${name}/observables`, { method: 'DELETE' });
      if (!resp.ok) {
        console.error('Failed to clear observables, status:', resp.status);
      } else {
        setObservableMap({});
        setObservableTree([]);
      }
    } catch (e) {
      console.error('Error clearing observables:', e);
    } finally {
      setClearLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="agent-status-container">
        <div className="loading-container">
          <div className="loader" />
          <p>Loading agent status...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="agent-status-container">
        <div className="error-container">
          <h2><i className="fas fa-exclamation-triangle" /> Error</h2>
          <p>{error}</p>
          <Link to="/agents" className="back-btn">
            <i className="fas fa-arrow-left" /> Back to Agents
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="agent-status-container">
      {/* Page Header */}
      <header className="page-header">
        <div className="header-title-section">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <Link to="/agents" className="back-link" title="Back to agents">
              <i className="fas fa-arrow-left" />
            </Link>
            <h1>{name}</h1>
          </div>
          <p className="agent-subtitle">Monitor agent activity and observables in real-time</p>
        </div>
      </header>

      {/* Current Status Section */}
      {statusData && (
        <div className="status-section">
          <div 
            className="status-section-header"
            onClick={() => setShowStatus(!showStatus)}
          >
            <h2>
              <i className="fas fa-chart-line" />
              Current Status
            </h2>
            <i className={`fas fa-chevron-down status-section-toggle ${showStatus ? 'expanded' : ''}`} />
          </div>
          <p className="status-section-description">
            Real-time summary of the agent&apos;s thoughts and actions
          </p>
          {showStatus && (
            <div style={{ marginTop: '1rem' }}>
              {(Array.isArray(statusData?.History) && statusData.History.length === 0) && (
                <div style={{ color: 'var(--color-text-muted)', textAlign: 'center', padding: '2rem' }}>
                  <i className="fas fa-inbox" style={{ fontSize: '2rem', marginBottom: '0.5rem', display: 'block' }} />
                  No status history available
                </div>
              )}
              {Array.isArray(statusData?.History) && statusData.History.map((item, idx) => (
                <div key={idx} className="card" style={{ marginBottom: '0.75rem' }}>
                  {typeof item === 'string'
                    ? item.replace(/<br\s*\/?>/gi, '\n')
                    : JSON.stringify(item, null, 2)}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Observable Updates Section */}
      {observableTree.length > 0 && (
        <div className="status-section">
          <div className="status-section-header">
            <h2>
              <i className="fas fa-eye" />
              Observable Updates
            </h2>
            <button
              className="action-btn delete-btn"
              onClick={handleClearObservables}
              disabled={clearLoading}
              style={{ fontSize: '0.85rem', padding: '0.4rem 0.75rem' }}
            >
              {clearLoading ? (
                <><i className="fas fa-spinner fa-spin" /> Clearing...</>
              ) : (
                <><i className="fas fa-trash" /> Clear History</>
              )}
            </button>
          </div>
          <p className="status-section-description">
            Drill down into agent activities triggered by connectors
          </p>
          <div style={{ marginTop: '1rem' }}>
            {observableTree.map((observable) => (
              <ObservableCard key={observable.id} observable={observable} />
            ))}
          </div>
        </div>
      )}

      {/* Empty State */}
      {observableTree.length === 0 && statusData && (
        <div className="status-section">
          <div style={{ textAlign: 'center', padding: '3rem', color: 'var(--color-text-muted)' }}>
            <i className="fas fa-satellite-dish" style={{ fontSize: '3rem', marginBottom: '1rem', display: 'block', opacity: 0.5 }} />
            <h3 style={{ marginBottom: '0.5rem' }}>No Observables Yet</h3>
            <p>Connectors will create observables when the agent is triggered.</p>
          </div>
        </div>
      )}
    </div>
  );
}

export default AgentStatus;
