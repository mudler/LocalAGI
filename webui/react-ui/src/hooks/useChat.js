import { useState, useCallback, useEffect, useRef } from "react";
import { chatApi } from "../utils/api";
import { useSSE } from "./useSSE";

/**
 * Custom hook for chat functionality
 * @param {string} agentId - Id of the agent to chat with
 * @param {Object} model - Model object (should include id)
 * @returns {Object} - Chat state and functions
 */
export function useChat(agentId, model) {
  const [messages, setMessages] = useState([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState(null);
  const processedMessageIds = useRef(new Set());
  const localMessageContents = useRef(new Set()); // Track locally added message contents
  const eventSourceRef = useRef(null);

  // Fetch initial chat history on mount or when agentId changes
  useEffect(() => {
    if (!agentId) return;
    const fetchHistory = async () => {
      try {
        const result = await chatApi.getChatHistory(agentId);
        if (result?.messages?.length) {
          const formatted = result.messages.map((msg, index) => ({
            id: msg.id || `${index}-${msg.sender}`, // fallback id
            sender: msg.sender,
            content: msg.content,
            type: msg.type,
            timestamp: msg.timestamp || new Date().toISOString(),
          }));
          setMessages(formatted);
          formatted.forEach((msg) => {
            processedMessageIds.current.add(msg.id);
            if (msg.sender === "user") {
              localMessageContents.current.add(msg.content);
            }
          });
        }
      } catch (err) {
        console.error("Failed to fetch chat history:", err);
      }
    };
    fetchHistory();
  }, [agentId]);

  const {
    messages: sseMessages,
    statusUpdates,
    errorMessages,
    isConnected,
  } = useSSE(model && typeof model === "string" ? agentId : null);

  // Set up additional SSE connection for streaming messages
  useEffect(() => {
    if (!agentId) return;

    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const sseUrl = `/sse/${agentId}`;
    const eventSource = new EventSource(sseUrl);
    eventSourceRef.current = eventSource;

    // Handle streaming message chunks
    eventSource.addEventListener("json_message_chunk", (event) => {
      const data = JSON.parse(event.data);
      
      setMessages((prevMessages) => {
        const existingIndex = prevMessages.findIndex(msg => msg.id === data.id);
        
        if (existingIndex >= 0) {
          // Update existing streaming message
          const updated = [...prevMessages];
          updated[existingIndex] = {
            ...updated[existingIndex],
            content: data.content, // Use accumulated content
            streaming: true,
            loading: false,
          };
          return updated;
        } else {
          // Create new streaming message (remove any loading message first)
          const withoutLoading = prevMessages.filter(
            (msg) => !(msg.sender === "assistant" && msg.loading)
          );
          
          return [...withoutLoading, {
            id: data.id,
            sender: data.sender,
            content: data.content,
            timestamp: data.createdAt,
            loading: false,
            streaming: true,
          }];
        }
      });
    });

    // Handle final message completion
    eventSource.addEventListener("json_message", (event) => {
      const data = JSON.parse(event.data);
      // Only handle final messages (with final: true flag)
      if (data.final) {
        setMessages((prevMessages) => {
          const existingIndex = prevMessages.findIndex(msg => msg.id === data.id);
          
          if (existingIndex >= 0) {
            // Update existing message
            return prevMessages.map((msg) =>
              msg.id === data.id
                ? { ...msg, content: data.content, loading: false, streaming: false }
                : msg
            );
          } else {
            // Add new message if no match found
            return [...prevMessages, {
              id: data.id,
              sender: data.sender,
              content: data.content,
              timestamp: data.createdAt || data.timestamp || new Date().toISOString(),
              loading: false,
              streaming: false,
            }];
          }
        });
        
        // Mark as processed to avoid duplicate from regular SSE
        processedMessageIds.current.add(data.id);
      }
    });

    eventSource.onerror = (err) => {
      console.error("Streaming SSE connection error:", err);
    };

    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [agentId]);

  // Process SSE messages (keep existing functionality)
  useEffect(() => {
    if (!sseMessages || sseMessages.length === 0) return;
    const latestMessage = sseMessages[sseMessages.length - 1];
    if (processedMessageIds.current.has(latestMessage.id)) return;
    if (latestMessage.type === "json_message") {
      try {
        const messageData = latestMessage.content;
        if (processedMessageIds.current.has(messageData.id)) return;
        processedMessageIds.current.add(messageData.id);
        if (
          messageData.sender === "user" &&
          localMessageContents.current.has(messageData.content)
        )
          return;
        setMessages((prev) => {
          // Remove the latest loading message if present
          const withoutLoading = prev.filter(
            (msg) => !(msg.sender === "assistant" && msg.loading)
          );
          return [
            ...withoutLoading,
            {
              id: messageData.id,
              sender: messageData.sender,
              content: messageData.content,
              timestamp: messageData.timestamp,
            },
          ];
        });
      } catch (err) {
        console.error("Error processing JSON message:", err);
      }
    }
  }, [sseMessages]);

  // Process status updates
  useEffect(() => {
    if (!statusUpdates || statusUpdates.length === 0) return;
    const latestStatus = statusUpdates[statusUpdates.length - 1];
    if (latestStatus.type === "status") {
      try {
        const statusData = latestStatus.content;
        if (statusData.status === "processing") {
          setSending(true);
        } else if (statusData.status === "completed") {
          setSending(false);
        }
      } catch (err) {
        console.error("Error processing status update:", err);
      }
    }
  }, [statusUpdates]);

  // Process errors
  useEffect(() => {
    if (!errorMessages || errorMessages.length === 0) return;
    const latestError = errorMessages[errorMessages.length - 1];
    try {
      const errorData = latestError.content;
      if (errorData.error) {
        setError(errorData.error);
      }
    } catch (err) {
      console.error("Error processing error message:", err);
    }
  }, [errorMessages]);

  const sendMessage = useCallback(
    async (content) => {
      if (!model || !content) return false;
      setSending(true);
      setError(null);

      const messageId = `${Date.now()}-${Math.random()
        .toString(36)
        .substr(2, 9)}`;
      const userMessage = {
        id: messageId,
        sender: "user",
        content,
        timestamp: new Date().toISOString(),
      };

      const loadingMessageId = `${messageId}-loading`;
      const loadingMessage = {
        id: loadingMessageId,
        sender: "assistant",
        content: "",
        loading: true,
        timestamp: new Date().toISOString(),
      };

      setMessages((prev) => [...prev, userMessage, loadingMessage]);
      localMessageContents.current.add(content);

      try {
        // For local model (response comes from SSE), just wait
        await chatApi.sendMessage(agentId, content);
        // SSE will handle replacement, so leave loading message
      } catch (err) {
        setError(err.message || "Failed to send message");
        setMessages((prev) =>
          prev.filter((msg) => msg.id !== loadingMessageId)
        );
      } finally {
        setSending(false);
      }
    },
    [agentId, model]
  );

  const clearChat = useCallback(async () => {
    try {
      await chatApi.clearChat(agentId);
    } catch (err) {
      console.error("Failed to clear chat history:", err);
    }
    setMessages([]);
    processedMessageIds.current.clear();
    localMessageContents.current.clear();
  }, [agentId]);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    messages,
    sending,
    error,
    isConnected,
    sendMessage,
    clearChat,
    clearError,
  };
}
