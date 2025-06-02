import { useState, useEffect, useCallback, useRef } from "react";
import { API_CONFIG } from "../utils/config";

/**
 * Custom hook for Server-Sent Events (SSE)
 * @param {string} agentId - Id of the agent to connect to
 * @returns {Object} - SSE state and messages
 */
export function useSSE(agentId) {
  const [messages, setMessages] = useState([]);
  const [statusUpdates, setStatusUpdates] = useState([]);
  const [errorMessages, setErrorMessages] = useState([]);
  const [isConnected, setIsConnected] = useState(false);
  const eventSourceRef = useRef(null);

  console.log("ag id", agentId);

  // Connect to SSE endpoint
  const connect = useCallback(() => {
    console.log("AGENT ID", agentId);
    if (!agentId) return;

    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Create a new EventSource connection
    const sseUrl = new URL(
      `${API_CONFIG.endpoints.sse(agentId)}`,
      window.location.origin
    ).href;
    const eventSource = new EventSource(sseUrl);
    eventSourceRef.current = eventSource;

    // Handle connection open
    eventSource.onopen = () => {
      console.log("SSE connection opened");
      setIsConnected(true);
    };

    // Handle connection error
    eventSource.onerror = (error) => {
      console.error("SSE connection error:", error);
      setIsConnected(false);

      // Try to reconnect after a delay
      setTimeout(() => {
        if (eventSourceRef.current === eventSource) {
          connect();
        }
      }, 5000);
    };

    // Handle 'json_message' event
    eventSource.addEventListener("json_message", (event) => {
      try {
        const data = JSON.parse(event.data);
        const timestamp = data.timestamp || new Date().toISOString();

        setMessages((prev) => [
          ...prev,
          {
            id: `json-message-${Date.now()}`,
            type: "json_message",
            content: data,
            timestamp,
          },
        ]);
      } catch (error) {
        console.error("Error parsing JSON message:", error);
      }
    });

    // Handle 'json_status' event
    eventSource.addEventListener("json_status", (event) => {
      try {
        const data = JSON.parse(event.data);
        const timestamp = data.timestamp || new Date().toISOString();

        setStatusUpdates((prev) => [
          ...prev,
          {
            id: `json-status-${Date.now()}`,
            type: "status",
            content: data,
            timestamp,
          },
        ]);
      } catch (error) {
        console.error("Error parsing status message:", error);
      }
    });

    // Handle 'error' event
    eventSource.addEventListener("json_error", (event) => {
      try {
        const data = JSON.parse(event.data);
        const timestamp = data.timestamp || new Date().toISOString();

        setErrorMessages((prev) => [
          ...prev,
          {
            id: `error-${Date.now()}`,
            type: "error",
            content: data,
            timestamp,
          },
        ]);
      } catch (error) {
        console.error("Error parsing error message:", error);
      }
    });

    return () => {
      eventSource.close();
    };
  }, [agentId]);

  // Connect on mount and when agentId changes
  useEffect(() => {
    connect();

    // Cleanup on unmount
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [connect]);

  // Reconnect function
  const reconnect = useCallback(() => {
    connect();
  }, [connect]);

  return {
    messages,
    statusUpdates,
    errorMessages,
    isConnected,
    reconnect,
  };
}
