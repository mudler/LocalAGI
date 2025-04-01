<p align="center">
  <img src="https://github.com/user-attachments/assets/304ad402-5ddc-441b-a4b9-55ff9eec72be" alt="LocalAgent Logo" width="220"/>
</p>

<h1 align="center">LOCAL AGENT</h1>
<h3 align="center"><em>AI that stays where it belongs — on your machine.</em></h3>

<div align="center">
  
[![Go Report Card](https://goreportcard.com/badge/github.com/mudler/LocalAgent)](https://goreportcard.com/report/github.com/mudler/LocalAgent)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub stars](https://img.shields.io/github/stars/mudler/LocalAgent)](https://github.com/mudler/LocalAgent/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/mudler/LocalAgent)](https://github.com/mudler/LocalAgent/issues)
  
</div>

## TAKE BACK CONTROL

**LocalAgent** is an AI platform that runs 100% on your hardware. No clouds. No data sharing. No compromises.

Built for those who value privacy as non-negotiable, LocalAgent lets you deploy intelligent agents that never phone home. Your data stays where you put it — period. Are you tired of agent wrappers to cloud APIs? me too.

## WHY LOCALAGENT?

- **✓ TRUE PRIVACY** — Everything runs on your hardware, nothing leaves your machine
- **✓ MODEL FREEDOM** — Works with local LLM formats (GGUF, GGML) you already have
- **✓ BUILD YOUR WAY** — Extensible architecture for custom agents with specialized skills
- **✓ SLICK INTERFACE** — Clean web UI for hassle-free agent interactions
- **✓ DEV-FRIENDLY** — Comprehensive REST API for seamless integration
- **✓ PLAYS WELL WITH OTHERS** — Optimized for [LocalAI](https://github.com/mudler/LocalAI)
- **✓ RUN ANYWHERE** — Linux, macOS, Windows — we've got you covered

## THE LOCAL ECOSYSTEM

LocalAgent is part of a trinity of tools designed to keep AI under your control:

- [**LocalAI**](https://github.com/mudler/LocalAI) — Run LLMs on your hardware
- [**LocalRAG**](https://github.com/mudler/LocalRAG) — Local Retrieval-Augmented Generation
- [**LocalAgent**](https://github.com/mudler/LocalAgent) — Deploy AI agents that respect your privacy

## Features

### Powerful WebUI

![Screenshot from 2025-03-11 22-50-24](https://github.com/user-attachments/assets/cd5228a3-4e67-4271-8fce-eccd229e6e58)
![Screenshot from 2025-03-11 22-50-06](https://github.com/user-attachments/assets/0a5ddb03-85ff-4995-8217-785d3249ffb1)
![Screenshot from 2025-03-11 22-49-56](https://github.com/user-attachments/assets/65af8ee6-ed2b-4e60-8906-ea12b28ecc58)


### Connectors ready-to-go

<p align="center">
  <img src="https://github.com/user-attachments/assets/4171072f-e4bf-4485-982b-55d55086f8fc" alt="Telegram Logo" width="60"/>
  <img src="https://github.com/user-attachments/assets/9235da84-0187-4f26-8482-32dcc55702ef" alt="Discord Logo" width="220"/>
  <img src="https://github.com/user-attachments/assets/a88c3d88-a387-4fb5-b513-22bdd5da7413" alt="Slack Logo" width="220"/>
  <img src="https://github.com/user-attachments/assets/d249cdf5-ab34-4ab1-afdf-b99e2db182d2" alt="IRC Logo" width="220"/>
  <img src="https://github.com/user-attachments/assets/52c852b0-4b50-4926-9fa0-aa50613ac622" alt="Github Logo" width="220"/>
</p>

## QUICK START

### One-Command Docker Setup

The fastest way to get everything running — LocalRAG, LocalAI, and LocalAgent pre-configured:

```bash
docker-compose up
```

> No API keys. No cloud subscriptions. No external dependencies. Just AI that works.

### Manual Launch

Run the binary and you're live:

```bash
./localagent
```

Access your agents at `http://localhost:3000`

### Environment Configuration

| Variable | What It Does |
|----------|--------------|
| `LOCALAGENT_MODEL` | Your go-to model |
| `LOCALAGENT_MULTIMODAL_MODEL` | Optional model for multimodal capabilities |
| `LOCALAGENT_LLM_API_URL` | OpenAI-compatible API server URL |
| `LOCALAGENT_LLM_API_KEY` | API authentication |
| `LOCALAGENT_TIMEOUT` | Request timeout settings |
| `LOCALAGENT_STATE_DIR` | Where state gets stored |
| `LOCALAGENT_LOCALRAG_URL` | LocalRAG connection |
| `LOCALAGENT_ENABLE_CONVERSATIONS_LOGGING` | Toggle conversation logs |
| `LOCALAGENT_API_KEYS` | A comma separated list of api keys used for authentication |

## INSTALLATION OPTIONS

### Pre-Built Binaries

Download ready-to-run binaries from the [Releases](https://github.com/mudler/LocalAgent/releases) page.

### Source Build

Requirements:
- Go 1.20+
- Git
- Bun 1.2+

```bash
# Clone repo
git clone https://github.com/mudler/LocalAgent.git
cd LocalAgent

# Build it
cd webui/react-ui && bun i && bun run build
cd ../..
go build -o localagent

# Run it
./localagent
```

### Development

The development workflow is similar to the source build, but with additional steps for hot reloading of the frontend:

```bash
# Clone repo
git clone https://github.com/mudler/LocalAgent.git
cd LocalAgent

# Install dependencies and start frontend development server
cd webui/react-ui && bun i && bun run dev
```

Then in seperate terminal:

```bash
# Start development server
cd ../.. && go run main.go
```

> Note: see webui/react-ui/.vite.config.js for env vars that can be used to configure the backend URL

## CONNECTORS

Link your agents to the services you already use. Configuration examples below.

### GitHub Issues

```json
{
  "token": "YOUR_PAT_TOKEN",
  "repository": "repo-to-monitor",
  "owner": "repo-owner",
  "botUserName": "bot-username"
}
```

### Discord

After [creating your Discord bot](https://discordpy.readthedocs.io/en/stable/discord.html):

```json
{
  "token": "Bot YOUR_DISCORD_TOKEN",
  "defaultChannel": "OPTIONAL_CHANNEL_ID"
}
```
> Don't forget to enable "Message Content Intent" in Bot(tab) settings!
> Enable " Message Content Intent " in the Bot tab!

### Slack

Use the included `slack.yaml` manifest to create your app, then configure:

```json
{
  "botToken": "xoxb-your-bot-token",
  "appToken": "xapp-your-app-token"
}
```

- Create Oauth token bot token from "OAuth & Permissions" -> "OAuth Tokens for Your Workspace"
- Create App level token (from "Basic Information" -> "App-Level Tokens" ( scope connections:writeRoute authorizations:read ))


### Telegram

Get a token from @botfather, then:

```json
{ 
  "token": "your-bot-father-token" 
}
```

### IRC

Connect to IRC networks:

```json
{
  "server": "irc.example.com",
  "port": "6667",
  "nickname": "LocalAgentBot",
  "channel": "#yourchannel",
  "alwaysReply": "false"
}
```

## REST API

### Agent Management

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/api/agents` | GET | List all available agents | [Example](#get-all-agents) |
| `/api/agent/:name/status` | GET | View agent status history | [Example](#get-agent-status) |
| `/api/agent/create` | POST | Create a new agent | [Example](#create-agent) |
| `/api/agent/:name` | DELETE | Remove an agent | [Example](#delete-agent) |
| `/api/agent/:name/pause` | PUT | Pause agent activities | [Example](#pause-agent) |
| `/api/agent/:name/start` | PUT | Resume a paused agent | [Example](#start-agent) |
| `/api/agent/:name/config` | GET | Get agent configuration | |
| `/api/agent/:name/config` | PUT | Update agent configuration | |
| `/api/meta/agent/config` | GET | Get agent configuration metadata | |
| `/settings/export/:name` | GET | Export agent config | [Example](#export-agent) |
| `/settings/import` | POST | Import agent config | [Example](#import-agent) |

### Actions and Groups

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/api/actions` | GET | List available actions | |
| `/api/action/:name/run` | POST | Execute an action | |
| `/api/agent/group/generateProfiles` | POST | Generate group profiles | |
| `/api/agent/group/create` | POST | Create a new agent group | |

### Chat Interactions

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/api/chat/:name` | POST | Send message & get response | [Example](#send-message) |
| `/api/notify/:name` | POST | Send notification to agent | [Example](#notify-agent) |
| `/api/sse/:name` | GET | Real-time agent event stream | [Example](#agent-sse-stream) |
| `/v1/responses` | POST | Send message & get response | [OpenAI's Responses](https://platform.openai.com/docs/api-reference/responses/create) |

<details>
<summary><strong>Curl Examples</strong></summary>

#### Get All Agents
```bash
curl -X GET "http://localhost:3000/api/agents"
```

#### Get Agent Status
```bash
curl -X GET "http://localhost:3000/api/agent/my-agent/status"
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
curl -X DELETE "http://localhost:3000/api/agent/my-agent"
```

#### Pause Agent
```bash
curl -X PUT "http://localhost:3000/api/agent/my-agent/pause"
```

#### Start Agent
```bash
curl -X PUT "http://localhost:3000/api/agent/my-agent/start"
```

#### Get Agent Configuration
```bash
curl -X GET "http://localhost:3000/api/agent/my-agent/config"
```

#### Update Agent Configuration
```bash
curl -X PUT "http://localhost:3000/api/agent/my-agent/config" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "system_prompt": "You are an AI assistant."
  }'
```

#### Export Agent
```bash
curl -X GET "http://localhost:3000/settings/export/my-agent" --output my-agent.json
```

#### Import Agent
```bash
curl -X POST "http://localhost:3000/settings/import" \
  -F "file=@/path/to/my-agent.json"
```

#### Send Message
```bash
curl -X POST "http://localhost:3000/api/chat/my-agent" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, how are you today?"}'
```

#### Notify Agent
```bash
curl -X POST "http://localhost:3000/api/notify/my-agent" \
  -H "Content-Type: application/json" \
  -d '{"message": "Important notification"}'
```

#### Agent SSE Stream
```bash
curl -N -X GET "http://localhost:3000/api/sse/my-agent"
```
Note: For proper SSE handling, you should use a client that supports SSE natively.

</details>

### Agent Configuration Reference

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

## LICENSE

MIT License — See the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>LOCAL PROCESSING. GLOBAL THINKING.</strong><br>
  Made with ❤️ by <a href="https://github.com/mudler">mudler</a>
</p>
