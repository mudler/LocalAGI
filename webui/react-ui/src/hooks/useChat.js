import { useState, useCallback, useEffect, useRef } from 'react';
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
  const [streamReasoning, setStreamReasoning] = useState('');
  const [streamContent, setStreamContent] = useState('');
  const [streamToolCalls, setStreamToolCalls] = useState([]);
  const processedMessageIds = useRef(new Set());
  const processedStreamIds = useRef(new Set());
  const localMessageContents = useRef(new Set()); // Track locally added message contents

  // Use SSE hook to receive real-time messages
  const { messages: sseMessages, statusUpdates, errorMessages, streamEvents, isConnected } = useSSE(agentName);
  
  // Process SSE messages into chat messages
  useEffect(() => {
    if (!sseMessages || sseMessages.length === 0) return;
    
    // Process the latest SSE message
    const latestMessage = sseMessages[sseMessages.length - 1];
    
    // Skip if we've already processed this message
    if (processedMessageIds.current.has(latestMessage.id)) {
      return;
    }
    
    // Handle JSON messages
    if (latestMessage.type === 'json_message') {
      try {
        // The message should already be a parsed JSON object
        const messageData = latestMessage.content;
        
        // Skip if we've already processed this message ID
        if (processedMessageIds.current.has(messageData.id)) {
          return;
        }
        
        // Add to processed set to avoid duplicates
        processedMessageIds.current.add(messageData.id);
        
        // Skip user messages that we've already added locally
        if (messageData.sender === 'user' && localMessageContents.current.has(messageData.content)) {
          return;
        }
        
        // Add the message to our state
        setMessages(prev => [...prev, {
          id: messageData.id,
          sender: messageData.sender,
          content: messageData.content,
          timestamp: messageData.timestamp,
        }]);
      } catch (err) {
        console.error('Error processing JSON message:', err);
      }
    }
  }, [sseMessages]);
  
  // Process status updates
  useEffect(() => {
    if (!statusUpdates || statusUpdates.length === 0) return;
    
    const latestStatus = statusUpdates[statusUpdates.length - 1];
    
    // Handle status updates
    if (latestStatus.type === 'status') {
      try {
        // The status should be a parsed JSON object
        const statusData = latestStatus.content;
        
        if (statusData.status === 'processing') {
          setSending(true);
          setStreamReasoning('');
          setStreamContent('');
          setStreamToolCalls([]);
        } else if (statusData.status === 'completed') {
          setSending(false);
          setStreamReasoning('');
          setStreamContent('');
          setStreamToolCalls([]);
        }
      } catch (err) {
        console.error('Error processing status update:', err);
      }
    }
  }, [statusUpdates]);
  
  // Process error messages
  useEffect(() => {
    if (!errorMessages || errorMessages.length === 0) return;
    
    const latestError = errorMessages[errorMessages.length - 1];
    
    try {
      // The error should be a parsed JSON object
      const errorData = latestError.content;
      
      if (errorData.error) {
        setError(errorData.error);
      }
    } catch (err) {
      console.error('Error processing error message:', err);
    }
  }, [errorMessages]);

  // Process stream events (reasoning, content, tool_call, done)
  useEffect(() => {
    if (!streamEvents || streamEvents.length === 0) return;

    const latestEvent = streamEvents[streamEvents.length - 1];
    if (processedStreamIds.current.has(latestEvent.id)) return;
    processedStreamIds.current.add(latestEvent.id);

    const data = latestEvent.content;
    if (data.type === 'reasoning') {
      setStreamReasoning(prev => prev + (data.content || ''));
    } else if (data.type === 'content') {
      setStreamContent(prev => prev + (data.content || ''));
    } else if (data.type === 'tool_call') {
      const name = data.tool_name || '';
      const args = data.tool_args || '';
      if (name) {
        // Reset reasoning and content when a new tool call starts —
        // each iteration gets its own thinking block
        setStreamReasoning('');
        setStreamContent('');
      }
      setStreamToolCalls(prev => {
        if (name) {
          return [...prev, { name, args }];
        }
        if (prev.length === 0) return prev;
        const updated = [...prev];
        updated[updated.length - 1] = { ...updated[updated.length - 1], args: updated[updated.length - 1].args + args };
        return updated;
      });
    } else if (data.type === 'done') {
      // Stream complete — content finalized by json_message event
    }
  }, [streamEvents]);

  // Send a message to the agent
  const sendMessage = useCallback(async (content) => {
    if (!agentName || !content) return false;
    
    setSending(true);
    setError(null);
    
    try {
      // Add user message to the local state immediately for better UX
      const messageId = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      
      const userMessage = {
        id: messageId,
        sender: 'user',
        content,
        timestamp: new Date().toISOString(),
      };
      
      setMessages(prev => [...prev, userMessage]);
      
      // Track this message content to avoid duplication from SSE
      localMessageContents.current.add(content);
      
      // Use the JSON-based API endpoint
      await chatApi.sendMessage(agentName, content);
      return true;
    } catch (err) {
      setError(err.message || 'Failed to send message');
      console.error('Error sending message:', err);
      setSending(false);
      return false;
    } finally {
      // Ensure sending state is reset after a timeout in case SSE doesn't update
      setTimeout(() => {
        if (sending) {
          setSending(false);
        }
      }, 5000); // 5 second timeout
    }
  }, [agentName, sending]);

  // Clear chat history
  const clearChat = useCallback(() => {
    setMessages([]);
    processedMessageIds.current.clear();
    localMessageContents.current.clear(); // Clear tracked local messages
  }, []);

  // Clear error state
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    messages,
    sending,
    error,
    isConnected,
    streamReasoning,
    streamContent,
    streamToolCalls,
    sendMessage,
    clearChat,
    clearError,
  };
}