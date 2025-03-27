/**
 * Application configuration
 */

// Get the base URL from Vite's environment variables or default to '/app/'
export const BASE_URL = import.meta.env.BASE_URL || '/app/';

// API endpoints configuration
export const API_CONFIG = {
  // Base URL for API requests
  baseUrl: '/',  // API endpoints are at the root, not under /app/
  
  // Default headers for API requests
  headers: {
    'Content-Type': 'application/json',
  },
  
  // Endpoints
  endpoints: {
    // Agent endpoints
    agents: '/api/agents',
    agentConfig: (name) => `/api/agent/${name}/config`,
    agentConfigMetadata: '/api/meta/agent/config',
    createAgent: '/create',
    deleteAgent: (name) => `/api/agent/${name}`,
    pauseAgent: (name) => `/api/agent/${name}/pause`,
    startAgent: (name) => `/api/agent/${name}/start`,
    exportAgent: (name) => `/settings/export/${name}`,
    importAgent: '/settings/import',
    
    // Group endpoints
    generateGroupProfiles: '/api/agent/group/generateProfiles',
    createGroup: '/api/agent/group/create',
    
    // Chat endpoints
    chat: (name) => `/chat/${name}`,
    notify: (name) => `/notify/${name}`,
    responses: '/v1/responses',
    
    // SSE endpoint
    sse: (name) => `/sse/${name}`,
    
    // Action endpoints
    listActions: '/actions',
    executeAction: (name) => `/action/${name}/run`,
    
    // Status endpoint
    status: (name) => `/status/${name}`,
  }
};
