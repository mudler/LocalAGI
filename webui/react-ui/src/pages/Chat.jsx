import { useState, useRef, useEffect } from 'react';
import { useParams, useOutletContext } from 'react-router-dom';
import { useChat } from '../hooks/useChat';

function Chat() {
  const { name } = useParams();
  const { showToast } = useOutletContext();
  const [message, setMessage] = useState('');
  const messagesEndRef = useRef(null);
  
  // Use our custom chat hook
  const { 
    messages, 
    sending, 
    error, 
    isConnected, 
    sendMessage, 
    clearChat,
    clearError
  } = useChat(name);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

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
