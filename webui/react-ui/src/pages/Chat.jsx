import { useState, useRef, useEffect } from "react";
import { useParams, useOutletContext } from "react-router-dom";
import { useChat } from "../hooks/useChat";
import Header from "../components/Header";
import { agentApi } from "../utils/api";
import TypingIndicator from "../components/TypingIndicator";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

function Chat() {
  const { id } = useParams();
  const { showToast } = useOutletContext();
  const [message, setMessage] = useState("");
  const [agentConfig, setAgentConfig] = useState(null);
  const [isOpenRouter, setIsOpenRouter] = useState(false);
  const messagesEndRef = useRef(null);
  
  // Observable status tracking
  const [currentStatus, setCurrentStatus] = useState(null);
  const [eventSource, setEventSource] = useState(null);

  // Helper function to map observable data to user-friendly status messages
  const getStatusMessage = (observable) => {
    if (!observable) return null;
    
    // Check for errors first
    if (observable.completion?.error) {
      return 'Error while processing. Please try again.';
    }
    
    const name = observable.name?.toLowerCase() || '';
    
    // Map different observable types to user-friendly messages
    switch (name) {
      case 'job':
        return 'Thinking';
      case 'decision':
        // Check for tool calls in completion to provide more specific status
        const completion = observable.completion;
        if (completion?.chat_completion_response?.choices?.[0]?.message?.tool_calls) {
          const toolCalls = completion.chat_completion_response.choices[0].message.tool_calls;
          if (Array.isArray(toolCalls) && toolCalls.length > 0) {
            let toolName = toolCalls[0].function?.name || toolCalls[0].name || '';
            
            // Try to extract actual tool from arguments if function name is generic (like "pick_tool")
            if (toolName === 'pick_tool' || toolName === 'call_tool') {
              try {
                const args = JSON.parse(toolCalls[0].function?.arguments || '{}');
                if (args.tool) {
                  toolName = args.tool;
                }
              } catch (e) {
                // If parsing fails, keep the original toolName
                console.log('Failed to parse tool arguments:', e);
              }
            }
            
            if (toolName.toLowerCase().includes('reasoning') || toolName.toLowerCase().includes('reason')) {
              return 'Reasoning';
            }
            // if (toolName.toLowerCase().includes('search')) {
            //   return 'Searching the web';
            // }
            // if (toolName.toLowerCase().includes('browse')) {
            //   return 'Browsing the web';
            // }
            // if (toolName.toLowerCase().includes('github')) {
            //   return 'Checking GitHub';
            // }
            // if (toolName.toLowerCase().includes('email') || toolName.toLowerCase().includes('mail')) {
            //   return 'Composing email';
            // }
            // if (toolName.toLowerCase().includes('shell') || toolName.toLowerCase().includes('command')) {
            //   return 'Running command';
            // }
            // if (toolName.toLowerCase().includes('write') || toolName.toLowerCase().includes('create')) {
            //   return 'Writing content';
            // }
            // if (toolName.toLowerCase().includes('read') || toolName.toLowerCase().includes('get')) {
            //   return 'Reading content';
            // }
            if (toolName.toLowerCase().includes('reply')) {
              return 'Preparing response';
            }
            // Generic tool call message
            return null;
          }
        }
        return 'Thinking';
      case 'action':
        // Try to get more specific message based on action type
        const actionName = observable.creation?.function_definition?.name;
        if (actionName) {
          if (actionName.includes('search')) return 'Searching the web';
          if (actionName.includes('browse')) return 'Browsing the web page';
          if (actionName.includes('github')) return 'Checking GitHub';
          if (actionName.includes('email') || actionName.includes('mail')) return 'Sending email';
          if (actionName.includes('shell')) return 'Running command';
          return `Running ${actionName}`;
        }
        return 'Taking action';
      case 'filter':
        return 'Checking permissions';
      default:
        return 'Processing';
    }
  };

  // Helper function to check if an observable has an error
  const hasError = (observable) => {
    return observable?.completion?.error;
  };


  // Fetch agent config on mount
  useEffect(() => {
    const fetchAgentConfig = async () => {
      try {
        const config = await agentApi.getAgentConfig(id);
        setAgentConfig(config);
        setIsOpenRouter(config.model.split("/")[0] === "openrouter");
      } catch (error) {
        console.error("Failed to load agent config", error);
        showToast && showToast(error?.message || String(error), "error");
        setAgentConfig(null);
      }
    };
    fetchAgentConfig();
  }, []);

  // Setup SSE connection for observable updates
  useEffect(() => {
    if (!id) return;

    const sse = new EventSource(`/sse/${id}`);
    setEventSource(sse);

    sse.addEventListener('observable_update', (event) => {
      const data = JSON.parse(event.data);
      
      if (data.completion) {
        // Observable is completed
        if (data.completion.error) {
          setCurrentStatus({ message: 'Error while processing. Please try again.', isError: true });
        } else {
          if(data.name.toLowerCase() === 'decision') {
            const statusMessage = getStatusMessage(data);
            if (statusMessage) {
              setCurrentStatus({ message: statusMessage, isError: false });
            }
          }
        }
      } else {
        // Observable is active, show its status
        const statusMessage = getStatusMessage(data);
        if (statusMessage) {
          setCurrentStatus({ message: statusMessage, isError: false });
        }
      }
    });

    sse.onerror = (err) => {
      console.error('SSE connection error:', err);
    };

    return () => {
      if (sse) {
        sse.close();
      }
    };
  }, [id]);

  // Use our custom chat hook with model from agent config
  const {
    messages,
    sending,
    error,
    isConnected,
    sendMessage,
    clearChat,
    clearError,
  } = useChat(id, agentConfig?.model);



  // Clear status when we receive a new assistant message
  useEffect(() => {
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1];
      if (lastMessage.sender === 'agent' && !lastMessage.loading) {
        setCurrentStatus(null);
      }
    }
  }, [messages]);

  useEffect(() => {
    if (agentConfig) {
      document.title = `Chat with ${agentConfig.name} - LocalAGI`;
    }
    return () => {
      document.title = "LocalAGI";
    };
  }, [agentConfig]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, currentStatus]);

  // useEffect(() => {
  //   if (error) {
  //     showToast && showToast(error?.message || String(error), "error");
  //     clearError();
  //   }
  // }, [error, showToast, clearError]);

  const handleSend = (e) => {
    e.preventDefault();
    if (message.trim() !== "") {
      sendMessage(message);
      setMessage("");
      // Clear any existing status when sending a new message
      setCurrentStatus(null);
    }
  };

  if (!agentConfig) {
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

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Chat with"
            description="Send messages and interact with your agent in real time."
            name={agentConfig.name}
          />
          {/* No right content for chat header */}
        </div>

        {/* Chat Window */}
        <div
          className="section-box chat-section-box"
          style={{
            width: "100%",
            height: "calc(100vh - 300px)",
            display: "flex",
            flexDirection: "column",
            margin: 0,
            maxWidth: "none",
          }}
        >
          <div
            style={{
              flex: 1,
              overflowY: "auto",
            }}
          >
            {messages.length === 0 ? (
              <div
                style={{
                  color: "var(--text-light)",
                  textAlign: "center",
                  marginTop: 48,
                }}
              >
                No messages yet. Say hello!
              </div>
            ) : (
              messages.map((msg, idx) => (
                msg.loading ? (
                  null
                ) : (
                  <div
                    key={idx}
                    style={{
                      marginBottom: 12,
                      display: "flex",
                      flexDirection:
                        msg.sender === "user" ? "row-reverse" : "row",
                    }}
                  >
                    {
                      msg.sender === "user" ? (
                        <div
                          style={{
                            background: "#e0e7ff",
                            color: "#222",
                            borderRadius: 18,
                            padding: "12px 18px",
                            maxWidth: "70%",
                            fontSize: "1rem",
                            boxShadow: "0 2px 6px rgba(0,0,0,0.04)",
                            alignSelf: "flex-end",
                          }}
                        >
                          <div className="markdown-content">
                            <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                          </div>
                        </div>
                      ) : (
                        // Check if this is an error message
                        msg.type === "error" ? (
                          <div
                            style={{
                              color: "#991b1b",
                              padding: "12px 0",
                              maxWidth: "70%",
                              fontSize: "1rem",
                              alignSelf: "flex-start",
                              display: "flex",
                              alignItems: "center",
                              gap: "8px",
                            }}
                          >
                            <span style={{ fontSize: "16px", fontWeight: 400 }}>
                              <div className="markdown-content">
                                <ReactMarkdown remarkPlugins={[remarkGfm]}>Error while processing. Please try again.</ReactMarkdown>
                              </div>
                            </span>
                          </div>
                        ) : (
                          <div
                            style={{
                              background: "transparent",
                              color: "#222",
                              padding: "12px 0",
                              maxWidth: "70%",
                              fontSize: "1rem",
                              alignSelf: "flex-start",
                              position: "relative",
                            }}
                          >
                            <div className="markdown-content">
                              <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                            </div>
                          </div>
                        )
                      )
                    }
                  </div>
                )
              ))
            )}
            
            {/* Show current status as a temporary message */}
            {currentStatus && (
              <div
                style={{
                  marginBottom: 12,
                  display: "flex",
                  flexDirection: "row",
                }}
              >
                <div
                  style={{
                    color: currentStatus.isError ? "#991b1b" : "#6b7280",
                    padding: "12px 0",
                    maxWidth: "70%",
                    fontSize: "1rem",
                    alignSelf: "flex-start",
                    display: "flex",
                    alignItems: "center",
                    gap: "8px",
                  }}
                >
                  {currentStatus.isError ? (
                    null
                  ) : (
                    <div
                      style={{
                        width: "16px",
                        height: "16px",
                        border: "2px solid #e5e7eb",
                        borderTop: "2px solid #6b7280",
                        borderRadius: "50%",
                        animation: "spin 1s linear infinite",
                        flexShrink: 0,
                      }}
                    />
                  )}
                  <span style={{ fontSize: "16px", fontWeight: currentStatus.isError ? 400 : 500 }}>{currentStatus.message}</span>
                </div>
              </div>
            )}
            
            <div ref={messagesEndRef} />
          </div>

          {/* Chat Input */}
          <form
            onSubmit={handleSend}
            style={{ display: "flex", gap: 12, alignItems: "center" }}
            autoComplete="off"
          >
            <input
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={
                isOpenRouter || isConnected
                  ? "Type your message..."
                  : "Connecting..."
              }
              disabled={sending || (!isOpenRouter && !isConnected)}
              style={{
                flex: 1,
                padding: "12px 16px",
                border: "1px solid #e5e7eb",
                borderRadius: 8,
                fontSize: "1rem",
                background:
                  sending || (!isOpenRouter && !isConnected)
                    ? "#f3f4f6"
                    : "#fff",
                color: "#222",
                outline: "none",
                transition: "border-color 0.15s",
              }}
            />
            <button
              type="submit"
              className="action-btn"
              style={{ minWidth: 120 }}
              disabled={
                sending ||
                (!isOpenRouter && !isConnected) ||
                message.trim() === ""
              }
            >
              <i className="fas fa-paper-plane"></i> Send
            </button>
            <button
              type="button"
              className="action-btn"
              style={{ background: "#f6f8fa", color: "#222", minWidth: 120 }}
              onClick={clearChat}
              disabled={sending || messages.length === 0}
            >
              <i className="fas fa-trash"></i> Clear Chat
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default Chat;
