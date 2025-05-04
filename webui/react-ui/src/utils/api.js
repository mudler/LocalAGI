/**
 * API utility for communicating with the Go backend
 */
import { API_CONFIG } from './config';

// Helper function for handling API responses
const handleResponse = async (response) => {
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `API error: ${response.status}`);
  }
  
  // Check if response is JSON
  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    return response.json();
  }
  
  return response.text();
};

// Helper function to build a full URL
const buildUrl = (endpoint) => {
  return `${API_CONFIG.baseUrl}${endpoint.startsWith('/') ? endpoint.substring(1) : endpoint}`;
};

// Helper function to convert ActionDefinition to FormFieldDefinition format
const convertActionDefinitionToFields = (definition) => {
  if (!definition || !definition.Properties) {
    return [];
  }

  const fields = [];
  const required = definition.Required || [];

  console.debug('Action definition:', definition);

  Object.entries(definition.Properties).forEach(([name, property]) => {
    const field = {
      name,
      label: name.charAt(0).toUpperCase() + name.slice(1),
      type: 'text', // Default to text, we'll enhance this later
      required: required.includes(name),
      helpText: property.Description || '',
      defaultValue: property.Default,
    };

    if (property.enum && property.enum.length > 0) {
      field.type = 'select';
      field.options = property.enum;
    } else {
      switch (property.type) {
        case 'integer':
          field.type = 'number';
          field.min = property.Minimum;
          field.max = property.Maximum;
        break;
      case 'boolean':
        field.type = 'checkbox';
        break;
    }
    // TODO: Handle Object and Array types which require nested fields
  }

    fields.push(field);
  });

  return fields;
};

// Agent-related API calls
export const agentApi = {
  // Get list of all agents
  getAgents: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.agents), {
      headers: API_CONFIG.headers
    });
    return handleResponse(response);
  },
  
  // Get a specific agent's configuration
  getAgentConfig: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.agentConfig(name)), {
      headers: API_CONFIG.headers
    });
    return handleResponse(response);
  },
  
  // Get agent configuration metadata
  getAgentConfigMetadata: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.agentConfigMetadata), {
      headers: API_CONFIG.headers
    });
    const metadata = await handleResponse(response);
    
    // Process metadata to group by section
    if (metadata) {
      const groupedMetadata = {};
      
      // Handle Fields - Group by section
      if (metadata.Fields) {
        metadata.Fields.forEach(field => {
          const section = field.tags?.section || 'Other';
          const sectionKey = `${section}Section`; // Add "Section" postfix
          
          if (!groupedMetadata[sectionKey]) {
            groupedMetadata[sectionKey] = [];
          }
          
          groupedMetadata[sectionKey].push(field);
        });
      }
      
      // Pass through connectors and actions field groups directly
      // Make sure to assign the correct metadata to each section
      if (metadata.Connectors) {
        groupedMetadata.connectors = metadata.Connectors;
      }
      
      if (metadata.Actions) {
        groupedMetadata.actions = metadata.Actions;
      }
      groupedMetadata.dynamicPrompts = metadata.DynamicPrompts;
      groupedMetadata.filters = metadata.Filters;

      return groupedMetadata;
    }
    
    return metadata;
  },
  
  // Create a new agent
  createAgent: async (config) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.createAgent), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(config),
    });
    return handleResponse(response);
  },
  
  // Update an existing agent's configuration
  updateAgentConfig: async (name, config) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.agentConfig(name)), {
      method: 'PUT',
      headers: API_CONFIG.headers,
      body: JSON.stringify(config),
    });
    return handleResponse(response);
  },
  
  // Delete an agent
  deleteAgent: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.deleteAgent(name)), {
      method: 'DELETE',
      headers: API_CONFIG.headers,
    });
    return handleResponse(response);
  },
  
  // Pause an agent
  pauseAgent: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.pauseAgent(name)), {
      method: 'PUT',
      headers: API_CONFIG.headers,
      body: JSON.stringify({}),
    });
    return handleResponse(response);
  },
  
  // Start an agent
  startAgent: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.startAgent(name)), {
      method: 'PUT',
      headers: API_CONFIG.headers,
      body: JSON.stringify({}),
    });
    return handleResponse(response);
  },
  
  // Export agent configuration
  exportAgentConfig: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.exportAgent(name)), {
      headers: API_CONFIG.headers
    });
    return handleResponse(response);
  },
  
  // Import agent configuration
  importAgent: async (formData) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.importAgent), {
      method: 'POST',
      body: formData,
    });
    return handleResponse(response);
  },
  
  // Generate group profiles
  generateGroupProfiles: async (data) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.generateGroupProfiles), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(data),
    });
    return handleResponse(response);
  },
  
  // Create a group of agents
  createGroup: async (data) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.createGroup), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(data),
    });
    return handleResponse(response);
  },
};

// Chat-related API calls
export const chatApi = {
  // Send a message to an agent using the JSON-based API
  sendMessage: async (name, message) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.chat(name)), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ message }),
    });
    return handleResponse(response);
  },
  
  // Send a notification to an agent
  sendNotification: async (name, message) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.notify(name)), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: new URLSearchParams({ message }),
    });
    return handleResponse(response);
  },
  
  // Get responses from an agent
  getResponses: async (data) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.responses), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(data),
    });
    return handleResponse(response);
  },
};

// Action-related API calls
export const actionApi = {
  // List available actions
  listActions: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.listActions), {
      headers: API_CONFIG.headers
    });
    return handleResponse(response);
  },

  // Get action definition
  getActionDefinition: async (name, config = {}) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.actionDefinition(name)), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(config),
    });
    const definition = await handleResponse(response);
    return convertActionDefinitionToFields(definition);
  },

  // Execute an action for an agent
  executeAction: async (name, actionData) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.executeAction(name)), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(actionData),
    });
    return handleResponse(response);
  },
};

// Status-related API calls
export const statusApi = {
  // Get agent status history
  getStatusHistory: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.status(name)), {
      headers: API_CONFIG.headers
    });
    return handleResponse(response);
  },
};
