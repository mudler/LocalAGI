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
    clearChat 
  } = useChat(name);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Show error toast if there's an error
  useEffect(() => {
    if (error) {
      showToast(error, 'error');
    }
  }, [error, showToast]);

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!message.trim()) return;
    
    const success = await sendMessage(message.trim());
    if (success) {
      setMessage('');
    }
  };

  return (
    <div className="chat-container">
      <header className="chat-header">
        <h1>Chat with {name}</h1>
        <div className="connection-status">
          <span className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}>
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
        <button 
          className="clear-chat-btn"
          onClick={clearChat}
          disabled={messages.length === 0}
        >
          Clear Chat
        </button>
      </header>

      <div className="messages-container">
        {messages.length === 0 ? (
          <div className="empty-chat">
            <p>No messages yet. Start a conversation with {name}!</p>
          </div>
        ) : (
          messages.map((msg) => (
            <div 
              key={msg.id} 
              className={`message ${msg.sender === 'user' ? 'user-message' : 'agent-message'}`}
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

      <form className="message-form" onSubmit={handleSubmit}>
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="Type your message..."
          disabled={sending || !isConnected}
          className="message-input"
        />
        <button 
          type="submit" 
          disabled={sending || !message.trim() || !isConnected}
          className="send-button"
        >
          {sending ? 'Sending...' : 'Send'}
        </button>
      </form>
    </div>
  );
}

export default Chat;
