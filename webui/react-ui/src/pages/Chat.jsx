import { useState, useRef, useEffect } from "react";
import { useParams, useOutletContext } from "react-router-dom";
import { useChat } from "../hooks/useChat";
import Header from "../components/Header";
import { agentApi } from "../utils/api";

function Chat() {
  const { name } = useParams();
  const { showToast } = useOutletContext();
  const [message, setMessage] = useState("");
  const [agentConfig, setAgentConfig] = useState(null);
  const [isOpenRouter, setIsOpenRouter] = useState(false);
  const messagesEndRef = useRef(null);

  // Fetch agent config on mount
  useEffect(() => {
    const fetchAgentConfig = async () => {
      try {
        const config = await agentApi.getAgentConfig(name);
        setAgentConfig(config);
        setIsOpenRouter(config.model.split("/")[0] === "openrouter");
      } catch (error) {
        console.error("Failed to load agent config", error);
        showToast && showToast(error?.message || String(error), "error");
        setAgentConfig(null);
      }
    };
    fetchAgentConfig();
  }, [name, showToast]);

  // Use our custom chat hook with model from agent config
  const {
    messages,
    sending,
    error,
    isConnected,
    sendMessage,
    clearChat,
    clearError,
  } = useChat(name, agentConfig?.model);

  console.log("Connected: ", isConnected);

  useEffect(() => {
    if (name) {
      document.title = `Chat with ${name} - LocalAGI`;
    }
    return () => {
      document.title = "LocalAGI";
    };
  }, [name]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    if (error) {
      showToast && showToast(error?.message || String(error), "error");
      clearError();
    }
  }, [error, showToast, clearError]);

  const handleSend = (e) => {
    e.preventDefault();
    if (message.trim() !== "") {
      sendMessage(message);
      setMessage("");
    }
  };

  if (!agentConfig) {
    return (
      <div className="dashboard-container">
        <div className="main-content-area">
          <p>Loading agent configuration...</p>
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
            name={name}
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
                <div
                  key={idx}
                  style={{
                    marginBottom: 12,
                    display: "flex",
                    flexDirection:
                      msg.sender === "user" ? "row-reverse" : "row",
                  }}
                >
                  <div
                    style={{
                      background: msg.sender === "user" ? "#e0e7ff" : "#f3f4f6",
                      color: "#222",
                      borderRadius: 18,
                      padding: "12px 18px",
                      maxWidth: "70%",
                      fontSize: "1rem",
                      boxShadow: "0 2px 6px rgba(0,0,0,0.04)",
                      alignSelf:
                        msg.sender === "user" ? "flex-end" : "flex-start",
                    }}
                  >
                    {msg.content}
                  </div>
                </div>
              ))
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
