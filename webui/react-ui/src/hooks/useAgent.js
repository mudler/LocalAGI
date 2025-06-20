import { useState, useEffect, useCallback } from "react";
import { agentApi } from "../utils/api";
import { useOutletContext } from "react-router-dom";

/**
 * Custom hook for managing agent state
 * @param {string} agentId - Id of the agent to manage
 * @returns {Object} - Agent state and management functions
 */
export function useAgent(agentId) {
  const [agent, setAgent] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const { showToast } = useOutletContext();

  // Fetch agent configuration
  const fetchAgent = useCallback(async () => {
    if (!agentId) return;

    setLoading(true);
    setError(null);

    try {
      // Fetch the agent configuration
      const config = await agentApi.getAgentConfig(agentId);

      // Fetch the agent status
      const response = await fetch(`/api/agent/${agentId}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch agent status: ${response.status}`);
      }
      const statusData = await response.json();

      // Combine configuration with active status
      setAgent({
        ...config,
        active: statusData.active,
      });
    } catch (err) {
      setError(err.message || "Failed to fetch agent configuration");
      console.error("Error fetching agent:", err);
    } finally {
      setLoading(false);
    }
  }, [agentId]);

  // Update agent configuration
  const updateAgent = useCallback(
    async (config) => {
      if (!agentId) return;

      setLoading(true);
      setError(null);

      try {
        await agentApi.updateAgentConfig(agentId, config);
        // Refresh agent data after update
        await fetchAgent();
        return true;
      } catch (err) {
        if(err?.message){
          showToast && showToast(err.message.charAt(0).toUpperCase() + err.message.slice(1), "error");
        } else {
          showToast && showToast("Failed to create agent", "error");
        }
        setError(err.message || "Failed to update agent configuration");
        console.error("Error updating agent:", err);
        return false;
      } finally {
        setLoading(false);
      }
    },
    [agentId, fetchAgent]
  );

  // Delete agent
  const deleteAgent = useCallback(async () => {
    if (!agentId) return;

    setLoading(true);
    setError(null);

    try {
      await agentApi.deleteAgent(agentId);
      setAgent(null);
      return true;
    } catch (err) {
      setError(err.message || "Failed to delete agent");
      console.error("Error deleting agent:", err);
      return false;
    } finally {
      setLoading(false);
    }
  }, [agentId]);

  // Load agent data on mount or when agentId changes
  useEffect(() => {
    fetchAgent();
  }, [agentId, fetchAgent]);

  return {
    agent,
    loading,
    error,
    fetchAgent,
    updateAgent,
    deleteAgent,
    setAgent
  };
}
