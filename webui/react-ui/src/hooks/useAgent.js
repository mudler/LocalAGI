import { useState, useEffect, useCallback } from 'react';
import { agentApi } from '../utils/api';

/**
 * Custom hook for managing agent state
 * @param {string} agentName - Name of the agent to manage
 * @returns {Object} - Agent state and management functions
 */
export function useAgent(agentName) {
  const [agent, setAgent] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Fetch agent configuration
  const fetchAgent = useCallback(async () => {
    if (!agentName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      // Fetch the agent configuration
      const config = await agentApi.getAgentConfig(agentName);
      
      // Fetch the agent status
      const response = await fetch(`/api/agent/${agentName}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch agent status: ${response.status}`);
      }
      const statusData = await response.json();
      
      // Combine configuration with active status
      setAgent({
        ...config,
        active: statusData.active
      });
    } catch (err) {
      setError(err.message || 'Failed to fetch agent configuration');
      console.error('Error fetching agent:', err);
    } finally {
      setLoading(false);
    }
  }, [agentName]);

  // Update agent configuration
  const updateAgent = useCallback(async (config) => {
    if (!agentName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      await agentApi.updateAgentConfig(agentName, config);
      // Refresh agent data after update
      await fetchAgent();
      return true;
    } catch (err) {
      setError(err.message || 'Failed to update agent configuration');
      console.error('Error updating agent:', err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [agentName, fetchAgent]);

  // Toggle agent status (pause/start)
  const toggleAgentStatus = useCallback(async (isActive) => {
    if (!agentName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      if (isActive) {
        await agentApi.pauseAgent(agentName);
      } else {
        await agentApi.startAgent(agentName);
      }
      
      // Update the agent's active status in the local state
      setAgent(prevAgent => ({
        ...prevAgent,
        active: !isActive
      }));
      
      return true;
    } catch (err) {
      setError(err.message || 'Failed to toggle agent status');
      console.error('Error toggling agent status:', err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [agentName]);

  // Delete agent
  const deleteAgent = useCallback(async () => {
    if (!agentName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      await agentApi.deleteAgent(agentName);
      setAgent(null);
      return true;
    } catch (err) {
      setError(err.message || 'Failed to delete agent');
      console.error('Error deleting agent:', err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [agentName]);

  // Load agent data on mount or when agentName changes
  useEffect(() => {
    fetchAgent();
  }, [agentName, fetchAgent]);

  return {
    agent,
    loading,
    error,
    fetchAgent,
    updateAgent,
    toggleAgentStatus,
    deleteAgent,
  };
}
