// Common utility functions for agent forms
const AgentFormUtils = {
    // Add dynamic component based on template
    addDynamicComponent: function(sectionId, templateFunction, options = {}) {
        const section = document.getElementById(sectionId);
        if (!section) return;
        
        const index = section.getElementsByClassName(options.className || 'dynamic-component').length;
        const templateData = { index, ...options };
        
        // Create a new element from the template
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = templateFunction(index, templateData);
        const newElement = tempDiv.firstElementChild;
        
        // Add the new element to the section
        section.appendChild(newElement);
        
        // If it's a connector, add event listener for type change
        if (options.className === 'connector') {
            const newIndex = index;
            const connectorType = document.getElementById(`connectorType${newIndex}`);
            if (connectorType) {
                // Add event listener for future changes
                connectorType.addEventListener('change', function() {
                    loadConnectorForm(newIndex, this.value, null);
                });
                
                // If a connector type is already selected (default value), load its form immediately
                if (connectorType.value) {
                    loadConnectorForm(newIndex, connectorType.value, null);
                }
            }
        }
        
        return newElement;
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
            const typeSelect = document.getElementById(`connectorType${i}`);
            
            if (typeSelect) {
                const connectorType = typeSelect.value;
                const configContainer = document.getElementById(`connectorConfigContainer${i}`);
                
                // Only process if we have a metaform
                if (configContainer && configContainer.querySelector('.metaform')) {
                    try {
                        // Get all form fields
                        const fields = configContainer.querySelectorAll('.connector-field');
                        let configObj = {};
                        
                        // Process each field based on its type
                        fields.forEach(field => {
                            const fieldName = field.dataset.fieldName;
                            const fieldType = field.dataset.fieldType;
                            
                            // Convert value based on field type
                            let value = field.value;
                            if (fieldType === 'number' && value !== '') {
                                value = parseFloat(value);
                            }
                            
                            configObj[fieldName] = value;
                        });
                        
                        // Add the connector to the list
                        connectors.push({
                            type: connectorType,
                            config: JSON.stringify(configObj)
                        });
                    } catch (err) {
                        console.error(`Error processing connector ${i} form:`, err);
                        showToast(`Error in connector ${i+1} configuration`, 'error');
                        
                        // If button is provided, restore its state
                        if (button) {
                            const originalButtonText = button.getAttribute('data-original-text');
                            button.innerHTML = originalButtonText;
                            button.disabled = false;
                        }
                        
                        return null; // Indicate validation error
                    }
                } else {
                    // If no form is loaded, create an empty config
                    connectors.push({
                        type: connectorType,
                        config: '{}'
                    });
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
                // If parsing fails, use the raw string
                configElement.value = configValue;
            }
        } 
        // If it's already an object, stringify it
        else if (typeof configValue === 'object' && configValue !== null) {
            configElement.value = JSON.stringify(configValue, null, 2);
        }
        // Default to empty object
        else {
            configElement.value = '{}';
        }
    },
    
    // Helper function to set select value (with fallback if option doesn't exist)
    setSelectValue: function(selectElement, value) {
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
            // Otherwise select the first option
            selectElement.selectedIndex = 0;
        }
    }
};

// Function to load connector form based on type
function loadConnectorForm(index, connectorType, configData) {
    if (!connectorType) return;
    
    const configContainer = document.getElementById(`connectorConfigContainer${index}`);
    if (!configContainer) return;
    
    // Show loading indicator
    configContainer.innerHTML = '<div class="loading-spinner">Loading form...</div>';
    
    // Fetch the form for the selected connector type
    fetch(`/settings/connector/form/${connectorType}`)
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to load connector form');
            }
            return response.text();
        })
        .then(html => {
            // Replace the container content with the form
            configContainer.innerHTML = html;
            
            // Store the connector type as a data attribute on the form
            const metaform = configContainer.querySelector('.metaform');
            if (metaform) {
                metaform.setAttribute('data-connector-type', connectorType);
                
                // Add a hidden input to store the connector type
                const hiddenInput = document.createElement('input');
                hiddenInput.type = 'hidden';
                hiddenInput.name = 'connector-type';
                hiddenInput.value = connectorType;
                metaform.appendChild(hiddenInput);
                
                // If we have config data, populate the form fields
                if (configData) {
                    try {
                        // Parse the config JSON
                        const parsedConfig = JSON.parse(configData);
                        
                        // Find all form fields
                        const fields = metaform.querySelectorAll('.connector-field');
                        
                        // Populate each field with the corresponding value from the config
                        fields.forEach(field => {
                            const fieldName = field.dataset.fieldName;
                            if (parsedConfig[fieldName] !== undefined) {
                                field.value = parsedConfig[fieldName];
                            }
                        });
                    } catch (error) {
                        console.warn(`Failed to populate connector form for ${connectorType}:`, error);
                    }
                }
            }
        })
        .catch(error => {
            console.error('Error loading connector form:', error);
            configContainer.innerHTML = `
                <div class="error-message">
                    <p>Failed to load connector form: ${error.message}</p>
                </div>
            `;
        });
}

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
                <div id="connectorConfigContainer${index}" class="mb-4">
                    <div class="text-center py-4">
                        <p>Select a connector type to load its configuration form.</p>
                    </div>
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
                    <label for="mcpURL${index}">Server URL</label>
                    <input type="text" id="mcpURL${index}" name="mcp_servers[${index}].url" placeholder="https://example.com">
                </div>
                <div class="mb-4">
                    <label for="mcpToken${index}">API Token (Optional)</label>
                    <input type="text" id="mcpToken${index}" name="mcp_servers[${index}].token" placeholder="API token">
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
                    <label for="actionsName${index}">Action Type</label>
                    <select name="actions[${index}].name" id="actionsName${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="actionsConfig${index}">Action Config (JSON)</label>
                    <textarea id="actionsConfig${index}" name="actions[${index}].config" placeholder='{"param":"value"}'>{}</textarea>
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
                    <label for="promptName${index}">Prompt Block Type</label>
                    <select name="promptblocks[${index}].name" id="promptName${index}">
                        ${data.options}
                    </select>
                </div>
                <div class="mb-4">
                    <label for="promptConfig${index}">Prompt Block Config (JSON)</label>
                    <textarea id="promptConfig${index}" name="promptblocks[${index}].config" placeholder='{"param":"value"}'>{}</textarea>
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
    
    // Add additional CSS for loading spinner and error messages
    const style = document.createElement('style');
    style.textContent = `
        .loading-spinner {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100px;
            color: #f0f0f0;
        }
        
        .loading-spinner::after {
            content: '';
            width: 20px;
            height: 20px;
            border: 2px solid #f0f0f0;
            border-top-color: transparent;
            border-radius: 50%;
            animation: spinner 1s linear infinite;
            margin-left: 10px;
        }
        
        @keyframes spinner {
            to { transform: rotate(360deg); }
        }
        
        .error-message {
            color: #ff5555;
            padding: 10px;
            border: 1px solid #ff5555;
            border-radius: 4px;
            margin-bottom: 10px;
        }
    `;
    document.head.appendChild(style);
}