import { useState, useCallback, useEffect, useRef } from "react";
import { chatApi } from "../utils/api";
import { useSSE } from "./useSSE";

/**
 * Custom hook for chat functionality
 * @param {string} agentName - Name of the agent to chat with
 * @param {Object} model - Model object (should include id)
 * @returns {Object} - Chat state and functions
 */
export function useChat(agentName, model) {
  const [messages, setMessages] = useState([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState(null);
  const processedMessageIds = useRef(new Set());
  const localMessageContents = useRef(new Set()); // Track locally added message contents

  // Use SSE hook to receive real-time messages (only for local models)
  const {
    messages: sseMessages,
    statusUpdates,
    errorMessages,
    isConnected,
  } = useSSE(
    model?.id && typeof model.id === "string"
      ? model.id.split("/")[0] === "local"
        ? agentName
        : null
      : null
  );

  // Process SSE messages into chat messages (local models only)
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
        setMessages((prev) => [
          ...prev,
          {
            id: messageData.id,
            sender: messageData.sender,
            content: messageData.content,
            timestamp: messageData.timestamp,
          },
        ]);
      } catch (err) {
        console.error("Error processing JSON message:", err);
      }
    }
  }, [sseMessages]);

  // Process status updates (local models only)
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

  // Process error messages (local models only)
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

  // Send a message to the agent or OpenRouter
  const sendMessage = useCallback(
    async (content) => {
      if (!model || !content) return false;
      console.log("[useChat] sendMessage: model object:", model); // DEBUG LOG
      setSending(true);
      setError(null);
      try {
        // Add user message to the local state immediately for better UX
        const messageId = `${Date.now()}-${Math.random()
          .toString(36)
          .substr(2, 9)}`;
        const userMessage = {
          id: messageId,
          sender: "user",
          content,
          timestamp: new Date().toISOString(),
        };
        setMessages((prev) => [...prev, userMessage]);
        localMessageContents.current.add(content);
        if (model.split("/")[0] === "openrouter") {
          // Send to backend proxy endpoint
          const res = await fetch("/api/openrouter/chat", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              model: model.replace("openrouter/", ""),
              messages: [{ role: "user", content }],
            }),
          });
          if (!res.ok) {
            const err = await res.json();
            const errorMsg =
              (typeof err.error === "string" && err.error) ||
              err.error?.message ||
              JSON.stringify(err.error) ||
              "Failed to get OpenRouter response";
            throw new Error(errorMsg);
          }
          const data = await res.json();
          // OpenRouter returns choices[0].message
          const reply = data.choices?.[0]?.message;
          if (reply) {
            setMessages((prev) => [
              ...prev,
              {
                id: `${messageId}-openrouter`,
                sender: "assistant",
                content: reply.content,
                timestamp: new Date().toISOString(),
              },
            ]);
          }
        } else {
          // Use the JSON-based API endpoint for local models
          await chatApi.sendMessage(agentName, content);
        }
        return true;
      } catch (err) {
        setError(err.message || "Failed to send message");
        console.error("Error sending message:", err);
      } finally {
        setSending(false);
      }
    },
    [agentName, model]
  );

  // Clear chat history
  const clearChat = useCallback(() => {
    setMessages([]);
    processedMessageIds.current.clear();
    localMessageContents.current.clear();
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
    sendMessage,
    clearChat,
    clearError,
  };
}
