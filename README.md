<p align="center">
  <img src="./webui/react-ui/public/logo_1.png" alt="LocalAGI Logo" width="220"/>
</p>

<h3 align="center"><em>Your AI. Your Hardware. Your Rules</em></h3>

<div align="center">
  
[![Go Report Card](https://goreportcard.com/badge/github.com/mudler/LocalAGI)](https://goreportcard.com/report/github.com/mudler/LocalAGI)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub stars](https://img.shields.io/github/stars/mudler/LocalAGI)](https://github.com/mudler/LocalAGI/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/mudler/LocalAGI)](https://github.com/mudler/LocalAGI/issues)

</div>

Create customizable AI assistants, automations, chat bots and agents that run 100% locally. No need for agentic Python libraries or cloud service keys, just bring your GPU (or even just CPU) and a web browser.

**LocalAGI** is a powerful, self-hostable AI Agent platform that allows you to design AI automations without writing code. A complete drop-in replacement for OpenAI's Responses APIs with advanced agentic capabilities. No clouds. No data leaks. Just pure local AI that works on consumer-grade hardware (CPU and GPU).

## 🛡️ Take Back Your Privacy

Are you tired of AI wrappers calling out to cloud APIs, risking your privacy? So were we.

LocalAGI ensures your data stays exactly where you want it—on your hardware. No API keys, no cloud subscriptions, no compromise.

## 🌟 Key Features

- 🎛 **No-Code Agents**: Easy-to-configure multiple agents via Web UI.
- 🖥 **Web-Based Interface**: Simple and intuitive agent management.
- 🤖 **Advanced Agent Teaming**: Instantly create cooperative agent teams from a single prompt.
- 📡 **Connectors Galore**: Built-in integrations with Discord, Slack, Telegram, GitHub Issues, and IRC.
- 🛠 **Comprehensive REST API**: Seamless integration into your workflows. Every agent created will support OpenAI Responses API out of the box.
- 📚 **Short & Long-Term Memory**: Powered by [LocalRecall](https://github.com/mudler/LocalRecall).
- 🧠 **Planning & Reasoning**: Agents intelligently plan, reason, and adapt.
- 🔄 **Periodic Tasks**: Schedule tasks with cron-like syntax.
- 💾 **Memory Management**: Control memory usage with options for long-term and summary memory.
- 🖼 **Multimodal Support**: Ready for vision, text, and more.
- 🔧 **Extensible Custom Actions**: Easily script dynamic agent behaviors in Go (interpreted, no compilation!).
- 🛠 **Fully Customizable Models**: Use your own models or integrate seamlessly with [LocalAI](https://github.com/mudler/LocalAI).
- 📊 **Observability**: Monitor agent status and view detailed observable updates in real-time.

## 🛠️ Quickstart

```bash
# Clone the repository
git clone https://github.com/mudler/LocalAGI
cd LocalAGI

# CPU setup (default)
docker compose up

# NVIDIA GPU setup
docker compose -f docker-compose.nvidia.yaml up

# Intel GPU setup (for Intel Arc and integrated GPUs)
docker compose -f docker-compose.intel.yaml up

# Start with a specific model (see available models in models.localai.io, or localai.io to use any model in huggingface)
MODEL_NAME=gemma-3-12b-it docker compose up

# NVIDIA GPU setup with custom multimodal and image models
MODEL_NAME=gemma-3-12b-it \
MULTIMODAL_MODEL=minicpm-v-2_6 \
IMAGE_MODEL=flux.1-dev-ggml \
docker compose -f docker-compose.nvidia.yaml up
```

Now you can access and manage your agents at [http://localhost:8080](http://localhost:8080)

Still having issues? see this Youtube video: https://youtu.be/HtVwIxW3ePg

## Videos

[![Creating a basic agent](https://img.youtube.com/vi/HtVwIxW3ePg/mqdefault.jpg)](https://youtu.be/HtVwIxW3ePg)
[![Agent Observability](https://img.youtube.com/vi/v82rswGJt_M/mqdefault.jpg)](https://youtu.be/v82rswGJt_M)

## 📚🆕 Local Stack Family

🆕 LocalAI is now part of a comprehensive suite of AI tools designed to work together:

<table>
  <tr>
    <td width="50%" valign="top">
      <a href="https://github.com/mudler/LocalAI">
        <img src="https://raw.githubusercontent.com/mudler/LocalAI/refs/heads/master/core/http/static/logo_horizontal.png" width="300" alt="LocalAI Logo">
      </a>
    </td>
    <td width="50%" valign="top">
      <h3><a href="https://github.com/mudler/LocalAI">LocalAI</a></h3>
      <p>LocalAI is the free, Open Source OpenAI alternative. LocalAI act as a drop-in replacement REST API that's compatible with OpenAI API specifications for local AI inferencing. Does not require GPU.</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <a href="https://github.com/mudler/LocalRecall">
        <img src="https://raw.githubusercontent.com/mudler/LocalRecall/refs/heads/main/static/localrecall_horizontal.png" width="300" alt="LocalRecall Logo">
      </a>
    </td>
    <td width="50%" valign="top">
      <h3><a href="https://github.com/mudler/LocalRecall">LocalRecall</a></h3>
      <p>A REST-ful API and knowledge base management system that provides persistent memory and storage capabilities for AI agents.</p>
    </td>
  </tr>
</table>

## 🖥️ Hardware Configurations

LocalAGI supports multiple hardware configurations through Docker Compose profiles:

### CPU (Default)
- No special configuration needed
- Runs on any system with Docker
- Best for testing and development
- Supports text models only

### NVIDIA GPU
- Requires NVIDIA GPU and drivers
- Uses CUDA for acceleration
- Best for high-performance inference
- Supports text, multimodal, and image generation models
- Run with: `docker compose -f docker-compose.nvidia.yaml up`
- Default models:
  - Text: `gemma-3-12b-it-qat`
  - Multimodal: `minicpm-v-2_6`
  - Image: `sd-1.5-ggml`
- Environment variables:
  - `MODEL_NAME`: Text model to use
  - `MULTIMODAL_MODEL`: Multimodal model to use
  - `IMAGE_MODEL`: Image generation model to use
  - `LOCALAI_SINGLE_ACTIVE_BACKEND`: Set to `true` to enable single active backend mode

### Intel GPU
- Supports Intel Arc and integrated GPUs
- Uses SYCL for acceleration
- Best for Intel-based systems
- Supports text, multimodal, and image generation models
- Run with: `docker compose -f docker-compose.intel.yaml up`
- Default models:
  - Text: `gemma-3-12b-it-qat`
  - Multimodal: `minicpm-v-2_6`
  - Image: `sd-1.5-ggml`
- Environment variables:
  - `MODEL_NAME`: Text model to use
  - `MULTIMODAL_MODEL`: Multimodal model to use
  - `IMAGE_MODEL`: Image generation model to use
  - `LOCALAI_SINGLE_ACTIVE_BACKEND`: Set to `true` to enable single active backend mode

## Customize models

You can customize the models used by LocalAGI by setting environment variables when running docker-compose. For example:

```bash
# CPU with custom model
MODEL_NAME=gemma-3-12b-it docker compose up

# NVIDIA GPU with custom models
MODEL_NAME=gemma-3-12b-it \
MULTIMODAL_MODEL=minicpm-v-2_6 \
IMAGE_MODEL=flux.1-dev-ggml \
docker compose -f docker-compose.nvidia.yaml up

# Intel GPU with custom models
MODEL_NAME=gemma-3-12b-it \
MULTIMODAL_MODEL=minicpm-v-2_6 \
IMAGE_MODEL=sd-1.5-ggml \
docker compose -f docker-compose.intel.yaml up
```

If no models are specified, it will use the defaults:
- Text model: `gemma-3-12b-it-qat`
- Multimodal model: `minicpm-v-2_6`
- Image model: `sd-1.5-ggml`

Good (relatively small) models that have been tested are:

- `qwen_qwq-32b` (best in co-ordinating agents)
- `gemma-3-12b-it`
- `gemma-3-27b-it`

## 🏆 Why Choose LocalAGI?

- **✓ Ultimate Privacy**: No data ever leaves your hardware.
- **✓ Flexible Model Integration**: Supports GGUF, GGML, and more thanks to [LocalAI](https://github.com/mudler/LocalAI).
- **✓ Developer-Friendly**: Rich APIs and intuitive interfaces.
- **✓ Effortless Setup**: Simple Docker compose setups and pre-built binaries.
- **✓ Feature-Rich**: From planning to multimodal capabilities, connectors for Slack, MCP support, LocalAGI has it all.

## 🌟 Screenshots

### Powerful Web UI

![Web UI Dashboard](https://github.com/user-attachments/assets/a40194f9-af3a-461f-8b39-5f4612fbf221)
![Web UI Agent Settings](https://github.com/user-attachments/assets/fb3c3e2a-cd53-4ca8-97aa-c5da51ff1f83)
![Web UI Create Group](https://github.com/user-attachments/assets/102189a2-0fba-4a1e-b0cb-f99268ef8062)
![Web UI Agent Observability](https://github.com/user-attachments/assets/f7359048-9d28-4cf1-9151-1f5556ce9235)


### Connectors Ready-to-Go

<p align="center">
  <img src="https://github.com/user-attachments/assets/4171072f-e4bf-4485-982b-55d55086f8fc" alt="Telegram" width="60"/>
  <img src="https://github.com/user-attachments/assets/9235da84-0187-4f26-8482-32dcc55702ef" alt="Discord" width="220"/>
  <img src="https://github.com/user-attachments/assets/a88c3d88-a387-4fb5-b513-22bdd5da7413" alt="Slack" width="220"/>
  <img src="https://github.com/user-attachments/assets/d249cdf5-ab34-4ab1-afdf-b99e2db182d2" alt="IRC" width="220"/>
  <img src="https://github.com/user-attachments/assets/52c852b0-4b50-4926-9fa0-aa50613ac622" alt="GitHub" width="220"/>
</p>

## 📖 Full Documentation

Explore detailed documentation including:
- [Installation Options](#installation-options)
- [REST API Documentation](#rest-api)
- [Connector Configuration](#connectors)
- [Agent Configuration](#agent-configuration-reference)

### Environment Configuration

LocalAGI supports environment configurations. Note that these environment variables needs to be specified in the localagi container in the docker-compose file to have effect.

| Variable | What It Does |
|----------|--------------|
| `LOCALAGI_MODEL` | Your go-to model |
| `LOCALAGI_MULTIMODAL_MODEL` | Optional model for multimodal capabilities |
| `LOCALAGI_LLM_API_URL` | OpenAI-compatible API server URL |
| `LOCALAGI_LLM_API_KEY` | API authentication |
| `LOCALAGI_TIMEOUT` | Request timeout settings |
| `LOCALAGI_STATE_DIR` | Where state gets stored |
| `LOCALAGI_LOCALRAG_URL` | LocalRecall connection |
| `LOCALAGI_ENABLE_CONVERSATIONS_LOGGING` | Toggle conversation logs |
| `LOCALAGI_API_KEYS` | A comma separated list of api keys used for authentication |

## Installation Options

### Pre-Built Binaries

Download ready-to-run binaries from the [Releases](https://github.com/mudler/LocalAGI/releases) page.

### Source Build

Requirements:
- Go 1.20+
- Git
- Bun 1.2+

```bash
# Clone repo
git clone https://github.com/mudler/LocalAGI.git
cd LocalAGI

# Build it
cd webui/react-ui && bun i && bun run build
cd ../..
go build -o localagi

# Run it
./localagi
```

### Development

The development workflow is similar to the source build, but with additional steps for hot reloading of the frontend:

```bash
# Clone repo
git clone https://github.com/mudler/LocalAGI.git
cd LocalAGI

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
  "nickname": "LocalAGIBot",
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
