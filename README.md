<p align="center">
  <img src="https://github.com/user-attachments/assets/304ad402-5ddc-441b-a4b9-55ff9eec72be" alt="LocalAgent Logo" width="200"/>
</p>

<div align="center">
  
[![Go Report Card](https://goreportcard.com/badge/github.com/mudler/LocalAgent)](https://goreportcard.com/report/github.com/mudler/LocalAgent)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub stars](https://img.shields.io/github/stars/mudler/LocalAgent)](https://github.com/mudler/LocalAgent/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/mudler/LocalAgent)](https://github.com/mudler/LocalAgent/issues)
  
</div>

**LocalAgent** is an AI Agent platform with the aim to runs 100% locally on your machine. Designed for privacy, efficiency, and flexibility, LocalAgent allows you to build, deploy, and interact with AI agents without sending your data to external services.

## Features

- **100% Local Execution**: All processing happens on your own hardware - no data leaves your machine
- **Multiple Model Support**: Compatible with various local LLM formats (GGUF, GGML, etc.)
- **Extensible Architecture**: Build custom agents with specialized capabilities
- **Web-based GUI**: User-friendly interface for easy interaction with your agents
- **RESTful API**: Comprehensive API for developers to integrate LocalAgent into their applications
- **Works well locally!**: It is well tested and meant to work with [LocalAI](https://github.com/mudler/LocalAI)
- **Cross-platform**: Works on Linux, macOS, and Windows

LocalAgent is part of a set of open source tools aimed to streamline AI usage locally, see also its sister projects:

- [LocalAI](https://github.com/mudler/LocalAI)
- [LocalRAG](https://github.com/mudler/LocalRAG)

## Installation

### Prerequisites

For building from source:

- Go 1.20 or later
- Git

### From Source

```bash
# Clone the repository
git clone https://github.com/mudler/LocalAgent.git
cd LocalAgent

# Build the application
go build -o localagent

# Run LocalAgent
./localagent
```

### Using Docker containers

```bash
docker run -ti -p 3000:3000 -v  quay.io/mudler/localagent
```

### Pre-built Binaries

Download the pre-built binaries for your platform from the [Releases](https://github.com/mudler/LocalAgent/releases) page.

## Getting Started

After installation, you can start LocalAgent with default settings:

```bash
./localagent
```

This will start both the API server and the web interface. By default, the web interface is accessible at `http://localhost:3000`.

### Environment Variables

LocalAgent can be configured using the following environment variables:

| Variable                      | Description                                      |
|-------------------------------|--------------------------------------------------|
| `LOCALAGENT_MODEL`                  | Specifies the test model to use                  |
| `LOCALAGENT_LLM_API_URL`                     | URL of the API server                            |
| `LOCALAGENT_API_KEY`                     | API key for authentication                       |
| `LOCALAGENT_TIMEOUT`                     | Timeout duration for requests                    |
| `LOCALAGENT_STATE_DIR`                   | Directory to store state information             |
| `LOCALAGENT_LOCALRAG_URL`                   | LocalRAG URL               |
| `LOCALAGENT_ENABLE_CONVERSATIONS_LOGGING`| Enable or disable logging of conversations       |

## Documentation

### Connectors

LocalAgent can be connected to a wide range of services. Each service support a set of configuration, examples are provided below for every connector.

#### Github (issues)

Create an user and a PAT token, and associate to a repository:

```json
{
	"token": "PAT_TOKEN",
    "repository": "repository-to-watch-issues",
    "owner": "repository-owner",
    "botUserName": "username"
}
```

#### Discord

Follow the steps in: https://discordpy.readthedocs.io/en/stable/discord.html to create a discord bot.   

The token of the bot is in the "Bot" tab. Also enable " Message Content Intent " in the Bot tab!

```json
{
"token": "Bot DISCORDTOKENHERE",
"defaultChannel": "OPTIONALCHANNELINT"
}
```

#### Slack

See slack.yaml

- Create a new App from a manifest (copy-paste from `slack.yaml`)
- Create Oauth token bot token from "OAuth & Permissions" -> "OAuth Tokens for Your Workspace"
- Create App level token (from "Basic Information" -> "App-Level Tokens" ( `scope connections:writeRoute authorizations:read` ))

In the UI, when configuring the connector:

```json
{
"botToken": "xoxb-...",
"appToken": "xapp-1-..."
}
```

#### Telegram

Ask a token to @botfather

In the UI, when configuring the connector:

```json
{ "token": "botfathertoken" }
```

### REST API

The LocalAgent API follows RESTful principles and uses JSON for request and response bodies.





## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/mudler">mudler</a>
</p>