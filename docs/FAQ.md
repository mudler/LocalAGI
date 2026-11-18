# Frequently Asked Questions (FAQ)

## General Questions

### What is LocalAGI?
LocalAGI is a self-hostable AI Agent platform that runs 100% locally on your hardware. It allows you to create customizable AI assistants, automations, chat bots, and agents without requiring Python libraries or cloud service keys. Your data stays on your hardware - no clouds, no data leaks.

### How does LocalAGI differ from other AI agent frameworks?

| Framework | Key Difference |
|-----------|---------------|
| **LangChain** | LangChain is a Python library for building LLM applications. LocalAGI is a complete platform with Web UI, connectors, and built-in skills - no coding required. |
| **AutoGPT** | AutoGPT is a single autonomous agent. LocalAGI supports multiple agents, agent teams, and cooperative workflows via Web UI. |
| **CrewAI** | CrewAI focuses on role-playing multi-agent systems in Python. LocalAGI offers no-code agent creation with built-in connectors (Discord, Slack, Telegram, IRC, GitHub). |
| **OpenAI Operator** | OpenAI Operator is a cloud-based service. LocalAGI runs entirely locally with LocalAI integration - no API keys needed, complete privacy. |

### Why should I choose LocalAGI?
- **✓ Ultimate Privacy**: No data ever leaves your hardware
- **✓ No Cloud Dependencies**: Works with LocalAI, no external API keys required
- **✓ No-Code Interface**: Create agents via Web UI without programming
- **✓ Built-in Connectors**: Ready-to-use Discord, Slack, Telegram, IRC, GitHub integrations
- **✓ Skills Management**: Create, import, and sync reusable agent skills
- **✓ Knowledge Base**: Built-in RAG with semantic search for long-term memory
- **✓ MCP Support**: Connect to MCP servers for extended capabilities
- **✓ Multimodal**: Supports vision, text, and image generation

## Installation

### What are the system requirements?
- **CPU**: Any modern processor (works on consumer-grade hardware)
- **GPU**: Optional but recommended for faster inference (NVIDIA, Intel Arc, AMD)
- **RAM**: 8GB minimum, 16GB+ recommended for larger models
- **Storage**: Depends on models (GGUF models are typically 2-8GB)
- **Docker**: Required for containerized deployment

### How do I install LocalAGI?
```bash
# Clone the repository
git clone https://github.com/mudler/LocalAGI
cd LocalAGI

# CPU setup (default)
docker compose up

# NVIDIA GPU setup
docker compose -f docker-compose.nvidia.yaml up

# Intel GPU setup
docker compose -f docker-compose.intel.yaml up

# AMD GPU setup
docker compose -f docker-compose.amd.yaml up
```

Access the Web UI at http://localhost:8080

### Can I use LocalAGI without Docker?
Yes! Download pre-built binaries from the [Releases](https://github.com/mudler/LocalAGI/releases) page, or build from source:
```bash
git clone https://github.com/mudler/LocalAGI.git
cd LocalAGI
cd webui/react-ui && bun i && bun run build
cd ../..
go build -o localagi
./localagi
```

## Models and Configuration

### What models can I use with LocalAGI?
LocalAGI integrates with LocalAI, supporting:
- **GGUF/GGML models**: Any model from HuggingFace in GGUF format
- **Recommended models**:
  - `qwen_qwq-32b` (best for coordinating agents)
  - `gemma-3-12b-it`
  - `gemma-3-27b-it`
- **Default models**:
  - Text: `gemma-3-4b-it-qat`
  - Multimodal: `moondream2-20250414`
  - Image: `sd-1.5-ggml` or `flux.1-dev-ggml`

### How do I customize models?
Set environment variables when running docker-compose:
```bash
# CPU with custom model
MODEL_NAME=gemma-3-12b-it docker compose up

# NVIDIA GPU with custom models
MODEL_NAME=gemma-3-12b-it \
MULTIMODAL_MODEL=moondream2-20250414 \
IMAGE_MODEL=flux.1-dev-ggml \
docker compose -f docker-compose.nvidia.yaml up
```

### Can I use cloud LLM providers?
Yes! Set `LOCALAGI_LLM_API_URL` and `LOCALAGI_LLM_API_KEY` to connect to OpenAI-compatible APIs (OpenAI, Anthropic, Azure, etc.). However, using LocalAI locally preserves privacy.

### What environment variables are available?

| Variable | Purpose |
|----------|---------|
| `LOCALAGI_MODEL` | Default text model |
| `LOCALAGI_MULTIMODAL_MODEL` | Multimodal model for vision |
| `LOCALAGI_LLM_API_URL` | OpenAI-compatible API URL |
| `LOCALAGI_LLM_API_KEY` | API authentication key |
| `LOCALAGI_TIMEOUT` | Request timeout (default: 10m) |
| `LOCALAGI_STATE_DIR` | State storage directory |
| `LOCALAGI_ENABLE_CONVERSATIONS_LOGGING` | Enable conversation logs |
| `LOCALAGI_API_KEYS` | Comma-separated API keys for authentication |
| `LOCALAGI_CUSTOM_ACTIONS_DIR` | Directory for custom Go action files |

## Features

### What are Skills?
Skills are reusable instructions and resources (scripts, references, assets) that agents can use. Manage skills in the Web UI under **Skills** section. Enable "Enable Skills" per agent to inject skill tools and list into the agent context. Skills follow the [skillserver](https://github.com/mudler/skillserver) format.

### What is the Knowledge Base?
The Knowledge Base is a built-in RAG (Retrieval-Augmented Generation) system for long-term memory. Agents with "Knowledge base" enabled can:
- Store and retrieve information across sessions
- Upload files and create collections
- Perform semantic search on stored content
- Uses LocalRecall libraries internally

### What connectors are available?
Built-in connectors for immediate integration:
- **Discord**: Bot integration with message content intent
- **Slack**: App with OAuth and app-level tokens
- **Telegram**: Bot via BotFather with group support
- **IRC**: Connect to IRC networks
- **GitHub Issues**: Monitor and respond to issues
- **Email**: SMTP/IMAP integration

### How do agent teams work?
Create cooperative agent teams from a single prompt in the Web UI. Agents can:
- Share information through the knowledge base
- Coordinate tasks with planning and reasoning
- Have different roles and capabilities
- Work on complex multi-step workflows

### What is MCP support?
MCP (Model Context Protocol) allows connecting to external tools and data sources:
- **Local MCP servers**: STDIO-based servers spawned by LocalAGI
- **Remote MCP servers**: HTTP-based servers over network
- Configure in agent settings via Web UI or API

### How does planning and reasoning work?
Agents with "Enable Reasoning" can:
- Plan multi-step actions before execution
- Reason about available tools and information
- Adapt plans based on results
- Break down complex tasks into subtasks

### What are Custom Actions?
Custom Actions are Go code snippets that extend agent capabilities:
- Write Go code directly in the Web UI
- No compilation required (interpreted at runtime)
- Use standard Go libraries (no external modules)
- Automatically load from `LOCALAGI_CUSTOM_ACTIONS_DIR`

### How do periodic tasks work?
Schedule tasks with cron-like syntax via `periodic_runs` configuration:
```json
{
  "periodic_runs": "0 * * * *"  // Run every hour
}
```

## Troubleshooting

### LocalAGI won't start
1. Check Docker is running: `docker ps`
2. Verify port 8080 is available: `netstat -tlnp | grep 8080`
3. Check logs: `docker compose logs`
4. Ensure sufficient memory for models

### Agents aren't responding
1. Check model is loaded in LocalAI
2. Verify `LOCALAGI_LLM_API_URL` is correct
3. Check agent status in Web UI
4. Review agent observability logs

### Knowledge base isn't working
1. Ensure agent has "Knowledge base" enabled
2. Check `LOCALAGI_STATE_DIR` is writable
3. Verify collections exist in Web UI Knowledge base section
4. Check embedding model is available

### Discord/Slack connector not working
- **Discord**: Enable "Message Content Intent" in Bot settings
- **Slack**: Verify both bot token and app token are correct
- Check connector configuration JSON syntax

### Custom actions aren't loading
1. Verify `LOCALAGI_CUSTOM_ACTIONS_DIR` is set
2. Check Go files have required functions: `Run`, `Definition`, `RequiredFields`
3. Ensure files use only standard Go libraries

### Out of memory errors
1. Use smaller models (gemma-3-4b-it-qat instead of gemma-3-27b-it)
2. Reduce `kb_results` in agent config
3. Enable `LOCALAGI_SINGLE_ACTIVE_BACKEND=true` for GPU
4. Use CPU setup for testing

## Getting Help

- **Documentation**: See Full Documentation section in README
- **YouTube Videos**: [Basic agent](https://youtu.be/HtVwIxW3ePg), [Observability](https://youtu.be/v82rswGJt_M), [Filters](https://youtu.be/d_we-AYksSw), [RAG](https://youtu.be/2Xvx78i5oBs)
- **GitHub Issues**: [Report bugs or request features](https://github.com/mudler/LocalAGI/issues)
- **Telegram Bot**: Try at [@LocalAGI_bot](https://t.me/LocalAGI_bot)
- **LocalAI**: [Model server documentation](https://github.com/mudler/LocalAI)

## Contributing

Contributions welcome! See [GitHub repository](https://github.com/mudler/LocalAGI) to:
- Report issues
- Submit pull requests
- Improve documentation
- Add new connectors or skills