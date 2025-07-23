<p align="center">
  <img src="./webui/react-ui/public/logo_1.png" alt="LocalAGI Logo" width="220"/>
</p>

<h3 align="center"><em>Your AI. Your Hardware. Your Rules</em></h3>

<div align="center">
  
[![Go Report Card](https://goreportcard.com/badge/github.com/bit-gpt/local-agi)](https://goreportcard.com/report/github.com/bit-gpt/local-agi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub stars](https://img.shields.io/github/stars/bit-gpt/local-agi)](https://github.com/bit-gpt/local-agi/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/bit-gpt/local-agi)](https://github.com/bit-gpt/local-agi/issues)


</div>

**LocalAGI** is a powerful, self-hostable AI Agent platform that allows you to design AI automations without writing code. A complete drop-in replacement for OpenAI's Responses APIs with advanced agentic capabilities.

## 🌟 Key Features

- 🎛 **No-Code Agents**: Easy-to-configure multiple agents via Web UI.
- 🖥 **Web-Based Interface**: Simple and intuitive agent management.
- 🤖 **Advanced Agent Teaming**: Instantly create cooperative agent teams from a single prompt.
- 📡 **Connectors Galore**: Built-in integrations with Discord, Slack, Telegram, GitHub Issues, and IRC.
- 🛠 **Comprehensive REST API**: Seamless integration into your workflows. Every agent created will support OpenAI Responses API out of the box.
- 🧠 **Planning & Reasoning**: Agents intelligently plan, reason, and adapt.
- 🔄 **Periodic Tasks**: Schedule tasks with cron-like syntax.
- 💾 **Memory Management**: Control memory usage with options for long-term.
- 🖼 **Multimodal Support**: Ready for vision, text, and more.
- 🔧 **Extensible Custom Actions**: Easily script dynamic agent behaviors in Go (interpreted, no compilation!).
- 📊 **Observability**: Monitor agent status and view detailed observable updates in real-time.

## 🛠️ Quickstart

```bash
# Clone the repository
git clone https://github.com/bit-gpt/local-agi
cd local-agi

# Build the project
./build.sh

# Run the Go backend (from the project root)
./app

# Access the application
# Open your browser and go to the address where the backend is running (e.g., http://localhost:3000)
# If you are unsure of the port, check your Go code (main.go or webui/app.go) for the port configuration.

```

Now you can access and manage your agents at [http://localhost:3000](http://localhost:3000)

## 🏆 Why Choose LocalAGI?

- **✓ Developer-Friendly**: Rich APIs and intuitive interfaces.
- **✓ Effortless Setup**: Simple setup and pre-built binaries.
- **✓ Feature-Rich**: From planning to multimodal capabilities, connectors for Slack, MCP support, LocalAGI has it all.

## 🌟 Screenshots

### Powerful Web UI

<table>
  <tr>
    <td>
      <img src="https://github.com/user-attachments/assets/a7d8e22f-336f-404e-9fa4-134ddba43645" alt="Web UI Dashboard" width="400"/>
    </td>
    <td>
      <img src="https://github.com/user-attachments/assets/93e69f82-30e7-437a-858c-d9055c124719" alt="Web UI Agent Settings" width="400"/>
    </td>
  </tr>
  <tr>
    <td>
      <img src="https://github.com/user-attachments/assets/005514a6-c92d-44d4-bd02-012547b3fedf" alt="Web UI Create Group" width="400"/>
    </td>
    <td>
      <img src="https://github.com/user-attachments/assets/a9ed3c9b-f9e4-4ccd-aba3-18ac766559d7" alt="Web UI Agent Status" width="400"/>
    </td>
  </tr>
  <tr>
    <td>
      <img src="https://github.com/user-attachments/assets/12b05000-5e03-4bc0-a70f-0f66099f5376" alt="Web UI Agent Chat" width="400"/>
    </td>
    <td>
      <img src="https://github.com/user-attachments/assets/45e39d03-d972-45e1-8d04-fdb00a68cb4c" alt="Web UI Agent Observability" width="400"/>
    </td>
  </tr>
</table>


### Connectors Ready-to-Go

<p align="center">
  <img src="https://github.com/user-attachments/assets/014dce65-7b93-490c-98e5-732cec92dda6" alt="Telegram" width="100"/>
  <img src="https://github.com/user-attachments/assets/d055abd2-61c8-447d-b0dd-69c181bdd705" alt="Discord" width="100"/>
  <img src="https://github.com/user-attachments/assets/4a9172c0-2d9a-446f-affd-b21dedf0b073" alt="Slack" width="100"/>
  <img src="https://github.com/user-attachments/assets/e7c13dff-8e27-48cb-901c-0c11fd6d5f05" alt="IRC" width="50"/>
  <img src="https://github.com/user-attachments/assets/bc0f3dd3-9595-4099-88cf-f0265add9986" alt="GitHub" width="100"/>
</p>

## 📖 Full Documentation

Explore detailed documentation including:
- [Installation Options](#installation-options)
- [REST API Documentation](#rest-api)
- [Connector Configuration](#connectors)
- [Agent Configuration](#agent-configuration-reference)

### Environment Configuration

LocalAGI supports environment configurations.

| Variable | What It Does |
|----------|--------------|
| `DB_HOST` | MySQL Database host address |
| `DB_NAME` | MySQL Database name |
| `DB_PASS` | MySQL Database password |
| `DB_USER` | MySQL Database user |
| `LOCALAGI_LLM_API_URL` | OpenAI-compatible API server URL (e.g., for OpenRouter) |
| `LOCALAGI_LLM_API_KEY` | API authentication key for LLM API |
| `LOCALAGI_TIMEOUT` | Request timeout settings (e.g., 5m) |
| `VITE_PRIVY_APP_ID` | Privy App ID for frontend (Vite) |
| `PRIVY_APP_ID` | Privy App ID for backend |
| `PRIVY_APP_SECRET` | Privy App Secret for backend authentication |
| `PRIVY_PUBLIC_KEY_PEM` | Privy public key PEM (if required) |

## Installation Options

### Pre-Built Binaries

Download ready-to-run binaries from the [Releases](https://github.com/mudler/LocalAGI/releases) page.

### Source Build

Requirements:
- Go 1.20+
- Git
- Bun 1.2+

```bash
# Clone the repository
git clone https://github.com/bit-gpt/local-agi
cd local-agi

# Build the project
./build.sh

# Run the Go backend (from the project root)
./app
```

## 🔧 Extending LocalAGI

LocalAGI provides a powerful way to extend its functionality with custom actions:


###  MCP (Model Context Protocol) Servers

LocalAGI supports both smithery.ai and glama.ai MCP servers, allowing you to extend functionality with external tools and services.

#### What is MCP?

The Model Context Protocol (MCP) is a standard for connecting AI applications to external data sources and tools. LocalAGI can connect to any MCP-compliant server to access additional capabilities.

#### Configuring MCP Servers in LocalAGI

1. **Via Web UI**: In the MCP Settings section of agent creation, add MCP servers

#### Best Practices

- **Security**: Always validate inputs and use proper authentication for remote MCP servers
- **Error Handling**: Implement robust error handling in your MCP servers
- **Documentation**: Provide clear descriptions for all tools exposed by your MCP server
- **Testing**: Test your MCP servers independently before integrating with LocalAGI
- **Resource Management**: Ensure your MCP servers properly clean up resources

### Development

The development workflow is similar to the source build, but with additional steps for hot reloading of the frontend:

```bash
# Clone repo
git clone https://github.com/bit-gpt/local-agi.git
cd local-agi

# Install dependencies and start frontend development server
cd webui/react-ui && bun i && bun run dev
```

Then in separate terminal:

```bash
# Start development server
cd ../.. && go run main.go
```

> Note: see webui/react-ui/.vite.config.js for env vars that can be used to configure the backend URL

## CONNECTORS

Link your agents to the services you already use. Configuration examples below.

<details>
<summary><strong>GitHub Issues</strong></summary>

```json
{
  "token": "YOUR_PAT_TOKEN",
  "repository": "repo-to-monitor",
  "owner": "repo-owner",
  "botUserName": "bot-username"
}
```
</details>

<details>
<summary><strong>Discord</strong></summary>

After [creating your Discord bot](https://discordpy.readthedocs.io/en/stable/discord.html):

```json
{
  "token": "Bot YOUR_DISCORD_TOKEN",
  "defaultChannel": "OPTIONAL_CHANNEL_ID"
}
```
> Don't forget to enable "Message Content Intent" in Bot(tab) settings!
> Enable " Message Content Intent " in the Bot tab!
</details>

<details>
<summary><strong>Slack</strong></summary>

Use the included `slack.yaml` manifest to create your app, then configure:

```json
{
  "botToken": "xoxb-your-bot-token",
  "appToken": "xapp-your-app-token"
}
```

- Create Oauth token bot token from "OAuth & Permissions" -> "OAuth Tokens for Your Workspace"
- Create App level token (from "Basic Information" -> "App-Level Tokens" ( scope connections:writeRoute authorizations:read ))
</details>

<details>
<summary><strong>Telegram</strong></summary>

Get a token from @botfather, then:

```json
{ 
  "token": "your-bot-father-token",
  "group_mode": "true",
  "mention_only": "true",
  "admins": "username1,username2"
}
```

Configuration options:
- `token`: Your bot token from BotFather
- `group_mode`: Enable/disable group chat functionality
- `mention_only`: When enabled, bot only responds when mentioned in groups
- `admins`: Comma-separated list of Telegram usernames allowed to use the bot in private chats
- `channel_id`: Optional channel ID for the bot to send messages to

> **Important**: For group functionality to work properly:
> 1. Go to @BotFather
> 2. Select your bot
> 3. Go to "Bot Settings" > "Group Privacy"
> 4. Select "Turn off" to allow the bot to read all messages in groups
> 5. Restart your bot after changing this setting
</details>

<details>
<summary><strong>IRC</strong></summary>

Connect to IRC networks:

```json
{
  "server": "irc.example.com",
  "port": "6667",
  "nickname": "LocalAGIBot",
  "channel": "#yourchannel",
  "alwaysReply": "false"
}
```
</details>

<details>
<summary><strong>Email</strong></summary>

```json
{
  "smtpServer": "smtp.gmail.com:587",
  "imapServer": "imap.gmail.com:993",
  "smtpInsecure": "false",
  "imapInsecure": "false",
  "username": "user@gmail.com",
  "email": "user@gmail.com",
  "password": "correct-horse-battery-staple",
  "name": "LogalAGI Agent"
}
```
</details>

## REST API

<details>
<summary><strong>Agent Management</strong></summary>

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/agents` | GET | List all available agents |
| `/api/agent/:id` | GET | Get agent details |
| `/api/agent/:id/status` | GET | View agent status history |
| `/api/agent/create` | POST | Create a new agent |
| `/api/agent/:id` | DELETE | Remove an agent |
| `/api/agent/:id/pause` | PUT | Pause agent activities |
| `/api/agent/:id/start` | PUT | Resume a paused agent |
| `/api/agent/:id/config` | GET | Get agent configuration |
| `/api/agent/:id/config` | PUT | Update agent configuration |
| `/api/agent/config/metadata` | GET | Get agent configuration metadata |
| `/api/meta/agent/config` | GET | Get agent configuration metadata |
| `/settings/export/:id` | GET | Export agent config |
| `/settings/import` | POST | Import agent config |
</details>

<details>
<summary><strong>Actions and Groups</strong></summary>

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/actions` | GET | List available actions |
| `/api/action/:name/run` | POST | Execute an action |
| `/api/action/:name/definition` | POST | Get action definition |
| `/api/agent/group/generateProfiles` | POST | Generate group profiles |
| `/api/agent/group/create` | POST | Create a new agent group |
</details>

<details>
<summary><strong>Chat and Communication</strong></summary>

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/chat/:id` | POST | Send message & get response |
| `/api/chat/:id` | GET | Get chat history |
| `/api/chat/:id` | DELETE | Clear chat history |
| `/api/sse/:id` | GET | Real-time agent event stream |
| `/api/agent/:id/observables` | GET | Get agent observables |
</details>

<details>
<summary><strong>Usage and Analytics</strong></summary>

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/usage` | GET | Get usage statistics |
</details>

<details>
<summary><strong>Curl Examples</strong></summary>

> **Note**: When using the API with curl, you need to include your Privy authentication token. You can either:
> - Include it as a cookie: `-b "privy-token=YOUR_TOKEN_HERE"`
> - Or set it as a header: `-H "Cookie: privy-token=YOUR_TOKEN_HERE"`
> 
> Replace `YOUR_TOKEN_HERE` with your actual Privy JWT token obtained from the web interface.

#### Get All Agents
```bash
curl -X GET "http://localhost:3000/api/agents"
```

#### Get Agent Details
```bash
curl -X GET "http://localhost:3000/api/agent/agent-id"
```

#### Get Agent Status
```bash
curl -X GET "http://localhost:3000/api/agent/agent-id/status"
```

#### Create Agent
```bash
curl -X POST "http://localhost:3000/api/agent/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-agent",
    "model": "gpt-4",
    "system_prompt": "You are an AI assistant.",
    "enable_kb": true,
    "enable_reasoning": true
  }'
```

#### Delete Agent
```bash
curl -X DELETE "http://localhost:3000/api/agent/agent-id"
```

#### Pause Agent
```bash
curl -X PUT "http://localhost:3000/api/agent/agent-id/pause"
```

#### Start Agent
```bash
curl -X PUT "http://localhost:3000/api/agent/agent-id/start"
```

#### Get Agent Configuration
```bash
curl -X GET "http://localhost:3000/api/agent/agent-id/config"
```

#### Update Agent Configuration
```bash
curl -X PUT "http://localhost:3000/api/agent/agent-id/config" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "system_prompt": "You are an AI assistant."
  }'
```

#### Export Agent
```bash
curl -X GET "http://localhost:3000/settings/export/agent-id" --output my-agent.json
```

#### Import Agent
```bash
curl -X POST "http://localhost:3000/settings/import" \
  -F "file=@/path/to/my-agent.json"
```

#### Send Message
```bash
curl -X POST "http://localhost:3000/api/chat/agent-id" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, how are you today?"}'
```

#### Get Chat History
```bash
curl -X GET "http://localhost:3000/api/agent/agent-id/chat"
```

#### Clear Chat History
```bash
curl -X DELETE "http://localhost:3000/api/agent/agent-id/chat"
```

#### Agent SSE Stream
```bash
curl -N -X GET "http://localhost:3000/api/sse/agent-id"
```
Note: For proper SSE handling, you should use a client that supports SSE natively.

#### Get Usage Statistics
```bash
curl -X GET "http://localhost:3000/api/usage"
```

#### Execute Action
```bash
curl -X POST "http://localhost:3000/api/action/action-name/run" \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "param1": "value1",
      "param2": "value2"
    }
  }'
```

#### Get Action Definition
```bash
curl -X POST "http://localhost:3000/api/action/action-name/definition"
```

#### Generate Group Profiles
```bash
curl -X POST "http://localhost:3000/api/agent/group/generateProfiles" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "A team of agents to help with project management"
  }'
```

#### Create Agent Group
```bash
curl -X POST "http://localhost:3000/api/agent/group/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "project-team",
    "agents": [
      {
        "name": "coordinator",
        "role": "Project Coordinator"
      },
      {
        "name": "developer",
        "role": "Developer"
      }
    ]
  }'
```
</details>

### Agent Configuration Reference

<details>
<summary><strong>Configuration Structure</strong></summary>

The agent configuration defines how an agent behaves and what capabilities it has. You can view the available configuration options and their descriptions by using the metadata endpoint:

```bash
curl -X GET "http://localhost:3000/api/meta/agent/config"
```

This will return a JSON object containing all available configuration fields, their types, and descriptions.

Here's an example of the agent configuration structure:

```json
{
  "name": "my-agent",
  "model": "gpt-4",
  "multimodal_model": "gpt-4-vision",
  "hud": true,
  "standalone_job": false,
  "random_identity": false,
  "initiate_conversations": true,
  "enable_planning": true,
  "identity_guidance": "You are a helpful assistant.",
  "periodic_runs": "0 * * * *",
  "permanent_goal": "Help users with their questions.",
  "enable_kb": true,
  "enable_reasoning": true,
  "kb_results": 5,
  "can_stop_itself": false,
  "system_prompt": "You are an AI assistant.",
  "long_term_memory": true,
  "summary_long_term_memory": false
}
```
</details>

<details>
<summary><strong>Environment Configuration</strong></summary>

LocalAGI supports environment configurations. Note that these environment variables needs to be specified in the localagi container in the docker-compose file to have effect.

| Variable | What It Does |
|----------|--------------|
| `DB_HOST` | MySQL Database host address |
| `DB_NAME` | MySQL Database name |
| `DB_PASS` | MySQL Database password |
| `DB_USER` | MySQL Database user |
| `LOCALAGI_LLM_API_URL` | OpenAI-compatible API server URL (e.g., for OpenRouter) |
| `LOCALAGI_LLM_API_KEY` | API authentication key for LLM API |
| `LOCALAGI_MODEL` | Model name to use (e.g., deepseek/deepseek-chat-v3-0324:free) |
| `LOCALAGI_TIMEOUT` | Request timeout settings (e.g., 5m) |
| `VITE_PRIVY_APP_ID` | Privy App ID for frontend (Vite) |
| `PRIVY_APP_ID` | Privy App ID for backend |
| `PRIVY_APP_SECRET` | Privy App Secret for backend authentication |
| `PRIVY_PUBLIC_KEY_PEM` | Privy public key PEM (if required) |
</details>

## LICENSE

MIT License — See the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>LOCAL PROCESSING. GLOBAL THINKING.</strong><br>
  Made with ❤️ by <a href="https://github.com/bit-gpt">BitGPT</a>
</p>
