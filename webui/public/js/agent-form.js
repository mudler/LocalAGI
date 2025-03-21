// Common utility functions for agent forms
const AgentFormUtils = {
    // Add dynamic component based on template
    addDynamicComponent: function(sectionId, templateFunction, dataItems) {
        const section = document.getElementById(sectionId);
        const newIndex = section.getElementsByClassName(dataItems.className).length;
        
        // Generate HTML from template function
        const newHtml = templateFunction(newIndex, dataItems);
        
        // Add to DOM
        section.insertAdjacentHTML('beforeend', newHtml);
    },
    
    // Process form data into JSON structure
    processFormData: function(formData) {
        const jsonData = {};
        
        // Process basic form fields
        for (const [key, value] of formData.entries()) {
            // Skip the array fields as they'll be processed separately
            if (!key.includes('[') && !key.includes('].')) {
                // Handle checkboxes
                if (value === 'on') {
                    jsonData[key] = true;
                } 
                // Handle numeric fields - specifically kb_results
                else if (key === 'kb_results') {
                    // Convert to integer or default to 3 if empty
                    jsonData[key] = value ? parseInt(value, 10) : 3;
                    
                    // Check if the parse was successful
                    if (isNaN(jsonData[key])) {
                        showToast('Knowledge Base Results must be a number', 'error');
                        return null; // Indicate validation error
                    }
                }
                // Handle other numeric fields if needed
                else if (key === 'periodic_runs' && value) {
                    // Try to parse as number if it looks like one
                    const numValue = parseInt(value, 10);
                    if (!isNaN(numValue) && String(numValue) === value) {
                        jsonData[key] = numValue;
                    } else {
                        jsonData[key] = value;
                    }
                }
                else {
                    jsonData[key] = value;
                }
            }
        }
        
        return jsonData;
    },
    
    // Process connectors from form
    processConnectors: function(button) {
        const connectors = [];
        const connectorElements = document.querySelectorAll('.connector');
        
        for (let i = 0; i < connectorElements.length; i++) {
            const typeSelect = document.getElementById(`connectorType${i}`);
            if (!typeSelect) {
                showToast(`Error: Could not find connector type select for index ${i}`, 'error');
                button.innerHTML = button.getAttribute('data-original-text');
                button.disabled = false;
                return null; // Validation failed
            }
            
            const type = typeSelect.value;
            if (!type) {
                showToast(`Please select a connector type for connector ${i+1}`, 'error');
                button.innerHTML = button.getAttribute('data-original-text');
                button.disabled = false;
                return null; // Validation failed
            }
            
            // Get all config fields for this connector
            const connector = {
                type: type,
                config: {}
            };
            
            // Find all config inputs for this connector
            const configInputs = document.querySelectorAll(`[name^="connectors[${i}][config]"]`);
            
            // Check if we have a JSON textarea (fallback template)
            const jsonTextarea = document.getElementById(`connectorConfig${i}`);
            if (jsonTextarea && jsonTextarea.value) {
                try {
                    // If it's a JSON textarea, parse it and use the result
                    const jsonConfig = JSON.parse(jsonTextarea.value);
                    // Convert the parsed JSON back to a string for the backend
                    connector.config = JSON.stringify(jsonConfig);
                } catch (e) {
                    // If it's not valid JSON, use it as is
                    connector.config = jsonTextarea.value;
                }
            } else {
                // Process individual form fields
                configInputs.forEach(input => {
                    // Extract the key from the name attribute
                    // Format: connectors[0][config][key]
                    const keyMatch = input.name.match(/\[config\]\[([^\]]+)\]/);
                    if (keyMatch && keyMatch[1]) {
                        const key = keyMatch[1];
                        // For checkboxes, set true/false based on checked state
                        if (input.type === 'checkbox') {
                            connector.config[key] = input.checked ? 'true' : 'false';
                        } else {
                            connector.config[key] = input.value;
                        }
                    }
                });
                
                // Convert the config object to a JSON string for the backend
                connector.config = JSON.stringify(connector.config);
            }
            
            connectors.push(connector);
        }
        
        return connectors;
    },
    
    // Process MCP servers from form
    processMCPServers: function() {
        const mcpServers = [];
        const mcpElements = document.querySelectorAll('.mcp_server');
        
        for (let i = 0; i < mcpElements.length; i++) {
            const urlInput = document.getElementById(`mcpURL${i}`);
            const tokenInput = document.getElementById(`mcpToken${i}`);
            
            if (urlInput && urlInput.value) {
                const server = {
                    url: urlInput.value
                };
                
                // Add token if present
                if (tokenInput && tokenInput.value) {
                    server.token = tokenInput.value;
                }
                
                mcpServers.push(server);
            }
        }
        
        return mcpServers;
    },
    
    // Process actions from form
    processActions: function(button) {
        const actions = [];
        const actionElements = document.querySelectorAll('.action');
        
        for (let i = 0; i < actionElements.length; i++) {
            const nameSelect = document.getElementById(`actionsName${i}`);
            const configTextarea = document.getElementById(`actionsConfig${i}`);
            
            if (!nameSelect) {
                showToast(`Error: Could not find action name select for index ${i}`, 'error');
                button.innerHTML = button.getAttribute('data-original-text');
                button.disabled = false;
                return null; // Validation failed
            }
            
            const name = nameSelect.value;
            if (!name) {
                showToast(`Please select an action type for action ${i+1}`, 'error');
                button.innerHTML = button.getAttribute('data-original-text');
                button.disabled = false;
                return null; // Validation failed
            }
            
            let config = {};
            if (configTextarea && configTextarea.value) {
                try {
                    config = JSON.parse(configTextarea.value);
                } catch (e) {
                    showToast(`Invalid JSON in action ${i+1} config: ${e.message}`, 'error');
                    button.innerHTML = button.getAttribute('data-original-text');
                    button.disabled = false;
                    return null; // Validation failed
                }
            }
            
            actions.push({
                name: name,
                config: JSON.stringify(config) // Convert to JSON string for backend
            });
        }
        
        return actions;
    },
    
    // Process prompt blocks from form
    processPromptBlocks: function(button) {
        const promptBlocks = [];
        const promptElements = document.querySelectorAll('.prompt_block');
        
        for (let i = 0; i < promptElements.length; i++) {
            const nameSelect = document.getElementById(`promptName${i}`);
            const configTextarea = document.getElementById(`promptConfig${i}`);
            
            if (!nameSelect) {
                showToast(`Error: Could not find prompt block name select for index ${i}`, 'error');
                button.innerHTML = button.getAttribute('data-original-text');
                button.disabled = false;
                return null; // Validation failed
            }
            
            const name = nameSelect.value;
            if (!name) {
                showToast(`Please select a prompt block type for block ${i+1}`, 'error');
                button.innerHTML = button.getAttribute('data-original-text');
                button.disabled = false;
                return null; // Validation failed
            }
            
            let config = {};
            if (configTextarea && configTextarea.value) {
                try {
                    config = JSON.parse(configTextarea.value);
                } catch (e) {
                    showToast(`Invalid JSON in prompt block ${i+1} config: ${e.message}`, 'error');
                    button.innerHTML = button.getAttribute('data-original-text');
                    button.disabled = false;
                    return null; // Validation failed
                }
            }
            
            promptBlocks.push({
                name: name,
                config: JSON.stringify(config) // Convert to JSON string for backend
            });
        }
        
        return promptBlocks;
    },
    
    // Helper function to format config values (for edit form)
    formatConfigValue: function(configElement, configValue) {
        if (!configElement) return;
        
        // If configValue is an object, stringify it
        if (typeof configValue === 'object' && configValue !== null) {
            try {
                configElement.value = JSON.stringify(configValue, null, 2);
            } catch (e) {
                console.error('Error stringifying config value:', e);
                configElement.value = '{}';
            }
        } 
        // If it's a string that looks like JSON, try to parse and pretty print it
        else if (typeof configValue === 'string' && (configValue.startsWith('{') || configValue.startsWith('['))) {
            try {
                const parsed = JSON.parse(configValue);
                configElement.value = JSON.stringify(parsed, null, 2);
            } catch (e) {
                // If it's not valid JSON, just use the string as is
                configElement.value = configValue;
            }
        }
        // Otherwise, just use the value as is
        else {
            configElement.value = configValue || '';
        }
    },
    
    // Helper function to set select value (with fallback if option doesn't exist)
    setSelectValue: function(selectElement, value) {
        if (!selectElement) return;
        
        // Check if the option exists
        let optionExists = false;
        for (let i = 0; i < selectElement.options.length; i++) {
            if (selectElement.options[i].value === value) {
                optionExists = true;
                break;
            }
        }
        
        // Set the value if the option exists
        if (optionExists) {
            selectElement.value = value;
        } else if (selectElement.options.length > 0) {
            // Otherwise, select the first option
            selectElement.selectedIndex = 0;
        }
    },

    // Render connector form based on type
    renderConnectorForm: function(index, type, config = {}) {
        const formContainer = document.getElementById(`connectorFormContainer${index}`);
        if (!formContainer) return;

        // Clear existing form
        formContainer.innerHTML = '';

        // Debug log to see what's happening
        console.log(`Rendering connector form for type: ${type}`);
        console.log(`Config for connector:`, config);
        console.log(`Available templates:`, ConnectorTemplates ? Object.keys(ConnectorTemplates) : 'None');

        // Ensure config is an object
        let configObj = config;
        if (typeof config === 'string') {
            try {
                configObj = JSON.parse(config);
            } catch (e) {
                console.error('Error parsing connector config string:', e);
                configObj = {};
            }
        }

        // If we have a template for this connector type in the global ConnectorTemplates object
        if (ConnectorTemplates && type && ConnectorTemplates[type]) {
            console.log(`Found template for ${type}`);
            // Get the template result which contains HTML and setValues function
            const templateResult = ConnectorTemplates[type](configObj, index);
            
            // Set the HTML content
            formContainer.innerHTML = templateResult.html;
            
            // Call the setValues function to set input values safely
            if (typeof templateResult.setValues === 'function') {
                setTimeout(templateResult.setValues, 0);
            }
        } else {
            console.log(`No template found for ${type}, using fallback`);
            // Use the fallback template
            if (ConnectorTemplates && ConnectorTemplates.fallback) {
                const fallbackResult = ConnectorTemplates.fallback(configObj, index);
                formContainer.innerHTML = fallbackResult.html;
                
                if (typeof fallbackResult.setValues === 'function') {
                    setTimeout(fallbackResult.setValues, 0);
                }
            } else {
                // Fallback to generic JSON textarea if no fallback template
                formContainer.innerHTML = `
                    <div class="form-group">
                        <label for="connectorConfig${index}">Connector Config (JSON)</label>
                        <textarea id="connectorConfig${index}" 
                                  name="connectors[${index}][config]" 
                                  class="form-control"
                                  placeholder='{"key":"value"}'></textarea>
                    </div>
                `;
                
                // Set the value safely after DOM is created
                setTimeout(function() {
                    const configTextarea = document.getElementById(`connectorConfig${index}`);
                    if (configTextarea) {
                        if (typeof configObj === 'object' && configObj !== null) {
                            configTextarea.value = JSON.stringify(configObj, null, 2);
                        } else if (typeof config === 'string') {
                            configTextarea.value = config;
                        }
                    }
                }, 0);
            }
        }
    }
};

// HTML Templates for dynamic elements
const AgentFormTemplates = {
    // Connector template
    connectorTemplate: function(index, data) {
        return `
            <div class="connector mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>Connector ${index + 1}</h2>
                <div class="mb-4">
                    <label for="connectorType${index}">Connector Type</label>
                    <select name="connectors[${index}][type]" 
                            id="connectorType${index}" 
                            class="form-control" 
                            onchange="AgentFormUtils.renderConnectorForm(${index}, this.value)">
                        <option value="">Select Connector Type</option>
                        ${data.options}
                    </select>
                </div>
                <div id="connectorFormContainer${index}">
                    <!-- Connector form will be dynamically inserted here -->
                    <div class="form-group">
                        <div class="placeholder-text">Select a connector type to configure</div>
                    </div>
                </div>
                <button type="button" class="remove-btn" onclick="this.closest('.connector').remove()">
                    <i class="fas fa-trash"></i> Remove Connector
                </button>
            </div>
        `;
    },
    
    // MCP Server template
    mcpServerTemplate: function(index, data) {
        return `
            <div class="mcp_server mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>MCP Server ${index + 1}</h2>
                <div class="mb-4">
                    <label for="mcpURL${index}">Server URL</label>
                    <input type="text" id="mcpURL${index}" name="mcp_servers[${index}][url]" placeholder="https://example.com">
                </div>
                <div class="mb-4">
                    <label for="mcpToken${index}">API Token (Optional)</label>
                    <input type="text" id="mcpToken${index}" name="mcp_servers[${index}][token]" placeholder="API token">
                </div>
                <button type="button" class="remove-btn" onclick="this.closest('.mcp_server').remove()">
                    <i class="fas fa-trash"></i> Remove Server
                </button>
            </div>
        `;
    },
    
    // Action template
    actionTemplate: function(index, data) {
        return `
            <div class="action mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>Action ${index + 1}</h2>
                <div class="mb-4">
                    <label for="actionsName${index}">Action Type</label>
                    <select name="actions[${index}][name]" id="actionsName${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="actionsConfig${index}">Action Config (JSON)</label>
                    <textarea id="actionsConfig${index}" name="actions[${index}][config]" placeholder='{"key":"value"}'>{}</textarea>
                </div>
                <button type="button" class="remove-btn" onclick="this.closest('.action').remove()">
                    <i class="fas fa-trash"></i> Remove Action
                </button>
            </div>
        `;
    },
    
    // Prompt Block template
    promptBlockTemplate: function(index, data) {
        return `
            <div class="prompt_block mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>Prompt Block ${index + 1}</h2>
                <div class="mb-4">
                    <label for="promptName${index}">Prompt Block Type</label>
                    <select name="promptblocks[${index}][name]" id="promptName${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="promptConfig${index}">Prompt Block Config (JSON)</label>
                    <textarea id="promptConfig${index}" name="promptblocks[${index}][config]" placeholder='{"key":"value"}'>{}</textarea>
                </div>
                <button type="button" class="remove-btn" onclick="this.closest('.prompt_block').remove()">
                    <i class="fas fa-trash"></i> Remove Prompt Block
                </button>
            </div>
        `;
    }
};

// Initialize form event listeners
function initAgentFormCommon(options = {}) {
    // Add connector button
    const addConnectorButton = document.getElementById('addConnectorButton');
    if (addConnectorButton) {
        addConnectorButton.addEventListener('click', function() {
            // Create options string
            let optionsHtml = '';
            if (options.connectors) {
                optionsHtml = options.connectors;
            }
            
            // Add new connector form
            AgentFormUtils.addDynamicComponent('connectorsSection', AgentFormTemplates.connectorTemplate, {
                className: 'connector',
                options: optionsHtml
            });
        });
    }
    
    // Add MCP server button
    const addMCPButton = document.getElementById('addMCPButton');
    if (addMCPButton) {
        addMCPButton.addEventListener('click', function() {
            // Add new MCP server form
            AgentFormUtils.addDynamicComponent('mcpSection', AgentFormTemplates.mcpServerTemplate, {
                className: 'mcp_server'
            });
        });
    }
    
    // Add action button
    const actionButton = document.getElementById('action_button');
    if (actionButton) {
        actionButton.addEventListener('click', function() {
            // Create options string
            let optionsHtml = '';
            if (options.actions) {
                optionsHtml = options.actions;
            }
            
            // Add new action form
            AgentFormUtils.addDynamicComponent('action_box', AgentFormTemplates.actionTemplate, {
                className: 'action',
                options: optionsHtml
            });
        });
    }
    
    // Add prompt block button
    const dynamicButton = document.getElementById('dynamic_button');
    if (dynamicButton) {
        dynamicButton.addEventListener('click', function() {
            // Create options string
            let optionsHtml = '';
            if (options.promptBlocks) {
                optionsHtml = options.promptBlocks;
            }
            
            // Add new prompt block form
            AgentFormUtils.addDynamicComponent('dynamic_box', AgentFormTemplates.promptBlockTemplate, {
                className: 'prompt_block',
                options: optionsHtml
            });
        });
    }
}

// Simple toast notification function
function showToast(message, type) {
    // Check if toast container exists, if not create it
    let toast = document.getElementById('toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'toast';
        toast.className = 'toast';
        
        const toastMessage = document.createElement('div');
        toastMessage.id = 'toast-message';
        toast.appendChild(toastMessage);
        
        document.body.appendChild(toast);
    }
    
    const toastMessage = document.getElementById('toast-message');
    
    // Set message
    toastMessage.textContent = message;
    
    // Set type class
    toast.className = 'toast';
    toast.classList.add(`toast-${type}`);
    
    // Show toast
    toast.classList.add('show');
    
    // Hide after 3 seconds
    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}
