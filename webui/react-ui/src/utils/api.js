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

// Collections API returns { success, message, data, error }. Throw if !ok or !success.
const handleCollectionsResponse = async (response) => {
  const data = await response.json().catch(() => ({}));
  if (!response.ok || data.success === false) {
    const msg = data.error?.message || data.error?.details || data.message || `API error: ${response.status}`;
    throw new Error(msg);
  }
  return data;
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

// Skills API (skills are stored under state dir / skills, not configurable)
export const skillsApi = {
  getConfig: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillsConfig));
    return handleResponse(response);
  },
  list: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillsList));
    return handleResponse(response);
  },
  search: async (q) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillsSearch(q)));
    return handleResponse(response);
  },
  get: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skill(name)));
    return handleResponse(response);
  },
  create: async (data) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillsList), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify(data),
    });
    return handleResponse(response);
  },
  update: async (name, data) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skill(name)), {
      method: 'PUT',
      headers: API_CONFIG.headers,
      body: JSON.stringify(data),
    });
    return handleResponse(response);
  },
  delete: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skill(name)), { method: 'DELETE' });
    if (response.status === 204) return;
    return handleResponse(response);
  },
  import: async (file) => {
    const form = new FormData();
    form.append('file', file);
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillsImport), {
      method: 'POST',
      body: form,
    });
    return handleResponse(response);
  },
  exportUrl: (name) => buildUrl(API_CONFIG.endpoints.skillExport(name)),
  listResources: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillResources(name)));
    return handleResponse(response);
  },
  getResource: async (name, path, { json = false } = {}) => {
    const url = buildUrl(API_CONFIG.endpoints.skillResource(name, path)) + (json ? '?encoding=base64' : '');
    const response = await fetch(url, { credentials: 'same-origin' });
    if (!response.ok) {
      const err = await response.json().catch(() => ({}));
      throw new Error(err.error || `Failed to get resource: ${response.status}`);
    }
    if (json) return response.json();
    const ct = response.headers.get('content-type') || '';
    if (ct.includes('application/json')) return response.json();
    if (ct.includes('text/') || ct.includes('application/javascript')) return response.text();
    return response.blob();
  },
  createResource: async (name, path, file) => {
    const form = new FormData();
    form.append('file', file);
    form.append('path', path);
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillResources(name)), {
      method: 'POST',
      body: form,
      credentials: 'same-origin',
    });
    if (!response.ok) {
      const err = await response.json().catch(() => ({}));
      throw new Error(err.error || `Failed to create resource: ${response.status}`);
    }
    return response.json();
  },
  updateResource: async (name, path, content) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillResource(name, path)), {
      method: 'PUT',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ content }),
      credentials: 'same-origin',
    });
    if (response.status !== 204) {
      const err = await response.json().catch(() => ({}));
      throw new Error(err.error || `Failed to update resource: ${response.status}`);
    }
  },
  deleteResource: async (name, path) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.skillResource(name, path)), {
      method: 'DELETE',
      credentials: 'same-origin',
    });
    if (response.status !== 204) {
      const err = await response.json().catch(() => ({}));
      throw new Error(err.error || `Failed to delete resource: ${response.status}`);
    }
  },
  listGitRepos: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.gitRepos));
    return handleResponse(response);
  },
  addGitRepo: async (url) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.gitRepos), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ url }),
    });
    return handleResponse(response);
  },
  updateGitRepo: async (id, data) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.gitRepo(id)), {
      method: 'PUT',
      headers: API_CONFIG.headers,
      body: JSON.stringify(data),
    });
    return handleResponse(response);
  },
  deleteGitRepo: async (id) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.gitRepo(id)), { method: 'DELETE' });
    if (response.status === 204) return;
    return handleResponse(response);
  },
  syncGitRepo: async (id) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.gitRepoSync(id)), { method: 'POST' });
    return handleResponse(response);
  },
  toggleGitRepo: async (id) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.gitRepoToggle(id)), { method: 'POST' });
    return handleResponse(response);
  },
};

// Collections / knowledge base API (LocalRecall-compatible)
export const collectionsApi = {
  list: async () => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collections));
    const data = await handleCollectionsResponse(response);
    return data.data?.collections || [];
  },
  create: async (name) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collections), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ name }),
    });
    return handleCollectionsResponse(response);
  },
  upload: async (collectionName, file) => {
    const form = new FormData();
    form.append('file', file);
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionUpload(collectionName)), {
      method: 'POST',
      body: form,
    });
    return handleCollectionsResponse(response);
  },
  listEntries: async (collectionName) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionEntries(collectionName)));
    const data = await handleCollectionsResponse(response);
    return data.data?.entries || [];
  },
  getEntryContent: async (collectionName, entry) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionEntry(collectionName, entry)));
    const data = await handleCollectionsResponse(response);
    return { content: data.data?.content ?? '', chunkCount: data.data?.chunk_count ?? 0, entry: data.data?.entry ?? entry };
  },
  search: async (collectionName, query, maxResults = 5) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionSearch(collectionName)), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ query, max_results: maxResults }),
    });
    const data = await handleCollectionsResponse(response);
    return data.data?.results || [];
  },
  reset: async (collectionName) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionReset(collectionName)), {
      method: 'POST',
      headers: API_CONFIG.headers,
    });
    return handleCollectionsResponse(response);
  },
  deleteEntry: async (collectionName, entry) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionDeleteEntry(collectionName)), {
      method: 'DELETE',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ entry }),
    });
    return handleCollectionsResponse(response);
  },
  listSources: async (collectionName) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionSources(collectionName)));
    const data = await handleCollectionsResponse(response);
    return data.data?.sources || [];
  },
  addSource: async (collectionName, url, updateIntervalMinutes = 60) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionSources(collectionName)), {
      method: 'POST',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ url, update_interval: updateIntervalMinutes }),
    });
    return handleCollectionsResponse(response);
  },
  removeSource: async (collectionName, url) => {
    const response = await fetch(buildUrl(API_CONFIG.endpoints.collectionSources(collectionName)), {
      method: 'DELETE',
      headers: API_CONFIG.headers,
      body: JSON.stringify({ url }),
    });
    return handleCollectionsResponse(response);
  },
};
