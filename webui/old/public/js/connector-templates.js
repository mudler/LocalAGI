/**
 * Connector Templates
 * 
 * This file contains templates for all connector types supported by LocalAgent.
 * Each template is a function that returns an HTML string for the connector's form.
 * 
 * Note: We don't need to escape HTML in the value attributes because browsers
 * handle these values safely when setting them via DOM properties after rendering.
 */

/**
 * Connector Templates
 * Each function takes a config object and returns an HTML string
 */
const ConnectorTemplates = {
    /**
     * Telegram Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    telegram: function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="telegramToken${index}">Telegram Bot Token</label>
                <input type="text" 
                       id="telegramToken${index}" 
                       name="connectors[${index}][config][token]" 
                       class="form-control"
                       placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11">
                <small class="form-text text-muted">Get this from @BotFather on Telegram</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM to avoid XSS
        const setValues = function() {
            const input = document.getElementById(`telegramToken${index}`);
            if (input) input.value = config.token || '';
        };
        
        return { html, setValues };
    },

    /**
     * Slack Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    slack: function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="slackAppToken${index}">Slack App Token</label>
                <input type="text" 
                       id="slackAppToken${index}" 
                       name="connectors[${index}][config][appToken]" 
                       class="form-control"
                       placeholder="xapp-...">
                <small class="form-text text-muted">App-level token starting with xapp-</small>
            </div>
            
            <div class="form-group">
                <label for="slackBotToken${index}">Slack Bot Token</label>
                <input type="text" 
                       id="slackBotToken${index}" 
                       name="connectors[${index}][config][botToken]" 
                       class="form-control"
                       placeholder="xoxb-...">
                <small class="form-text text-muted">Bot token starting with xoxb-</small>
            </div>
            
            <div class="form-group">
                <label for="slackChannelID${index}">Slack Channel ID</label>
                <input type="text" 
                       id="slackChannelID${index}" 
                       name="connectors[${index}][config][channelID]" 
                       class="form-control"
                       placeholder="C012345678">
                <small class="form-text text-muted">Channel ID where the bot will operate</small>
            </div>
            
            <div class="form-group form-check">
                <input type="checkbox" 
                       class="form-check-input" 
                       id="slackAlwaysReply${index}" 
                       name="connectors[${index}][config][alwaysReply]">
                <label class="form-check-label" for="slackAlwaysReply${index}">Always Reply</label>
                <small class="form-text text-muted">If checked, the bot will reply to all messages in the channel</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM to avoid XSS
        const setValues = function() {
            const appTokenInput = document.getElementById(`slackAppToken${index}`);
            const botTokenInput = document.getElementById(`slackBotToken${index}`);
            const channelIDInput = document.getElementById(`slackChannelID${index}`);
            const alwaysReplyInput = document.getElementById(`slackAlwaysReply${index}`);
            
            if (appTokenInput) appTokenInput.value = config.appToken || '';
            if (botTokenInput) botTokenInput.value = config.botToken || '';
            if (channelIDInput) channelIDInput.value = config.channelID || '';
            if (alwaysReplyInput) alwaysReplyInput.checked = config.alwaysReply === 'true';
        };
        
        return { html, setValues };
    },

    /**
     * Discord Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    discord: function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="discordToken${index}">Discord Bot Token</label>
                <input type="text" 
                       id="discordToken${index}" 
                       name="connectors[${index}][config][token]" 
                       class="form-control"
                       placeholder="Bot token from Discord Developer Portal">
            </div>
            
            <div class="form-group">
                <label for="discordChannelID${index}">Default Channel ID</label>
                <input type="text" 
                       id="discordChannelID${index}" 
                       name="connectors[${index}][config][defaultChannel]" 
                       class="form-control"
                       placeholder="Channel ID where the bot will operate">
            </div>
        `;
        
        // Function to set values after HTML is added to DOM
        const setValues = function() {
            const tokenInput = document.getElementById(`discordToken${index}`);
            const channelIDInput = document.getElementById(`discordChannelID${index}`);
            
            if (tokenInput) tokenInput.value = config.token || '';
            if (channelIDInput) channelIDInput.value = config.defaultChannel || '';
        };
        
        return { html, setValues };
    },

    /**
     * GitHub Issues Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    'github-issues': function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="githubIssuesToken${index}">GitHub Personal Access Token</label>
                <input type="text" 
                       id="githubIssuesToken${index}" 
                       name="connectors[${index}][config][token]" 
                       class="form-control"
                       placeholder="ghp_...">
                <small class="form-text text-muted">Needs repo and read:org permissions</small>
            </div>
            
            <div class="form-group">
                <label for="githubIssuesOwner${index}">Repository Owner</label>
                <input type="text" 
                       id="githubIssuesOwner${index}" 
                       name="connectors[${index}][config][owner]" 
                       class="form-control"
                       placeholder="username or organization">
            </div>
            
            <div class="form-group">
                <label for="githubIssuesRepo${index}">Repository Name</label>
                <input type="text" 
                       id="githubIssuesRepo${index}" 
                       name="connectors[${index}][config][repository]" 
                       class="form-control"
                       placeholder="repository-name">
            </div>
            
            <div class="form-group form-check">
                <input type="checkbox" 
                       class="form-check-input" 
                       id="githubIssuesReplyIfNoReplies${index}" 
                       name="connectors[${index}][config][replyIfNoReplies]"
                       value="true">
                <label class="form-check-label" for="githubIssuesReplyIfNoReplies${index}">Reply to issues with no replies</label>
                <small class="form-text text-muted">If checked, the bot will reply to issues that have no replies yet</small>
            </div>
            
            <div class="form-group">
                <label for="githubIssuesPollInterval${index}">Poll Interval (seconds)</label>
                <input type="number" 
                       id="githubIssuesPollInterval${index}" 
                       name="connectors[${index}][config][pollInterval]" 
                       class="form-control"
                       placeholder="60">
                <small class="form-text text-muted">How often to check for new issues (in seconds)</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM to avoid XSS
        const setValues = function() {
            const tokenInput = document.getElementById(`githubIssuesToken${index}`);
            const ownerInput = document.getElementById(`githubIssuesOwner${index}`);
            const repoInput = document.getElementById(`githubIssuesRepo${index}`);
            const replyIfNoRepliesInput = document.getElementById(`githubIssuesReplyIfNoReplies${index}`);
            const pollIntervalInput = document.getElementById(`githubIssuesPollInterval${index}`);
            
            if (tokenInput) tokenInput.value = config.token || '';
            if (ownerInput) ownerInput.value = config.owner || '';
            if (repoInput) repoInput.value = config.repository || '';
            if (replyIfNoRepliesInput) replyIfNoRepliesInput.checked = config.replyIfNoReplies === 'true';
            if (pollIntervalInput) pollIntervalInput.value = config.pollInterval || '60';
        };
        
        return { html, setValues };
    },

    /**
     * GitHub PRs Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    'github-prs': function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="githubPRsToken${index}">GitHub Token</label>
                <input type="text" 
                       id="githubPRsToken${index}" 
                       name="connectors[${index}][config][token]" 
                       class="form-control"
                       placeholder="ghp_...">
                <small class="form-text text-muted">Personal Access Token with repo permissions</small>
            </div>
            
            <div class="form-group">
                <label for="githubPRsOwner${index}">Repository Owner</label>
                <input type="text" 
                       id="githubPRsOwner${index}" 
                       name="connectors[${index}][config][owner]" 
                       class="form-control"
                       placeholder="username or organization">
            </div>
            
            <div class="form-group">
                <label for="githubPRsRepo${index}">Repository Name</label>
                <input type="text" 
                       id="githubPRsRepo${index}" 
                       name="connectors[${index}][config][repository]" 
                       class="form-control"
                       placeholder="repository-name">
            </div>
            
            <div class="form-group form-check">
                <input type="checkbox" 
                       class="form-check-input" 
                       id="githubPRsReplyIfNoReplies${index}" 
                       name="connectors[${index}][config][replyIfNoReplies]"
                       value="true">
                <label class="form-check-label" for="githubPRsReplyIfNoReplies${index}">Reply to PRs with no replies</label>
                <small class="form-text text-muted">If checked, the bot will reply to pull requests that have no replies yet</small>
            </div>
            
            <div class="form-group">
                <label for="githubPRsPollInterval${index}">Poll Interval (seconds)</label>
                <input type="number" 
                       id="githubPRsPollInterval${index}" 
                       name="connectors[${index}][config][pollInterval]" 
                       class="form-control"
                       placeholder="60">
                <small class="form-text text-muted">How often to check for new pull requests (in seconds)</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM to avoid XSS
        const setValues = function() {
            const tokenInput = document.getElementById(`githubPRsToken${index}`);
            const ownerInput = document.getElementById(`githubPRsOwner${index}`);
            const repoInput = document.getElementById(`githubPRsRepo${index}`);
            const replyIfNoRepliesInput = document.getElementById(`githubPRsReplyIfNoReplies${index}`);
            const pollIntervalInput = document.getElementById(`githubPRsPollInterval${index}`);
            
            if (tokenInput) tokenInput.value = config.token || '';
            if (ownerInput) ownerInput.value = config.owner || '';
            if (repoInput) repoInput.value = config.repository || '';
            if (replyIfNoRepliesInput) replyIfNoRepliesInput.checked = config.replyIfNoReplies === 'true';
            if (pollIntervalInput) pollIntervalInput.value = config.pollInterval || '60';
        };
        
        return { html, setValues };
    },

    /**
     * IRC Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    irc: function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="ircServer${index}">IRC Server</label>
                <input type="text" 
                       id="ircServer${index}" 
                       name="connectors[${index}][config][server]" 
                       class="form-control"
                       placeholder="irc.libera.chat">
            </div>
            
            <div class="form-group">
                <label for="ircPort${index}">Port</label>
                <input type="text" 
                       id="ircPort${index}" 
                       name="connectors[${index}][config][port]" 
                       class="form-control"
                       placeholder="6667">
            </div>
            
            <div class="form-group">
                <label for="ircChannel${index}">Channel</label>
                <input type="text" 
                       id="ircChannel${index}" 
                       name="connectors[${index}][config][channel]" 
                       class="form-control"
                       placeholder="#channel">
            </div>
            
            <div class="form-group">
                <label for="ircNick${index}">Nickname</label>
                <input type="text" 
                       id="ircNick${index}" 
                       name="connectors[${index}][config][nickname]" 
                       class="form-control"
                       placeholder="MyBot">
            </div>

            <div class="form-group form-check">
                <input type="checkbox" 
                       class="form-check-input" 
                       id="ircAlwaysReply${index}" 
                       name="connectors[${index}][config][alwaysReply]"
                       value="true">
                <label class="form-check-label" for="ircAlwaysReply${index}">Always reply to messages</label>
                <small class="form-text text-muted">If checked, the bot will always reply to messages, even if they are not directed at it</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM
        const setValues = function() {
            const serverInput = document.getElementById(`ircServer${index}`);
            const portInput = document.getElementById(`ircPort${index}`);
            const channelInput = document.getElementById(`ircChannel${index}`);
            const nickInput = document.getElementById(`ircNick${index}`);
            const alwaysReplyInput = document.getElementById(`ircAlwaysReply${index}`);
            
            if (serverInput) serverInput.value = config.server || '';
            if (portInput) portInput.value = config.port || '6667';
            if (channelInput) channelInput.value = config.channel || '';
            if (nickInput) nickInput.value = config.nickname || '';
            if (alwaysReplyInput) alwaysReplyInput.checked = config.alwaysReply === 'true';
        };
        
        return { html, setValues };
    },

    /**
     * Twitter Connector Template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    twitter: function(config = {}, index) {
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="twitterToken${index}">Twitter API Token</label>
                <input type="text" 
                       id="twitterToken${index}" 
                       name="connectors[${index}][config][token]" 
                       class="form-control"
                       placeholder="Your Twitter API token">
            </div>
            
            <div class="form-group">
                <label for="twitterBotUsername${index}">Bot Username</label>
                <input type="text" 
                       id="twitterBotUsername${index}" 
                       name="connectors[${index}][config][botUsername]" 
                       class="form-control"
                       placeholder="@YourBotUsername">
                <small class="form-text text-muted">Username of your Twitter bot (with or without @)</small>
            </div>
            
            <div class="form-group form-check">
                <input type="checkbox" 
                       class="form-check-input" 
                       id="twitterNoCharLimit${index}" 
                       name="connectors[${index}][config][noCharacterLimit]"
                       value="true">
                <label class="form-check-label" for="twitterNoCharLimit${index}">Disable character limit</label>
                <small class="form-text text-muted">If checked, the bot will not enforce Twitter's character limit</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM
        const setValues = function() {
            const tokenInput = document.getElementById(`twitterToken${index}`);
            const botUsernameInput = document.getElementById(`twitterBotUsername${index}`);
            const noCharLimitInput = document.getElementById(`twitterNoCharLimit${index}`);
            
            if (tokenInput) tokenInput.value = config.token || '';
            if (botUsernameInput) botUsernameInput.value = config.botUsername || '';
            if (noCharLimitInput) noCharLimitInput.checked = config.noCharacterLimit === 'true';
        };
        
        return { html, setValues };
    },
    
    /**
     * Fallback template for any connector without a specific template
     * @param {Object} config - Existing configuration values
     * @param {Number} index - Connector index
     * @returns {Object} HTML template and setValues function
     */
    fallback: function(config = {}, index) {
        // Convert config to a pretty-printed JSON string
        let configStr = '{}';
        try {
            if (typeof config === 'string') {
                // If it's already a string, try to parse it first to pretty-print
                configStr = JSON.stringify(JSON.parse(config), null, 2);
            } else if (typeof config === 'object' && config !== null) {
                configStr = JSON.stringify(config, null, 2);
            }
        } catch (e) {
            console.error('Error formatting config:', e);
            // If it's a string but not valid JSON, just use it as is
            if (typeof config === 'string') {
                configStr = config;
            }
        }
        
        // Return HTML without values in the template string
        const html = `
            <div class="form-group">
                <label for="connectorConfig${index}">Connector Configuration (JSON)</label>
                <textarea id="connectorConfig${index}" 
                          name="connectors[${index}][config]" 
                          class="form-control" 
                          rows="10"
                          placeholder='{"key":"value"}'>${escapeHTML(configStr)}</textarea>
                <small class="form-text text-muted">Enter the connector configuration as a JSON object</small>
            </div>
        `;
        
        // Function to set values after HTML is added to DOM
        const setValues = function() {
            const configInput = document.getElementById(`connectorConfig${index}`);
            
            if (configInput) configInput.value = configStr;
        };
        
        return { html, setValues };
    }
};
