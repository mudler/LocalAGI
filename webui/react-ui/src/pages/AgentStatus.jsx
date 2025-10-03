import { useState, useEffect } from 'react';
import CollapsibleRawSections from '../components/CollapsibleRawSections';

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

  // TODO:: Probably we have an array of two objects: [{type:"text", ...}, {type:"image_url", image_url: {url:"..."}}] 
  //        We could display the image and text along with multimedia icon
  if (typeof creationChatMsg === 'object') {
    console.log("Multimedia message?", creationChatMsg);
    creationChatMsg = 'Multimedia message';
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
    <div style={{ display: 'flex', flexDirection: 'column', gap: 2, margin: '2px 0 0 0' }}>
      {/* CREATION */}
      {creationChatMsg && (
        <div title={creationChatMsg} style={{ display: 'flex', alignItems: 'center', color: '#cfc', fontSize: 14 }}>
          <i className="fas fa-comment-dots" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{creationChatMsg}</span>
        </div>
      )}
      {creationFunctionDef && (
        <div title={creationFunctionDef} style={{ display: 'flex', alignItems: 'center', color: '#cfc', fontSize: 14 }}>
          <i className="fas fa-code" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{creationFunctionDef}</span>
        </div>
      )}
      {creationFunctionParams && (
        <div title={creationFunctionParams} style={{ display: 'flex', alignItems: 'center', color: '#fc9', fontSize: 14 }}>
          <i className="fas fa-sliders-h" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{creationFunctionParams}</span>
        </div>
      )}
      {/* COMPLETION */}
      {/* COMPLETION: Tool call summary if present */}
      {completionChatMsg && typeof completionChatMsg === 'object' && completionChatMsg.toolCallSummary && (
        <div
          title={completionChatMsg.toolCallSummary}
          style={{
            display: 'flex',
            alignItems: 'center',
            color: '#ffd966', // Distinct color for tool calls
            fontSize: 14,
            marginTop: 2,
            whiteSpace: 'pre-line',
            wordBreak: 'break-all',
          }}
        >
          <i className="fas fa-tools" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ whiteSpace: 'pre-line', display: 'block' }}>{completionChatMsg.toolCallSummary}</span>
        </div>
      )}
      {/* COMPLETION: Message content if present */}
      {completionChatMsg && ((typeof completionChatMsg === 'object' && completionChatMsg.message) || typeof completionChatMsg === 'string') && (
        <div
          title={typeof completionChatMsg === 'object' ? completionChatMsg.message : completionChatMsg}
          style={{
            display: 'flex',
            alignItems: 'center',
            color: '#8fc7ff',
            fontSize: 14,
            marginTop: 2,
          }}
        >
          <i className="fas fa-robot" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{typeof completionChatMsg === 'object' ? completionChatMsg.message : completionChatMsg}</span>
        </div>
      )}
      {completionActionResult && (
        <div title={completionActionResult} style={{ display: 'flex', alignItems: 'center', color: '#ffd700', fontSize: 14 }}>
          <i className="fas fa-bolt" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{completionActionResult}</span>
        </div>
      )}
      {completionAgentState && (
        <div title={completionAgentState} style={{ display: 'flex', alignItems: 'center', color: '#ffb8b8', fontSize: 14 }}>
          <i className="fas fa-brain" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{completionAgentState}</span>
        </div>
      )}
      {completionError && (
        <div title={completionError} style={{ display: 'flex', alignItems: 'center', color: '#f66', fontSize: 14 }}>
          <i className="fas fa-exclamation-triangle" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{completionError}</span>
        </div>
      )}
      {completionFilter && (
        <div title={completionFilter} style={{ display: 'flex', alignItems: 'center', color: '#ffd7', fontSize: 14 }}>
          <i className="fas fa-shield-alt" style={{ marginRight: 6, flex: '0 0 auto' }}></i>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block' }}>{completionFilter}</span>
        </div>
      )}
    </div>
  );
}

import { useParams, Link } from 'react-router-dom';
import hljs from 'highlight.js/lib/core';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/monokai.css';

hljs.registerLanguage('json', json);

function AgentStatus() {
  const [showStatus, setShowStatus] = useState(false);
  const { name } = useParams();
  const [statusData, setStatusData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [_eventSource, setEventSource] = useState(null);
  // Store all observables by id
  const [observableMap, setObservableMap] = useState({});
  const [observableTree, setObservableTree] = useState([]);
  const [expandedCards, setExpandedCards] = useState(new Map());
  const [clearLoading, setClearLoading] = useState(false);

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
  }, [name]);

  const handleClearObservables = async () => {
    if (clearLoading) return;
    setClearLoading(true);
    try {
      const resp = await fetch(`/api/agent/${name}/observables`, { method: 'DELETE' });
      if (!resp.ok) {
        console.error('Failed to clear observables, status:', resp.status);
      } else {
        // Clear local state immediately
        setObservableMap({});
        setObservableTree([]);
        setExpandedCards(new Map());
      }
    } catch (e) {
      console.error('Error clearing observables:', e);
    } finally {
      setClearLoading(false);
    }
  };

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
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h2 style={{ margin: 0 }}>Observable Updates</h2>
                <button
                  className="action-btn delete-btn"
                  onClick={handleClearObservables}
                  disabled={clearLoading}
                  title="Clear observable history"
                >
                  {clearLoading ? 'Clearing…' : 'Clear history'}
                </button>
              </div>
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
                        <div style={{ display: 'flex', gap: '10px', alignItems: 'center', maxWidth: '90%' }}>
                          <i className={`fas fa-${container.icon || 'robot'}`} style={{ verticalAlign: '-0.125em' }}></i>
                          <span style={{ width: '100%' }}>
                            <div style={{ display: 'flex', flexDirection: 'column', flex: 1 }}>
                              <span>
                                <span className='stat-label'>{container.name}</span>#<span className='stat-label'>{container.id}</span>
                              </span>
                              <ObservableSummary observable={container} />
                            </div>
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
                                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'hand', maxWidth: '100%' }}
                                    onClick={() => {
                                      const newExpanded = !expandedCards.get(childKey);
                                      setExpandedCards(new Map(expandedCards).set(childKey, newExpanded));
                                    }}
                                  >
                                    <div style={{ display: 'flex', maxWidth: '90%', gap: '10px', alignItems: 'center' }}>
                                      <i className={`fas fa-${child.icon || 'robot'}`} style={{ verticalAlign: '-0.125em' }}></i>
                                      <span style={{ width: '100%' }}>
                                        <div style={{ display: 'flex', flexDirection: 'column', flex: 1 }}>
                                          <span>
                                            <span className='stat-label'>{child.name}</span>#<span className='stat-label'>{child.id}</span>
                                          </span>
                                          <ObservableSummary observable={child} />
                                        </div>
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
                                    <CollapsibleRawSections container={child} />
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        )}
                        <CollapsibleRawSections container={container} />
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
