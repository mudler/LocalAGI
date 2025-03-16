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
        const connectorsElements = document.getElementsByClassName('connector');
        
        for (let i = 0; i < connectorsElements.length; i++) {
            const typeField = document.getElementById(`connectorType${i}`);
            const configField = document.getElementById(`connectorConfig${i}`);
            
            if (typeField && configField) {
                try {
                    // Validate JSON but send as string
                    const configValue = configField.value.trim() || '{}';
                    // Parse to validate but don't use the parsed object
                    JSON.parse(configValue); 
                    
                    connectors.push({
                        type: typeField.value,
                        config: configValue // Send the raw string, not parsed JSON
                    });
                } catch (err) {
                    console.error(`Error parsing connector ${i} config:`, err);
                    showToast(`Error in connector ${i+1} configuration: Invalid JSON`, 'error');
                    
                    // If button is provided, restore its state
                    if (button) {
                        const originalButtonText = button.getAttribute('data-original-text');
                        button.innerHTML = originalButtonText;
                        button.disabled = false;
                    }
                    
                    return null; // Indicate validation error
                }
            }
        }
        
        return connectors;
    },
    
    // Process MCP servers from form
    processMCPServers: function() {
        const mcpServers = [];
        const mcpServerElements = document.getElementsByClassName('mcp_server');
        
        for (let i = 0; i < mcpServerElements.length; i++) {
            const urlField = document.getElementById(`mcpURL${i}`);
            const tokenField = document.getElementById(`mcpToken${i}`);
            
            if (urlField && urlField.value.trim()) {
                mcpServers.push({
                    url: urlField.value.trim(),
                    token: tokenField ? tokenField.value.trim() : ''
                });
            }
        }
        
        return mcpServers;
    },
    
    // Process actions from form
    processActions: function(button) {
        const actions = [];
        const actionElements = document.getElementsByClassName('action');
        
        for (let i = 0; i < actionElements.length; i++) {
            const nameField = document.getElementById(`actionsName${i}`);
            const configField = document.getElementById(`actionsConfig${i}`);
            
            if (nameField && configField) {
                try {
                    // Validate JSON but send as string
                    const configValue = configField.value.trim() || '{}';
                    // Parse to validate but don't use the parsed object
                    JSON.parse(configValue);
                    
                    actions.push({
                        name: nameField.value,
                        config: configValue // Send the raw string, not parsed JSON
                    });
                } catch (err) {
                    console.error(`Error parsing action ${i} config:`, err);
                    showToast(`Error in action ${i+1} configuration: Invalid JSON`, 'error');
                    
                    // If button is provided, restore its state
                    if (button) {
                        const originalButtonText = button.getAttribute('data-original-text');
                        button.innerHTML = originalButtonText;
                        button.disabled = false;
                    }
                    
                    return null; // Indicate validation error
                }
            }
        }
        
        return actions;
    },
    
    // Process prompt blocks from form
    processPromptBlocks: function(button) {
        const promptBlocks = [];
        const promptBlockElements = document.getElementsByClassName('promptBlock');
        
        for (let i = 0; i < promptBlockElements.length; i++) {
            const nameField = document.getElementById(`promptName${i}`);
            const configField = document.getElementById(`promptConfig${i}`);
            
            if (nameField && configField) {
                try {
                    // Validate JSON but send as string
                    const configValue = configField.value.trim() || '{}';
                    // Parse to validate but don't use the parsed object
                    JSON.parse(configValue);
                    
                    promptBlocks.push({
                        name: nameField.value,
                        config: configValue // Send the raw string, not parsed JSON
                    });
                } catch (err) {
                    console.error(`Error parsing prompt block ${i} config:`, err);
                    showToast(`Error in prompt block ${i+1} configuration: Invalid JSON`, 'error');
                    
                    // If button is provided, restore its state
                    if (button) {
                        const originalButtonText = button.getAttribute('data-original-text');
                        button.innerHTML = originalButtonText;
                        button.disabled = false;
                    }
                    
                    return null; // Indicate validation error
                }
            }
        }
        
        return promptBlocks;
    },
    
    // Helper function to format config values (for edit form)
    formatConfigValue: function(configElement, configValue) {
        // If it's a string (already stringified JSON), try to parse it first
        if (typeof configValue === 'string') {
            try {
                const parsed = JSON.parse(configValue);
                configElement.value = JSON.stringify(parsed, null, 2);
            } catch (e) {
                console.warn("Failed to parse config JSON string:", e);
                configElement.value = configValue; // Keep as is if parsing fails
            }
        } else if (configValue !== undefined && configValue !== null) {
            // Direct object, just stringify with formatting
            configElement.value = JSON.stringify(configValue, null, 2);
        } else {
            // Default to empty object
            configElement.value = '{}';
        }
    },
    
    // Helper function to set select value (with fallback if option doesn't exist)
    setSelectValue: function(selectElement, value) {
        // Check if the option exists
        const optionExists = Array.from(selectElement.options).some(option => option.value === value);
        
        if (optionExists) {
            selectElement.value = value;
        } else if (value) {
            // If value is provided but option doesn't exist, create a new option
            const newOption = document.createElement('option');
            newOption.value = value;
            newOption.text = value + ' (custom)';
            selectElement.add(newOption);
            selectElement.value = value;
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
                    <select name="connectors[${index}].type" id="connectorType${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="connectorConfig${index}">Connector Config (JSON)</label>
                    <textarea id="connectorConfig${index}" name="connectors[${index}].config" placeholder='{"token":"sk-mg3.."}'>{}</textarea>
                </div>
            </div>
        `;
    },
    
    // MCP Server template
    mcpServerTemplate: function(index, data) {
        return `
            <div class="mcp_server mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>MCP Server ${index + 1}</h2>
                <div class="mb-4">
                    <label for="mcpURL${index}">MCP Server URL</label>
                    <input type="text" id="mcpURL${index}" name="mcp_servers[${index}].url" placeholder='https://...'>
                </div>
                <div class="mb-4">
                    <label for="mcpToken${index}">Bearer Token</label>
                    <input type="text" id="mcpToken${index}" name="mcp_servers[${index}].token" placeholder='Bearer token'>
                </div>
            </div>
        `;
    },
    
    // Action template
    actionTemplate: function(index, data) {
        return `
            <div class="action mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>Action ${index + 1}</h2>
                <div class="mb-4">
                    <label for="actionsName${index}">Action</label>
                    <select name="actions[${index}].name" id="actionsName${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="actionsConfig${index}">Action Config (JSON)</label>
                    <textarea id="actionsConfig${index}" name="actions[${index}].config" placeholder='{"results":"5"}'>{}</textarea>
                </div>
            </div>
        `;
    },
    
    // Prompt Block template
    promptBlockTemplate: function(index, data) {
        return `
            <div class="promptBlock mb-4 section-box" style="margin-top: 15px; padding: 15px;">
                <h2>Prompt Block ${index + 1}</h2>
                <div class="mb-4">
                    <label for="promptName${index}">Block Prompt</label>
                    <select name="promptblocks[${index}].name" id="promptName${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="promptConfig${index}">Prompt Config (JSON)</label>
                    <textarea id="promptConfig${index}" name="promptblocks[${index}].config" placeholder='{"results":"5"}'>{}</textarea>
                </div>
            </div>
        `;
    }
};

// Initialize form event listeners
function initAgentFormCommon(options = {}) {
    // Setup event listeners for dynamic component buttons
    if (options.enableConnectors !== false) {
        document.getElementById('addConnectorButton').addEventListener('click', function() {
            AgentFormUtils.addDynamicComponent('connectorsSection', AgentFormTemplates.connectorTemplate, {
                className: 'connector',
                options: options.connectors || ''
            });
        });
    }
    
    if (options.enableMCP !== false) {
        document.getElementById('addMCPButton').addEventListener('click', function() {
            AgentFormUtils.addDynamicComponent('mcpSection', AgentFormTemplates.mcpServerTemplate, {
                className: 'mcp_server'
            });
        });
    }
    
    if (options.enableActions !== false) {
        document.getElementById('action_button').addEventListener('click', function() {
            AgentFormUtils.addDynamicComponent('action_box', AgentFormTemplates.actionTemplate, {
                className: 'action',
                options: options.actions || ''
            });
        });
    }
    
    if (options.enablePromptBlocks !== false) {
        document.getElementById('dynamic_button').addEventListener('click', function() {
            AgentFormUtils.addDynamicComponent('dynamic_box', AgentFormTemplates.promptBlockTemplate, {
                className: 'promptBlock',
                options: options.promptBlocks || ''
            });
        });
    }
    
    // Add additional CSS for checkbox labels
    const style = document.createElement('style');
    style.textContent = `
        .checkbox-label {
            display: flex;
            align-items: center;
            cursor: pointer;
            margin-bottom: 10px;
        }
        
        .checkbox-label .checkbox-custom {
            margin-right: 10px;
        }
        
        @keyframes pulse {
            0% { transform: scale(1); }
            50% { transform: scale(1.05); }
            100% { transform: scale(1); }
        }
    `;
    document.head.appendChild(style);
}