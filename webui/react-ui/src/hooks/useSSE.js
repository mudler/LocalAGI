import { useState, useEffect } from 'react';
import { API_CONFIG } from '../utils/config';

/**
 * Helper function to build a full URL
 * @param {string} endpoint - API endpoint
 * @returns {string} - Full URL
 */
const buildUrl = (endpoint) => {
  return `${API_CONFIG.baseUrl}${endpoint.startsWith('/') ? endpoint.substring(1) : endpoint}`;
};

/**
 * Custom hook for handling Server-Sent Events (SSE)
 * @param {string} agentName - Name of the agent to connect to
 * @returns {Object} - SSE data and connection status
 */
export function useSSE(agentName) {
  const [data, setData] = useState([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (!agentName) return;

    // Create EventSource for SSE connection
    const eventSource = new EventSource(buildUrl(API_CONFIG.endpoints.sse(agentName)));
    
    // Connection opened
    eventSource.onopen = () => {
      setIsConnected(true);
      setError(null);
    };
    
    // Handle incoming messages
    eventSource.onmessage = (event) => {
      try {
        const parsedData = JSON.parse(event.data);
        setData((prevData) => [...prevData, parsedData]);
      } catch (err) {
        console.error('Error parsing SSE data:', err);
      }
    };
    
    // Handle errors
    eventSource.onerror = (err) => {
      setIsConnected(false);
      setError('SSE connection error');
      console.error('SSE connection error:', err);
    };
    
    // Clean up on unmount
    return () => {
      eventSource.close();
      setIsConnected(false);
    };
  }, [agentName]);

  // Function to clear the data
  const clearData = () => setData([]);

  return { data, isConnected, error, clearData };
}
