<p align="center">
  <img src="./webui/react-ui/public/logo_1.png" alt="LocalAGI Logo" width="220"/>
</p>

<h3 align="center"><em>Your AI. Your Hardware. Your Rules</em></h3>

<div align="center">
  
[![Go Report Card](https://goreportcard.com/badge/github.com/mudler/LocalAGI)](https://goreportcard.com/report/github.com/mudler/LocalAGI)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub stars](https://img.shields.io/github/stars/mudler/LocalAGI)](https://github.com/mudler/LocalAGI/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/mudler/LocalAGI)](https://github.com/mudler/LocalAGI/issues)


Try on [![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)](https://t.me/LocalAGI_bot)

</div>

Create customizable AI assistants, automations, chat bots and agents that run 100% locally. No need for agentic Python libraries or cloud service keys, just bring your GPU (or even just CPU) and a web browser.

**LocalAGI** is a powerful, self-hostable AI Agent platform that allows you to design AI automations without writing code. Create Agents with a couple of clicks, connect via MCP, and use built-in **Skills** (manage skills in the Web UI and enable them per agent). Every agent exposes a complete drop-in replacement for OpenAI's Responses APIs with advanced agentic capabilities. No clouds. No data leaks. Just pure local AI that works on consumer-grade hardware (CPU and GPU). Skills follow the [skillserver](https://github.com/mudler/skillserver) format and can be created, imported, or synced from git.

## 🛡️ Take Back Your Privacy

Are you tired of AI wrappers calling out to cloud APIs, risking your privacy? So were we.

LocalAGI ensures your data stays exactly where you want it—on your hardware. No API keys, no cloud subscriptions, no compromise.

## 🌟 Key Features

- 🎛 **No-Code Agents**: Easy-to-configure multiple agents via Web UI.
- 🖥 **Web-Based Interface**: Simple and intuitive agent management.
- 🤖 **Advanced Agent Teaming**: Instantly create cooperative agent teams from a single prompt.
- 📡 **Connectors**: Built-in integrations with Discord, Slack, Telegram, GitHub Issues, and IRC.
- 🛠 **Comprehensive REST API**: Seamless integration into your workflows. Every agent created will support OpenAI Responses API out of the box.
- 📚 **Short & Long-Term Memory**: Powered by [LocalRecall](https://github.com/mudler/LocalRecall).
- 🧠 **Planning & Reasoning**: Agents intelligently plan, reason, and adapt.
- 🔄 **Periodic Tasks**: Schedule tasks with cron-like syntax.
- 💾 **Memory Management**: Control memory usage with options for long-term and summary memory.
- 🖼 **Multimodal Support**: Ready for vision, text, and more.
- 🔧 **Extensible Custom Actions**: Easily script dynamic agent behaviors in Go (interpreted, no compilation!).
- 📚 **Built-in Skills**: Manage reusable agent skills in the Web UI (create, edit, import/export, git sync). Enable "Skills" per agent to inject skill tools and the skill list into the agent.
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

# AMD GPU setup
docker compose -f docker-compose.amd.yaml up

# Start with a specific model (see available models in models.localai.io, or localai.io to use any model in huggingface)
MODEL_NAME=gemma-3-12b-it docker compose up

# NVIDIA GPU setup with custom multimodal and image models
MODEL_NAME=gemma-3-12b-it \
MULTIMODAL_MODEL=moondream2-20250414 \
IMAGE_MODEL=flux.1-dev-ggml \
docker compose -f docker-compose.nvidia.yaml up
```

Now you can access and manage your agents at [http://localhost:8080](http://localhost:8080)

Still having issues? see this Youtube video: https://youtu.be/HtVwIxW3ePg

## Videos

[![Creating a basic agent](https://img.youtube.com/vi/HtVwIxW3ePg/mqdefault.jpg)](https://youtu.be/HtVwIxW3ePg)
[![Agent Observability](https://img.youtube.com/vi/v82rswGJt_M/mqdefault.jpg)](https://youtu.be/v82rswGJt_M)
[![Filters and Triggers](https://img.youtube.com/vi/d_we-AYksSw/mqdefault.jpg)](https://youtu.be/d_we-AYksSw)
[![RAG and Matrix](https://img.youtube.com/vi/2Xvx78i5oBs/mqdefault.jpg)](https://youtu.be/2Xvx78i5oBs)


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
  - Text: `gemma-3-4b-it-qat`
  - Multimodal: `moondream2-20250414`
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
  - Text: `gemma-3-4b-it-qat`
  - Multimodal: `moondream2-20250414`
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
MULTIMODAL_MODEL=moondream2-20250414 \
IMAGE_MODEL=flux.1-dev-ggml \
docker compose -f docker-compose.nvidia.yaml up

# Intel GPU with custom models
MODEL_NAME=gemma-3-12b-it \
MULTIMODAL_MODEL=moondream2-20250414 \
IMAGE_MODEL=sd-1.5-ggml \
docker compose -f docker-compose.intel.yaml up

# With custom actions directory
LOCALAGI_CUSTOM_ACTIONS_DIR=/app/custom-actions docker compose up
```

If no models are specified, it will use the defaults:
- Text model: `gemma-3-4b-it-qat`
- Multimodal model: `moondream2-20250414`
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
- **✓ Feature-Rich**: From planning to multimodal capabilities, connectors for Slack, MCP support, built-in Skills, LocalAGI has it all.

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
- [Skills](#3-skills)

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
| `LOCALAGI_CUSTOM_ACTIONS_DIR` | Directory containing custom Go action files to be automatically loaded |

Skills are stored in a fixed `skills` subdirectory under `LOCALAGI_STATE_DIR` (e.g. `/pool/skills` in Docker). Git repo config for skills lives in that directory. No extra environment variables are required.

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

### Using as a Library

LocalAGI can be used as a Go library to programmatically create and manage AI agents. Let's start with a simple example of creating a single agent:

<details>
<summary><strong>Basic Usage: Single Agent</strong></summary>

```go
import (
    "github.com/mudler/LocalAGI/core/agent"
    "github.com/mudler/LocalAGI/core/types"
)

// Create a new agent with basic configuration
agent, err := agent.New(
    agent.WithModel("gpt-4"),
    agent.WithLLMAPIURL("http://localhost:8080"),
    agent.WithLLMAPIKey("your-api-key"),
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithCharacter(agent.Character{
        Name: "my-agent",
    }),
    agent.WithActions(
        // Add your custom actions here
    ),
    agent.WithStateFile("./state/my-agent.state.json"),
    agent.WithCharacterFile("./state/my-agent.character.json"),
    agent.WithTimeout("10m"),
    agent.EnableKnowledgeBase(),
    agent.EnableReasoning(),
)

if err != nil {
    log.Fatal(err)
}

// Start the agent
go func() {
    if err := agent.Run(); err != nil {
        log.Printf("Agent stopped: %v", err)
    }
}()

// Stop the agent when done
agent.Stop()
```

This basic example shows how to:
- Create a single agent with essential configuration
- Set up the agent's model and API connection
- Configure basic features like knowledge base and reasoning
- Start and stop the agent

</details>

<details>
<summary><strong>Advanced Usage: Agent Pools</strong></summary>

For managing multiple agents, you can use the AgentPool system:

```go
import (
    "github.com/mudler/LocalAGI/core/state"
    "github.com/mudler/LocalAGI/core/types"
)

// Create a new agent pool
pool, err := state.NewAgentPool(
    "default-model",           // default model name
    "default-multimodal-model", // default multimodal model
    "image-model",            // image generation model
    "http://localhost:8080",  // API URL
    "your-api-key",          // API key
    "./state",               // state directory
    "http://localhost:8081", // LocalRAG API URL
    func(config *AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action {
        // Define available actions for agents
        return func(ctx context.Context, pool *AgentPool) []types.Action {
            return []types.Action{
                // Add your custom actions here
            }
        }
    },
    func(config *AgentConfig) []Connector {
        // Define connectors for agents
        return []Connector{
            // Add your custom connectors here
        }
    },
    func(config *AgentConfig) []DynamicPrompt {
        // Define dynamic prompts for agents
        return []DynamicPrompt{
            // Add your custom prompts here
        }
    },
    func(config *AgentConfig) types.JobFilters {
        // Define job filters for agents
        return types.JobFilters{
            // Add your custom filters here
        }
    },
    "10m", // timeout
    true,  // enable conversation logs
)

// Create a new agent in the pool
agentConfig := &AgentConfig{
    Name: "my-agent",
    Model: "gpt-4",
    SystemPrompt: "You are a helpful assistant.",
    EnableKnowledgeBase: true,
    EnableReasoning: true,
    // Add more configuration options as needed
}

err = pool.CreateAgent("my-agent", agentConfig)

// Start all agents
err = pool.StartAll()

// Get agent status
status := pool.GetStatusHistory("my-agent")

// Stop an agent
pool.Stop("my-agent")

// Remove an agent
err = pool.Remove("my-agent")
```

</details>

<details>
<summary><strong>Available Features</strong></summary>

Key features available through the library:

- **Single Agent Management**: Create and manage individual agents with basic configuration
- **Agent Pool Management**: Create, start, stop, and remove multiple agents
- **Configuration**: Customize agent behavior through AgentConfig
- **Actions**: Define custom actions for agents to perform
- **Connectors**: Add custom connectors for external services
- **Dynamic Prompts**: Create dynamic prompt templates
- **Job Filters**: Implement custom job filtering logic
- **Status Tracking**: Monitor agent status and history
- **State Persistence**: Automatic state saving and loading

For more details about available configuration options and features, refer to the [Agent Configuration Reference](#agent-configuration-reference) section.

</details>

## 🔧 Extending LocalAGI

LocalAGI provides two powerful ways to extend its functionality with custom actions:

### 1. Custom Actions (Go Code)

LocalAGI supports custom actions written in Go that can be defined inline when creating an agent. These actions are interpreted at runtime, so no compilation is required.

#### Automatic Custom Actions Loading

You can also place custom Go action files in a directory and have LocalAGI automatically load them. Set the `LOCALAGI_CUSTOM_ACTIONS_DIR` environment variable to point to a directory containing your custom action files. Each `.go` file in this directory will be automatically loaded and made available to all agents.

**Example setup:**
```bash
# Set the environment variable
export LOCALAGI_CUSTOM_ACTIONS_DIR="/path/to/custom/actions"

# Or in docker-compose.yaml
environment:
  - LOCALAGI_CUSTOM_ACTIONS_DIR=/app/custom-actions
```

**Directory structure:**
```
custom-actions/
├── weather_action.go
├── file_processor.go
└── database_query.go
```

Each file should contain the three required functions (`Run`, `Definition`, `RequiredFields`) as described below.

#### How Custom Actions Work

When creating a new Agent, in the action sections select the "custom" action, you can add the Golang code directly there.

Custom actions in LocalAGI require three main functions:

1. **`Run(config map[string]interface{}) (string, map[string]interface{}, error)`** - The main execution function
2. **`Definition() map[string][]string`** - Defines the action's parameters and their types
3. **`RequiredFields() []string`** - Specifies which parameters are required

Note: You can't use additional modules, but just use libraries that are included in Go.

#### Example: Weather Information Action

Here's a practical example of a custom action that fetches weather information:

```go
import (
    "encoding/json"
    "fmt"
    "net/http"
    "io"
)

type WeatherParams struct {
    City    string `json:"city"`
    Country string `json:"country"`
}

type WeatherResponse struct {
    Main struct {
        Temp     float64 `json:"temp"`
        Humidity int     `json:"humidity"`
    } `json:"main"`
    Weather []struct {
        Description string `json:"description"`
    } `json:"weather"`
}

func Run(config map[string]interface{}) (string, map[string]interface{}, error) {
    // Parse parameters
    p := WeatherParams{}
    b, err := json.Marshal(config)
    if err != nil {
        return "", map[string]interface{}{}, err
    }
    if err := json.Unmarshal(b, &p); err != nil {
        return "", map[string]interface{}{}, err
    }

    // Make API call to weather service
    url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s,%s&appid=YOUR_API_KEY&units=metric", p.City, p.Country)
    resp, err := http.Get(url)
    if err != nil {
        return "", map[string]interface{}{}, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", map[string]interface{}{}, err
    }

    var weather WeatherResponse
    if err := json.Unmarshal(body, &weather); err != nil {
        return "", map[string]interface{}{}, err
    }

    // Format response
    result := fmt.Sprintf("Weather in %s, %s: %.1f°C, %s, Humidity: %d%%", 
        p.City, p.Country, weather.Main.Temp, weather.Weather[0].Description, weather.Main.Humidity)

    return result, map[string]interface{}{}, nil
}

func Definition() map[string][]string {
    return map[string][]string{
        "city": []string{
            "string",
            "The city name to get weather for",
        },
        "country": []string{
            "string", 
            "The country code (e.g., US, UK, DE)",
        },
    }
}

func RequiredFields() []string {
    return []string{"city", "country"}
}
```

#### Example: File System Action

Here's another example that demonstrates file system operations:

```go
import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type FileParams struct {
    Path    string `json:"path"`
    Action  string `json:"action"`
    Content string `json:"content,omitempty"`
}

func Run(config map[string]interface{}) (string, map[string]interface{}, error) {
    p := FileParams{}
    b, err := json.Marshal(config)
    if err != nil {
        return "", map[string]interface{}{}, err
    }
    if err := json.Unmarshal(b, &p); err != nil {
        return "", map[string]interface{}{}, err
    }

    switch p.Action {
    case "read":
        content, err := os.ReadFile(p.Path)
        if err != nil {
            return "", map[string]interface{}{}, err
        }
        return string(content), map[string]interface{}{}, nil
        
    case "write":
        err := os.WriteFile(p.Path, []byte(p.Content), 0644)
        if err != nil {
            return "", map[string]interface{}{}, err
        }
        return fmt.Sprintf("Successfully wrote to %s", p.Path), map[string]interface{}{}, nil
        
    case "list":
        files, err := os.ReadDir(p.Path)
        if err != nil {
            return "", map[string]interface{}{}, err
        }
        
        var fileList []string
        for _, file := range files {
            fileList = append(fileList, file.Name())
        }
        
        result, _ := json.Marshal(fileList)
        return string(result), map[string]interface{}{}, nil
        
    default:
        return "", map[string]interface{}{}, fmt.Errorf("unknown action: %s", p.Action)
    }
}

func Definition() map[string][]string {
    return map[string][]string{
        "path": []string{
            "string",
            "The file or directory path",
        },
        "action": []string{
            "string",
            "The action to perform: read, write, or list",
        },
        "content": []string{
            "string",
            "Content to write (required for write action)",
        },
    }
}

func RequiredFields() []string {
    return []string{"path", "action"}
}
```

#### Using Custom Actions in Agents

To use custom actions, add them to your agent configuration:

1. **Via Web UI**: In the agent creation form, add a "Custom" action and paste your Go code
2. **Via API**: Include the custom action in your agent configuration JSON
3. **Via Library**: Add the custom action to your agent's actions list

### 2. MCP (Model Context Protocol) Servers

LocalAGI supports both local and remote MCP servers, allowing you to extend functionality with external tools and services.

#### What is MCP?

The Model Context Protocol (MCP) is a standard for connecting AI applications to external data sources and tools. LocalAGI can connect to any MCP-compliant server to access additional capabilities.

#### Local MCP Servers

Local MCP servers run as processes that LocalAGI can spawn and communicate with via STDIO.

##### Example: GitHub MCP Server

```json
{
  "mcpServers": {
    "github": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "GITHUB_PERSONAL_ACCESS_TOKEN",
        "ghcr.io/github/github-mcp-server"
      ],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "<YOUR_TOKEN>"
      }
    }
  }
}
```

#### Remote MCP Servers

Remote MCP servers are HTTP-based and can be accessed over the network.

#### Creating Your Own MCP Server

You can create MCP servers in any language that supports the MCP protocol and add the URLs of the servers to LocalAGI.

#### Configuring MCP Servers in LocalAGI

1. **Via Web UI**: In the MCP Settings section of agent creation, add MCP servers
2. **Via API**: Include MCP server configuration in your agent config

#### Best Practices

- **Security**: Always validate inputs and use proper authentication for remote MCP servers
- **Error Handling**: Implement robust error handling in your MCP servers
- **Documentation**: Provide clear descriptions for all tools exposed by your MCP server
- **Testing**: Test your MCP servers independently before integrating with LocalAGI
- **Resource Management**: Ensure your MCP servers properly clean up resources

### 3. Skills

LocalAGI includes built-in **Skills** management. Skills are reusable instructions and resources (scripts, references, assets) that agents can use when "Enable Skills" is turned on for that agent.

- **Skills section (Web UI)**: Open **Skills** in the sidebar. Skills are stored under the state directory (`STATE_DIR/skills`). Create, edit, search, import, and export skills. You can also add git repositories to sync skills from.
- **Per-agent**: In agent creation or settings, enable **Enable Skills** in Advanced Settings. The agent will receive a list of available skills in its context and have access to skill tools (list, read, search, resources) via the built-in skills MCP.
- Skills use the same format as [skillserver](https://github.com/mudler/skillserver) (e.g. `SKILL.md` in a directory). You can export skills from LocalAGI and use them with the standalone skillserver, or import skills created elsewhere.

In Docker, the state directory is persisted (`/pool`), so skills are stored in `/pool/skills`. To use a host folder for skills, mount it over that path in your compose file (e.g. `- ./my-skills:/pool/skills`).

### Development

The development workflow is similar to the source build, but with additional steps for hot reloading of the frontend:

```bash
# Clone repo
git clone https://github.com/mudler/LocalAGI.git
cd LocalAGI

cd webui/react-ui

# Install dependencies
bun i

# Compile frontend (the build directory needs to exist for the backend to start)
bun run build

# Start frontend development server
bun run dev
```

Then in separate terminal:

```bash
cd LocalAGI

# Create a "pool" directory for agent state
mkdir pool

# Set required environment variables
export LOCALAGI_MODEL=gemma-3-4b-it-qat
export LOCALAGI_MULTIMODAL_MODEL=moondream2-20250414
export LOCALAGI_IMAGE_MODEL=sd-1.5-ggml
export LOCALAGI_LLM_API_URL=http://localai:8080
export LOCALAGI_LOCALRAG_URL=http://localrecall:8080
export LOCALAGI_STATE_DIR=./pool
export LOCALAGI_TIMEOUT=5m
export LOCALAGI_ENABLE_CONVERSATIONS_LOGGING=false
export LOCALAGI_SSHBOX_URL=root:root@sshbox:22

# Start development server
go run main.go
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
</details>

<details>
<summary><strong>Actions and Groups</strong></summary>

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/api/actions` | GET | List available actions | |
| `/api/action/:name/run` | POST | Execute an action | |
| `/api/agent/group/generateProfiles` | POST | Generate group profiles | |
| `/api/agent/group/create` | POST | Create a new agent group | |
</details>

<details>
<summary><strong>Chat Interactions</strong></summary>

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/api/chat/:name` | POST | Send message & get response | [Example](#send-message) |
| `/api/notify/:name` | POST | Send notification to agent | [Example](#notify-agent) |
| `/api/sse/:name` | GET | Real-time agent event stream | [Example](#agent-sse-stream) |
| `/v1/responses` | POST | Send message & get response | [OpenAI's Responses](https://platform.openai.com/docs/api-reference/responses/create) |
</details>

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
| `LOCALAGI_MODEL` | Your go-to model |
| `LOCALAGI_MULTIMODAL_MODEL` | Optional model for multimodal capabilities |
| `LOCALAGI_LLM_API_URL` | OpenAI-compatible API server URL |
| `LOCALAGI_LLM_API_KEY` | API authentication |
| `LOCALAGI_TIMEOUT` | Request timeout settings |
| `LOCALAGI_STATE_DIR` | Where state gets stored |
| `LOCALAGI_LOCALRAG_URL` | LocalRecall connection |
| `LOCALAGI_SSHBOX_URL` | LocalAGI SSHBox URL, e.g. user:pass@ip:port |
| `LOCALAGI_ENABLE_CONVERSATIONS_LOGGING` | Toggle conversation logs |
| `LOCALAGI_API_KEYS` | A comma separated list of api keys used for authentication |
| `LOCALAGI_CUSTOM_ACTIONS_DIR` | Directory containing custom Go action files to be automatically loaded |
</details>

## LICENSE

MIT License — See the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>LOCAL PROCESSING. GLOBAL THINKING.</strong><br>
  Made with ❤️ by <a href="https://github.com/mudler">mudler</a>
</p>
