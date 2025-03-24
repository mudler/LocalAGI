import { useState, useCallback, useEffect } from 'react';
import { chatApi } from '../utils/api';
import { useSSE } from './useSSE';

/**
 * Custom hook for chat functionality
 * @param {string} agentName - Name of the agent to chat with
 * @returns {Object} - Chat state and functions
 */
export function useChat(agentName) {
  const [messages, setMessages] = useState([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState(null);
  
  // Use SSE hook to receive real-time messages
  const { data: sseData, isConnected } = useSSE(agentName);
  
  // Process SSE data into messages
  useEffect(() => {
    if (sseData && sseData.length > 0) {
      // Process the latest SSE data
      const latestData = sseData[sseData.length - 1];
      
      if (latestData.type === 'message') {
        setMessages(prev => [...prev, {
          id: Date.now().toString(),
          sender: 'agent',
          content: latestData.content,
          timestamp: new Date().toISOString(),
        }]);
      }
    }
  }, [sseData]);

  // Send a message to the agent
  const sendMessage = useCallback(async (content) => {
    if (!agentName || !content) return;
    
    setSending(true);
    setError(null);
    
    // Add user message to the list
    const userMessage = {
      id: Date.now().toString(),
      sender: 'user',
      content,
      timestamp: new Date().toISOString(),
    };
    
    setMessages(prev => [...prev, userMessage]);
    
    try {
      await chatApi.sendMessage(agentName, content);
      // The agent's response will come through SSE
      return true;
    } catch (err) {
      setError(err.message || 'Failed to send message');
      console.error('Error sending message:', err);
      return false;
    } finally {
      setSending(false);
    }
  }, [agentName]);

  // Clear chat history
  const clearChat = useCallback(() => {
    setMessages([]);
  }, []);

  return {
    messages,
    sending,
    error,
    isConnected,
    sendMessage,
    clearChat,
  };
}
