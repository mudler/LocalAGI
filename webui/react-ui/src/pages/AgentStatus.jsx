import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import CollapsibleRawSections from '../components/CollapsibleRawSections';
import Header from '../components/Header';
import hljs from 'highlight.js/lib/core';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/atom-one-dark.css';
import { useAgent } from '../hooks/useAgent';

hljs.registerLanguage('json', json);

function ObservableSummary({ observable }) {
  // --- CREATION SUMMARIES ---
  const creation = observable?.creation || {};
  // ChatCompletionRequest summary
  let creationChatMsg = '';
  // Prefer chat_completion_message if present (for jobs/top-level containers)
  if (creation?.chat_completion_message && creation.chat_completion_message.content) {
    creationChatMsg = creation.chat_completion_message.content;
  } else {
    const messages = creation?.chat_completion_request?.messages;
    if (Array.isArray(messages) && messages.length > 0) {
      const lastMsg = messages[messages.length - 1];
      creationChatMsg = lastMsg?.content || '';
    }
  }
  // FunctionDefinition summary
  let creationFunctionDef = '';
  if (creation?.function_definition?.name) {
    creationFunctionDef = `Function: ${creation.function_definition.name}`;
  }
  // FunctionParams summary
  let creationFunctionParams = '';
  if (creation?.function_params && Object.keys(creation.function_params).length > 0) {
    creationFunctionParams = `Params: ${JSON.stringify(creation.function_params)}`;
  }

  // --- COMPLETION SUMMARIES ---
  const completion = observable?.completion || {};
  // ChatCompletionResponse summary
  let completionChatMsg = '';
  let chatCompletion = completion?.chat_completion_response;

  if (!chatCompletion && Array.isArray(completion?.conversation) && completion.conversation.length > 0) {
    chatCompletion = { choices: completion.conversation.map(m => {
      return { message: m }
    }) }
  }

  if (
    chatCompletion &&
    Array.isArray(chatCompletion.choices) &&
    chatCompletion.choices.length > 0
  ) {
    const lastChoice = chatCompletion.choices[chatCompletion.choices.length - 1];
    // Prefer tool_call summary if present
    let toolCallSummary = '';
    const toolCalls = lastChoice?.message?.tool_calls;
    if (Array.isArray(toolCalls) && toolCalls.length > 0) {
      toolCallSummary = toolCalls.map(tc => {
        let args = '';
        // For OpenAI-style, arguments are in tc.function.arguments, function name in tc.function.name
        if (tc.function && tc.function.arguments) {
          try {
            args = typeof tc.function.arguments === 'string' ? tc.function.arguments : JSON.stringify(tc.function.arguments);
          } catch (e) {
            args = '[Unserializable arguments]';
          }
        }
        const toolName = tc.function?.name || tc.name || 'unknown';
        return `Tool call: ${toolName}(${args})`;
      }).join('\n');
    }
    completionChatMsg = lastChoice?.message?.content || '';
    // Attach toolCallSummary to completionChatMsg for rendering
    if (toolCallSummary) {
      completionChatMsg = { toolCallSummary, message: completionChatMsg };
    }
    // Else, it's just a string
  }
  
  // ActionResult summary
  let completionActionResult = '';
  if (completion?.action_result) {
    completionActionResult = `Action Result: ${String(completion.action_result).slice(0, 100)}`;
  }
  // AgentState summary
  let completionAgentState = '';
  if (completion?.agent_state) {
    completionAgentState = `Agent State: ${JSON.stringify(completion.agent_state)}`;
  }
  // Error summary
  let completionError = '';
  if (completion?.error) {
    completionError = `Error: ${completion.error}`;
  }

  let completionFilter = '';
  if (completion?.filter_result) {
    if (completion.filter_result?.has_triggers && !completion.filter_result?.triggered_by) {
      completionFilter = 'Failed to match any triggers';
    } else if (completion.filter_result?.triggered_by) {
      completionFilter = `Triggered by ${completion.filter_result.triggered_by}`;
    }

    if (completion?.filter_result?.failed_by)
      completionFilter = `${completionFilter ? completionFilter + ', ' : ''}Failed by ${completion.filter_result.failed_by}`;
  }

  // Only show if any summary is present
  if (!creationChatMsg && !creationFunctionDef && !creationFunctionParams &&
      !completionChatMsg && !completionActionResult && 
      !completionAgentState && !completionError && !completionFilter) {
    return null;
  }

  return (
    <div className="observable-summary">
      {/* CREATION */}
      {creationChatMsg && (
        <div className="summary-item creation-message" title={creationChatMsg}>
          <i className="fas fa-comment-dots"></i>
          <span>{creationChatMsg}</span>
        </div>
      )}
      {creationFunctionDef && (
        <div className="summary-item creation-function" title={creationFunctionDef}>
          <i className="fas fa-code"></i>
          <span>{creationFunctionDef}</span>
        </div>
      )}
      {creationFunctionParams && (
        <div className="summary-item creation-params" title={creationFunctionParams}>
          <i className="fas fa-sliders-h"></i>
          <span>{creationFunctionParams}</span>
        </div>
      )}
      
      {/* COMPLETION */}
      {/* COMPLETION: Tool call summary if present */}
      {completionChatMsg && typeof completionChatMsg === 'object' && completionChatMsg.toolCallSummary && (
        <div className="summary-item completion-tool-call" title={completionChatMsg.toolCallSummary}>
          <i className="fas fa-tools"></i>
          <span>{completionChatMsg.toolCallSummary}</span>
        </div>
      )}
      
      {/* COMPLETION: Message content if present */}
      {completionChatMsg && ((typeof completionChatMsg === 'object' && completionChatMsg.message) || typeof completionChatMsg === 'string') && (
        <div className="summary-item completion-message" title={typeof completionChatMsg === 'object' ? completionChatMsg.message : completionChatMsg}>
          <i className="fas fa-robot"></i>
          <span>{typeof completionChatMsg === 'object' ? completionChatMsg.message : completionChatMsg}</span>
        </div>
      )}
      
      {completionActionResult && (
        <div className="summary-item completion-action" title={completionActionResult}>
          <i className="fas fa-bolt"></i>
          <span>{completionActionResult}</span>
        </div>
      )}
      
      {completionAgentState && (
        <div className="summary-item completion-state" title={completionAgentState}>
          <i className="fas fa-brain"></i>
          <span>{completionAgentState}</span>
        </div>
      )}
      
      {completionError && (
        <div className="summary-item completion-error" title={completionError}>
          <i className="fas fa-exclamation-triangle"></i>
          <span>{completionError}</span>
        </div>
      )}
      
      {completionFilter && (
        <div className="summary-item completion-filter" title={completionFilter}>
          <i className="fas fa-shield-alt"></i>
          <span>{completionFilter}</span>
        </div>
      )}
    </div>
  );
}

function AgentStatus() {
  const [showStatus, setShowStatus] = useState(false);
  const { id } = useParams();
  const [statusData, setStatusData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [_eventSource, setEventSource] = useState(null);
  // Store all observables by id
  const [observableMap, setObservableMap] = useState({});
  const [observableTree, setObservableTree] = useState([]);
  const [expandedCards, setExpandedCards] = useState(new Map());

  const { agent } = useAgent(id);

  // Update document title
  useEffect(() => {
    if (agent?.name) {
      document.title = `Agent Status: ${agent.name} - LocalAGI`;
    }
    return () => {
      document.title = "LocalAGI";
    };
  }, [agent?.name]);

  // Fetch initial status data
  useEffect(() => {
    if (!id) {
      return;
    }
    const fetchStatusData = async () => {
      try {
        const response = await fetch(`/api/agent/${id}/status`);
        if (!response.ok) {
          throw new Error(`Server responded with status: ${response.status}`);
        }
        const data = await response.json();
        setStatusData(data);
      } catch (err) {
        console.error("Error fetching agent status:", err);
        setError(
          `Failed to load status for agent "${agent?.name || id}": ${err.message}`
        );
      } finally {
        setLoading(false);
      }
    };
    fetchStatusData();
  }, [id, agent?.name]);

  // Setup observables and SSE connection
  useEffect(() => {
    if (!id) return;

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
        const response = await fetch(`/api/agent/${id}/observables`);
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
    const sse = new EventSource(`/sse/${id}`);
    setEventSource(sse);

    sse.addEventListener('observable_update', (event) => {
      const data = JSON.parse(event.data);
      setObservableMap(prevMap => {
        const prev = prevMap[data.id] || {};
        const updated = {
          ...data,
          ...prev,
        };
        // Events can be received out of order
        if (data.creation)
          updated.creation = data.creation;
        if (data.completion)
          updated.completion = data.completion;
        if ((data.progress?.length ?? 0) > (prev.progress?.length ?? 0))
          updated.progress = data.progress;
        if (data.parent_id && !prevMap[data.parent_id])
          prevMap[data.parent_id] = {
            id: data.parent_id,
            name: "unknown",
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
  }, [id]);

  // Helper function to safely convert any value to a displayable string
  const formatValue = (value) => {
    if (value === null || value === undefined) {
      return "N/A";
    }

    if (typeof value === 'object') {
      try {
        return JSON.stringify(value, null, 2);
      } catch (err) {
        console.error("Error stringifying object:", err);
        return "[Complex Object]";
      }
    }

    return String(value);
  };

  if (loading) {
    return (
      <div className="dashboard-container">
        <div className="main-content-area">
          <div className="loading-container">
              <div className="spinner"></div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="dashboard-container">
        <div className="main-content-area">
          <div className="error-container">
            <h2>Error</h2>
            <p>{error}</p>
            <Link to="/agents" className="back-btn">
              <i className="fas fa-arrow-left"></i> Back to Agents
            </Link>
          </div>
        </div>
      </div>
    );
  }

  const hasStatusHistory = (Array.isArray(statusData?.History) && statusData.History.length > 0)

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Agent Status"
            description="See what the agent is doing and thinking"
            name={agent?.name || id}
          />
          
          <div className="header-right">
            <Link to={`/settings/${id}`} className="action-btn settings-btn">
              <i className="fas fa-cog"></i> Settings
            </Link>
            <Link to={`/talk/${id}/`} className="action-btn chat-btn">
              <i className="fas fa-comments"></i> Chat
            </Link>
          </div>
        </div>

        {error && (
          <div className="error-container">
            {error}
          </div>
        )}

        {statusData && (
          <>
            {/* Current Status Section */}
            <div className="section-box">
              <div 
                className={`section-header ${!hasStatusHistory ? 'no-history' : ''}`}
                onClick={() => hasStatusHistory && setShowStatus(prev => !prev)}
              >
                <h2>Current Status</h2>
                {hasStatusHistory && <i className={`fas fa-chevron-${showStatus ? 'up' : 'down'}`}></i>}
              </div>
             {
              hasStatusHistory 
              ? <p className="section-description">
                Summary of the agent's thoughts and actions
              </p>
              : <p className="section-description">
                 No status history available.
              </p>
             }
              
              {showStatus && (
                <div className="status-history">
                  {!hasStatusHistory && (
                    <div className="no-status-data">No status history available.</div>
                  )}
                  {Array.isArray(statusData?.History) && statusData.History.map((item, idx) => (
                    <div key={idx} className="status-item">
                      {/* Replace <br> tags with newlines, then render as pre-line */}
                      {typeof item === 'string'
                        ? item.replace(/<br\s*\/?>/gi, '\n')
                        : JSON.stringify(item)}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Observable Updates Section */}
            {observableTree.length > 0 && (
              <div className="section-box">
                <h2>Observable Updates</h2>
                <p className="section-description">
                  Drill down into what the agent is doing and thinking
                </p>
                
                <div className="observables-container">
                  {observableTree.map((container, idx) => (
                    <div key={container.id || idx} className="observable-card">
                      <div 
                        className="observable-header"
                        onClick={() => {
                          const newExpanded = !expandedCards.get(container.id);
                          setExpandedCards(new Map(expandedCards).set(container.id, newExpanded));
                        }}
                      >
                        <div className="observable-info">
                          <i className={`fas fa-${container.icon || 'robot'}`}></i>
                          <div className="observable-details">
                            <div className="observable-name">
                              <span className="stat-label">{container.name}</span>
                              <span className="observable-id">#{container.id}</span>
                            </div>
                            <ObservableSummary observable={container} />
                          </div>
                        </div>
                        
                        <div className="observable-actions">
                          <i className={`fas fa-chevron-${expandedCards.get(container.id) ? 'up' : 'down'}`}></i>
                          {!container.completion && (
                            <div className="spinner"></div>
                          )}
                        </div>
                      </div>
                      
                      {expandedCards.get(container.id) && (
                        <div className="observable-content">
                          {container.children && container.children.length > 0 && (
                            <div className="nested-observables">
                              <h4>Nested Observables</h4>
                              {container.children.map(child => {
                                const childKey = `child-${child.id}`;
                                const isExpanded = expandedCards.get(childKey);
                                return (
                                  <div key={`${container.id}-child-${child.id}`} className="nested-observable-card">
                                    <div 
                                      className="observable-header"
                                      onClick={() => {
                                        const newExpanded = !expandedCards.get(childKey);
                                        setExpandedCards(new Map(expandedCards).set(childKey, newExpanded));
                                      }}
                                    >
                                      <div className="observable-info">
                                        <i className={`fas fa-${child.icon || 'robot'}`}></i>
                                        <div className="observable-details">
                                          <div className="observable-name">
                                            <span className="stat-label">{child.name}</span>
                                            <span className="observable-id">#{child.id}</span>
                                          </div>
                                          <ObservableSummary observable={child} />
                                        </div>
                                      </div>
                                      
                                      <div className="observable-actions">
                                        <i className={`fas fa-chevron-${isExpanded ? 'up' : 'down'}`}></i>
                                        {!child.completion && (
                                          <div className="spinner"></div>
                                        )}
                                      </div>
                                    </div>
                                    
                                    {isExpanded && (
                                      <div className="observable-content">
                                        <CollapsibleRawSections container={child} />
                                      </div>
                                    )}
                                  </div>
                                );
                              })}
                            </div>
                          )}
                          <CollapsibleRawSections container={container} />
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default AgentStatus;
