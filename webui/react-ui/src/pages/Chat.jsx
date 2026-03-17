import { useState, useRef, useEffect } from 'react';
import { useParams, useOutletContext, useNavigate } from 'react-router-dom';
import { useChat } from '../hooks/useChat';

function Chat() {
  const { name } = useParams();
  const { showToast } = useOutletContext();
  const navigate = useNavigate();
  const [message, setMessage] = useState('');
  const messagesEndRef = useRef(null);
  
  // Use our custom chat hook
  const {
    messages,
    sending,
    error,
    isConnected,
    streamReasoning,
    streamContent,
    streamToolCalls,
    sendMessage,
    clearChat,
    clearError
  } = useChat(name);

  // Update document title
  useEffect(() => {
    if (name) {
      document.title = `Chat with ${name} - LocalAGI`;
    }
    return () => {
      document.title = 'LocalAGI'; // Reset title when component unmounts
    };
  }, [name]);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamContent, streamReasoning, streamToolCalls]);

  // Show error toast if there's an error
  useEffect(() => {
    if (error) {
      showToast(error, 'error');
      clearError();
    }
  }, [error, showToast, clearError]);

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!message.trim()) return;
    
    const success = await sendMessage(message.trim());
    if (success) {
      setMessage('');
    }
  };

  // Handle pressing Enter to send (Shift+Enter for new line)
  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <div className="agents-container">
      <header className="page-header">
        <h1>Chat with {name}</h1>
        <div className="connection-status" style={{ display: 'flex', alignItems: 'center' }}>
          <span 
            className={isConnected ? 'active' : 'inactive'} 
            style={{ 
              position: 'static', 
              display: 'inline-block',
              padding: '5px 12px',
              borderRadius: '20px',
              fontSize: '0.8rem',
              fontWeight: '500',
              textTransform: 'uppercase',
              letterSpacing: '1px',
              boxShadow: '0 0 10px rgba(0, 0, 0, 0.2)',
              marginLeft: '10px'
            }}
          >
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
        <button 
          className="action-btn delete-btn"
          onClick={clearChat}
          disabled={messages.length === 0}
        >
          <i className="fas fa-trash-alt"></i> Clear Chat
        </button>
      </header>

      <div className="chat-container">
        <div className="chat-messages">
          {messages.length === 0 ? (
            <div className="no-agents">
              <h2>No messages yet</h2>
              <p>Start a conversation with {name}!</p>
            </div>
          ) : (
            messages.map((msg) => (
              <div 
                key={msg.id} 
                className={`message ${msg.sender === 'user' ? 'message-user' : 'message-agent'}`}
              >
                <div className="message-content">
                  {msg.content}
                </div>
                <div className="message-timestamp">
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </div>
              </div>
            ))
          )}
          {sending && (streamReasoning || streamContent || streamToolCalls.length > 0) && (
            <div className="message message-agent">
              <div className="message-content">
                {streamReasoning && (
                  <details open={!streamContent && streamToolCalls.length === 0} style={{ marginBottom: (streamContent || streamToolCalls.length > 0) ? '0.5rem' : 0 }}>
                    <summary style={{ cursor: 'pointer', fontStyle: 'italic', opacity: 0.7 }}>
                      {streamContent || streamToolCalls.length > 0 ? 'Thinking' : 'Thinking...'}
                    </summary>
                    <div
                      ref={(el) => { if (el) el.scrollTop = el.scrollHeight; }}
                      style={{ whiteSpace: 'pre-wrap', opacity: 0.6, fontSize: '0.9em', marginTop: '0.25rem', maxHeight: '300px', overflowY: 'auto' }}
                    >
                      {streamReasoning}
                    </div>
                  </details>
                )}
                {streamToolCalls.length > 0 ? (
                  <div style={{ marginTop: '0.25rem' }}>
                    {streamToolCalls.map((tc, idx) => (
                      <div key={idx} style={{ fontSize: '0.85em', opacity: 0.7, padding: '2px 0' }}>
                        <i className="fas fa-wrench" style={{ marginRight: '6px' }} />
                        <strong>{tc.name}</strong>
                        {tc.args && <span style={{ opacity: 0.5, marginLeft: '4px', fontSize: '0.9em' }}>{tc.args}</span>}
                        <span style={{ opacity: 0.5, marginLeft: '4px' }}>calling...</span>
                      </div>
                    ))}
                  </div>
                ) : streamContent ? (
                  <div style={{ whiteSpace: 'pre-wrap' }}>{streamContent}</div>
                ) : null}
              </div>
            </div>
          )}
          {sending && !streamReasoning && !streamContent && streamToolCalls.length === 0 && (
            <div className="message message-agent">
              <div className="message-content" style={{ fontStyle: 'italic', opacity: 0.5 }}>
                <i className="fas fa-spinner fa-spin" style={{ marginRight: '6px' }} /> Working...
              </div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        <div className="chat-input">
          <form className="message-form" onSubmit={handleSubmit} style={{ display: 'flex', gap: '1rem', width: '100%' }}>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type your message... (Press Enter to send, Shift+Enter for new line)"
              disabled={sending || !isConnected}
              className="form-control"
              rows={5}
              style={{ flex: 1, resize: 'vertical', minHeight: '38px', maxHeight: '150px' }}
            />
            <button 
              type="submit" 
              disabled={sending || !message.trim() || !isConnected}
              className="action-btn chat-btn"
              style={{ alignSelf: 'flex-end' }}
            >
              <i className={`fas ${sending ? 'fa-spinner fa-spin' : 'fa-paper-plane'}`}></i> {sending ? 'Sending...' : 'Send'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default Chat;
