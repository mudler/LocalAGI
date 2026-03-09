# Architecture

This document describes the internal architecture of LocalAGI, covering component design, data flow, deployment patterns, and key engineering decisions.

---

## High-Level System Overview

LocalAGI is a self-hosted AI agent platform written in Go with a React frontend. It orchestrates multiple independent AI agents, each with its own connectors, actions, memory, and knowledge base. The system communicates with a local LLM inference server (LocalAI) through an OpenAI-compatible API and exposes its own REST/SSE API consumed by the Web UI and external integrations.

```mermaid
graph TB
    subgraph Clients
        WebUI["Web UI (React SPA)"]
        ExtAPI["External API Consumers"]
        Connectors["Connectors<br/>(Slack, Discord, Telegram, etc.)"]
    end

    subgraph LocalAGI["LocalAGI Server (Go)"]
        Router["Fiber HTTP Router"]
        AppLayer["Application Layer<br/>(webui package)"]
        Pool["Agent Pool"]
        Agent1["Agent 1"]
        Agent2["Agent 2"]
        AgentN["Agent N"]
        SSE["SSE Manager"]
        Scheduler["Task Scheduler"]
        Skills["Skills Service"]
        Collections["Collections / KB"]
    end

    subgraph Backends
        LLM["LocalAI<br/>(LLM Inference)"]
        Postgres["PostgreSQL<br/>(Vector Store)"]
        SSHBox["SSH Box<br/>(Shell Execution)"]
        DinD["Docker-in-Docker"]
    end

    WebUI -->|HTTP/SSE| Router
    ExtAPI -->|REST API| Router
    Router --> AppLayer
    AppLayer --> Pool
    AppLayer --> Skills
    AppLayer --> Collections
    Pool --> Agent1
    Pool --> Agent2
    Pool --> AgentN
    Agent1 --> SSE
    Agent1 --> Scheduler
    Agent1 -->|OpenAI-compatible API| LLM
    Agent1 -->|RAG queries| Collections
    Collections -->|vectors| Postgres
    Collections -->|embeddings| LLM
    Agent1 -->|shell commands| SSHBox
    SSHBox --> DinD
    Connectors -->|jobs| Pool
    SSE -->|real-time updates| WebUI
```

---

## Component Architecture

The codebase is organized into four top-level packages plus the entry point:

| Package | Purpose |
|---------|---------|
| `core/` | Agent runtime, state management, types, SSE, scheduling |
| `pkg/` | Shared utilities: LLM client, vector stores, config metadata |
| `services/` | Pluggable connectors, actions, filters, prompts, skills |
| `webui/` | HTTP API server (Fiber) and embedded React frontend |

```mermaid
graph LR
    subgraph core
        agent["core/agent<br/>Agent runtime"]
        stateP["core/state<br/>Agent Pool"]
        types["core/types<br/>Type definitions"]
        sse["core/sse<br/>Server-Sent Events"]
        sched["core/scheduler<br/>Task scheduler"]
        conv["core/conversations<br/>Conversation tracker"]
    end

    subgraph services
        actions["services/actions<br/>Built-in actions"]
        connectors["services/connectors<br/>Platform connectors"]
        filters["services/filters<br/>Job filters"]
        prompts["services/prompts<br/>Dynamic prompts"]
        skillsSvc["services/skills<br/>Skill management"]
    end

    subgraph pkg
        llm["pkg/llm<br/>LLM client"]
        vectorstore["pkg/vectorstore<br/>Vector store"]
        localrag["pkg/localrag<br/>LocalRecall client"]
        config["pkg/config<br/>Config metadata"]
    end

    subgraph webui
        app["webui/app.go<br/>Fiber application"]
        routes["webui/routes.go<br/>API routes"]
        handlers["webui/handlers<br/>Request handlers"]
        reactUI["webui/react-ui<br/>React SPA"]
    end

    app --> stateP
    app --> skillsSvc
    stateP --> agent
    agent --> types
    agent --> llm
    agent --> sse
    agent --> sched
    agent --> conv
    agent --> actions
    agent --> connectors
    agent --> filters
    agent --> prompts
    stateP --> vectorstore
    stateP --> localrag
```

---

## Data Flow

### Request Processing

Every interaction — whether from the Web UI, an API call, or a connector message — follows the same job-based pipeline:

```mermaid
sequenceDiagram
    participant C as Client
    participant R as Fiber Router
    participant A as App Layer
    participant P as Agent Pool
    participant Ag as Agent
    participant KB as Knowledge Base
    participant LLM as LocalAI (LLM)
    participant SSE as SSE Manager

    C->>R: HTTP Request (chat, action, etc.)
    R->>A: Route handler
    A->>P: GetAgent(name)
    P-->>A: Agent instance
    A->>Ag: Ask() / Execute()
    Ag->>Ag: Create Job, enqueue

    Note over Ag: Job Processing Loop
    Ag->>Ag: Apply pre-filters
    Ag->>KB: Knowledge base lookup (if enabled)
    KB-->>Ag: Relevant context
    Ag->>Ag: Render dynamic prompts
    Ag->>LLM: LLM inference (with tools)
    LLM-->>Ag: Response + tool calls

    loop Agentic Loop (tool use)
        Ag->>Ag: Execute selected action
        Ag->>LLM: Feed result back
        LLM-->>Ag: Next response
    end

    Ag->>Ag: Apply post-filters
    Ag->>Ag: Save conversation & memory
    Ag->>SSE: Broadcast update
    SSE-->>C: SSE event (real-time)
    Ag-->>A: Job result
    A-->>C: HTTP Response
```

### Connector-Initiated Flow

Connectors (Slack, Discord, Telegram, etc.) run as background goroutines. When they receive an external message, they create a `Job` and submit it to the agent's queue. Results are delivered back through the connector's callback.

```mermaid
graph LR
    Ext["External Platform<br/>(Slack, Discord, etc.)"] -->|message| Conn["Connector"]
    Conn -->|Create Job| Queue["Agent Job Queue"]
    Queue --> Loop["Agent Processing Loop"]
    Loop -->|result callback| Conn
    Conn -->|reply| Ext
```

---

## Backend Architecture

### Agent Runtime (`core/agent/`)

The `Agent` struct is the central runtime unit. Each agent:

- Maintains its own **job queue** (Go channel) processed in a dedicated goroutine
- Holds an **LLM client** (OpenAI-compatible, via the Cogito framework)
- Manages **MCP sessions** for tool discovery and execution
- Tracks **conversation history** per conversation ID
- Accesses a **knowledge base** for RAG-augmented responses
- Supports **pause/resume** with context-based cancellation

Key files:
- `core/agent/agent.go` — Main agent struct, job loop, LLM interaction
- `core/agent/options.go` — Configuration options
- `core/agent/mcp.go` — Model Context Protocol integration
- `core/agent/knowledgebase.go` — RAG knowledge base recall

### Agent Pool (`core/state/`)

The `AgentPool` manages the lifecycle of all agents:

- Creates, deletes, pauses, and starts agents
- Persists agent configurations to `pool.json`
- Provides the RAG provider (HTTP or embedded) to agents
- Tracks SSE managers for real-time client communication

### Type System (`core/types/`)

Core abstractions:

- **`Job`** — A unit of work with conversation history, available tools, callbacks, and metadata
- **`Action`** — Interface for executable tools (built-in and user-defined)
- **`JobFilter`** — Pre/post-processing hooks on jobs
- **`ConversationMessage`** — Message with role, content, and metadata
- **`AgentInternalState`** — Short-term memory: current task, next steps, history, goal

---

## Model Loading and Inference Pipeline

LocalAGI does not load models directly. It delegates all inference to **LocalAI**, an external service that handles model management, quantization, and GPU/CPU execution.

```mermaid
graph TB
    subgraph LocalAGI
        Agent["Agent"]
        LLMClient["LLM Client<br/>(OpenAI-compatible)"]
    end

    subgraph LocalAI["LocalAI Inference Server"]
        API["OpenAI-compatible API<br/>:8080"]
        ModelMgr["Model Manager"]
        Models["Loaded Models"]
        Backends["Inference Backends<br/>(llama.cpp, transformers, etc.)"]
    end

    Agent -->|tool-augmented prompt| LLMClient
    LLMClient -->|POST /v1/chat/completions| API
    API --> ModelMgr
    ModelMgr --> Models
    Models --> Backends
    Backends -->|response + tool calls| API
    API -->|JSON response| LLMClient
    LLMClient -->|parsed result| Agent
```

**Model configuration** is set per-agent:

| Model Type | Environment Variable | Purpose |
|---|---|---|
| Base model | `LOCALAGI_MODEL` | Primary text generation |
| Multimodal model | `LOCALAGI_MULTIMODAL_MODEL` | Vision and multimodal input |
| Transcription model | `LOCALAGI_TRANSCRIPTION_MODEL` | Audio-to-text |
| TTS model | `LOCALAGI_TTS_MODEL` | Text-to-speech |
| Embedding model | `EMBEDDING_MODEL` | Vector embeddings for RAG |

---

## API Layer Architecture

The API is built on the **Fiber** web framework (Go) and provides REST endpoints grouped by concern:

```mermaid
graph TB
    subgraph API["Fiber HTTP Router (:3000)"]
        direction TB
        AgentAPI["Agent Management<br/>POST /api/agent/create<br/>GET /api/agents<br/>GET /api/agent/:name<br/>DELETE /api/agent/:name<br/>POST /api/agent/:name/pause<br/>POST /api/agent/:name/start"]
        ChatAPI["Chat & Responses<br/>POST /api/chat/:name<br/>POST /v1/responses"]
        ActionAPI["Actions<br/>GET /api/actions<br/>GET /api/action/:name/definition<br/>POST /api/action/:name/run"]
        CollAPI["Collections / KB<br/>POST /api/collections/create<br/>POST /api/collections/:name/upload<br/>POST /api/collections/:name/search"]
        SkillAPI["Skills<br/>GET /api/skills<br/>POST /api/skills<br/>PUT /api/skills/:name"]
        ConfigAPI["Config & Settings<br/>GET /api/agent/:name/config<br/>GET /api/agent/config/metadata<br/>GET /settings/export/:name<br/>POST /settings/import"]
        SSEAPI["Real-time<br/>GET /sse/:name"]
        StaticUI["Static Files<br/>GET /app/*"]
    end

    AgentAPI --> Pool["Agent Pool"]
    ChatAPI --> Pool
    ActionAPI --> Pool
    CollAPI --> Collections["Collections Service"]
    SkillAPI --> Skills["Skills Service"]
    SSEAPI --> SSE["SSE Manager"]
    StaticUI --> React["Embedded React Build"]
```

**OpenAI Compatibility**: The `/v1/responses` endpoint provides a drop-in replacement for OpenAI's Responses API, enabling agents to be used by any OpenAI-compatible client.

**Authentication**: Optional API key authentication configured via `LOCALAGI_API_KEYS` (comma-separated list).

---

## Web UI Architecture

The frontend is a **React 19 SPA** built with Vite and bundled into the Go binary via `embed.FS`.

```mermaid
graph TB
    subgraph React["React SPA (webui/react-ui/)"]
        Router["React Router DOM"]
        Pages["Pages<br/>(Dashboard, Agent Config,<br/>Chat, Collections, Skills)"]
        Components["Components<br/>(Agent cards, Chat window,<br/>Action panels, Settings)"]
        Hooks["Custom Hooks<br/>(useAgent, useChat, useSSE)"]
        Contexts["Context Providers<br/>(Theme, Auth, Agent state)"]
    end

    subgraph Communication
        REST["REST API calls<br/>fetch(/api/...)"]
        SSEClient["SSE Client<br/>EventSource(/sse/:name)"]
    end

    Router --> Pages
    Pages --> Components
    Pages --> Hooks
    Hooks --> REST
    Hooks --> SSEClient
    Components --> Contexts

    REST -->|HTTP| BackendAPI["Go Fiber API"]
    SSEClient -->|streaming| BackendSSE["SSE Manager"]
```

**Build pipeline**: The React app is built with `bun` (package manager) and Vite (bundler). The compiled output is embedded into the Go binary at compile time, so the entire application ships as a single binary.

**Key UI capabilities**:
- No-code agent creation and configuration
- Real-time chat with streaming responses
- Agent status monitoring and observability
- Knowledge base management (collections, file uploads, search)
- Skill management (create, edit, import/export, git sync)
- Agent import/export

---

## P2P Networking Architecture

LocalAGI supports **agent-to-agent communication** through direct HTTP calls. Agents can invoke other agents as actions, enabling cooperative multi-agent workflows.

```mermaid
graph TB
    subgraph Instance1["LocalAGI Instance"]
        AgentA["Agent A"]
        AgentB["Agent B"]
    end

    subgraph Instance2["Remote LocalAGI Instance"]
        AgentC["Agent C"]
    end

    AgentA -->|"call-agent action<br/>(in-process)"| AgentB
    AgentA -->|"HTTP API call<br/>(cross-instance)"| AgentC
    AgentC -->|response| AgentA
```

**Agent teaming**: Multiple agents can be created from a single prompt and configured to collaborate. Each agent has a defined role and can call other agents as tools through the built-in `call-agent` action.

**MCP (Model Context Protocol)**: Agents can connect to external MCP servers (HTTP or stdio-based) to discover and use additional tools, enabling interoperability with the broader MCP ecosystem.

---

## Deployment Architecture

### Standard Docker Compose Deployment

The default deployment uses Docker Compose with five services:

```mermaid
graph TB
    subgraph DockerCompose["Docker Compose Stack"]
        LocalAGI["localagi<br/>:8080 → :3000<br/>(Go + React)"]
        LocalAI["localai<br/>:8081 → :8080<br/>(LLM Inference)"]
        Postgres["postgres<br/>:5432<br/>(Vector Store)"]
        SSHBox["sshbox<br/>:22<br/>(Shell Execution)"]
        DinD["dind<br/>:2375<br/>(Docker-in-Docker)"]
    end

    User["User / Browser"] -->|":8080"| LocalAGI
    LocalAGI -->|"OpenAI API"| LocalAI
    LocalAGI -->|"SQL + pgvector"| Postgres
    LocalAGI -->|"SSH"| SSHBox
    SSHBox -->|"Docker API"| DinD

    subgraph Volumes
        models["models"]
        pool["localagi_pool"]
        pgdata["postgres_data"]
    end

    LocalAI --- models
    LocalAGI --- pool
    Postgres --- pgdata
```

| Service | Image | Purpose |
|---------|-------|---------|
| `localagi` | Custom (Dockerfile.webui) | Main application server |
| `localai` | `localai/localai:master` | LLM inference, embeddings |
| `postgres` | `localrecall:*-postgresql` | Vector storage via pgvector |
| `sshbox` | Custom (Dockerfile.sshbox) | Sandboxed shell execution |
| `dind` | `docker:dind` | Docker-in-Docker for agent scripts |

### GPU Variants

Specialized compose files support different GPU backends:
- `docker-compose.nvidia.yaml` — NVIDIA CUDA
- `docker-compose.amd.yaml` — AMD ROCm
- `docker-compose.intel.yaml` — Intel SYCL

### Minimal Deployment

For CPU-only or external LLM setups, only the `localagi` service is required. Point `LOCALAGI_LLM_API_URL` to any OpenAI-compatible endpoint.

---

## Scalability Considerations

### Current Architecture Boundaries

LocalAGI is designed as a **single-instance, self-hosted application**. The architecture optimizes for simplicity and privacy over horizontal scaling.

```mermaid
graph TB
    subgraph SingleInstance["Single LocalAGI Instance"]
        Pool["Agent Pool<br/>(in-memory)"]
        A1["Agent 1"]
        A2["Agent 2"]
        AN["Agent N"]
        Pool --> A1
        Pool --> A2
        Pool --> AN
    end

    subgraph Bottlenecks
        LLM["LLM Inference<br/>(GPU-bound)"]
        Disk["State Persistence<br/>(disk I/O)"]
        Memory["Agent Memory<br/>(RAM)"]
    end

    A1 --> LLM
    A2 --> LLM
    AN --> LLM
    Pool --> Disk
    A1 --> Memory
```

**Vertical scaling levers**:
- Add more GPU VRAM to run larger models or more concurrent inferences
- Increase RAM for more agents and larger knowledge bases
- Use faster storage (NVMe) for state persistence and vector search

**Horizontal scaling patterns**:
- Run multiple LocalAGI instances behind a load balancer, each managing a subset of agents
- Use an external PostgreSQL instance shared across instances for knowledge base
- Point multiple instances at a shared LocalAI cluster for inference

### Agent Concurrency

Each agent processes jobs sequentially from its own queue (Go channel). Multiple agents run concurrently. The primary bottleneck is LLM inference throughput — multiple agents competing for the same LLM endpoint will queue at the inference layer.

---

## Performance Bottlenecks and Optimization

| Bottleneck | Impact | Mitigation |
|---|---|---|
| **LLM inference latency** | Dominates end-to-end response time | Use smaller/quantized models; batch inference in LocalAI; use GPU acceleration |
| **Embedding generation** | Slows knowledge base ingestion and search | Use lightweight embedding models (e.g., `granite-embedding-107m-multilingual`); pre-compute embeddings |
| **Vector search** | Scales with collection size | Use PostgreSQL with pgvector indexing; tune chunk sizes (`MAX_CHUNKING_SIZE`, `CHUNK_OVERLAP`) |
| **Agentic loops** | Multiple LLM round-trips per request | Limit tool call depth; use capable models that resolve in fewer iterations |
| **Memory usage** | Grows with number of agents and conversation history | Configure conversation duration (`LOCALAGI_CONVERSATION_DURATION`); use memory compaction |
| **State persistence** | Disk I/O on pool/agent save | Use fast local storage; state writes are infrequent |

**Optimization tips**:
- Start with smaller models (e.g., `gemma-3-4b-it-qat`) and scale up as needed
- Enable knowledge base only on agents that need it
- Tune `LOCALAGI_TIMEOUT` to fail fast on stuck inferences
- Use the in-process vector engine (`chromem`) to avoid PostgreSQL overhead for small deployments

---

## Security Architecture

```mermaid
graph TB
    subgraph External["External Boundary"]
        User["User / Browser"]
        ExtPlatform["External Platforms"]
    end

    subgraph LocalAGI["LocalAGI (Trust Boundary)"]
        APIAuth["API Key Auth<br/>(optional)"]
        Router["Fiber Router"]
        Agents["Agent Pool"]
    end

    subgraph Sandboxed["Sandboxed Execution"]
        SSHBox["SSH Box Container"]
        DinD["Docker-in-Docker"]
    end

    subgraph LocalNetwork["Local Network Only"]
        LLM["LocalAI"]
        DB["PostgreSQL"]
    end

    User -->|"API keys"| APIAuth
    APIAuth --> Router
    Router --> Agents
    Agents -->|"shell actions"| SSHBox
    SSHBox --> DinD
    Agents --> LLM
    Agents --> DB
    ExtPlatform <-->|"connector tokens"| Agents
```

### Key Security Properties

**Data sovereignty**: All data stays on the user's hardware. No external API calls are required for core functionality. LLM inference, embeddings, and vector search all run locally.

**API authentication**: Optional API key authentication protects the REST API. Keys are configured via `LOCALAGI_API_KEYS`.

**Sandboxed execution**: Shell commands run inside an isolated SSH container (`sshbox`), which connects to a Docker-in-Docker instance. This prevents agent-executed code from accessing the host system.

**Connector credentials**: Platform tokens (Slack, Discord, Telegram, etc.) are stored in agent configuration files within the state directory. Protect the state directory with appropriate file permissions.

### Security Recommendations

- **Always enable API keys** in production deployments
- **Restrict network access** — LocalAI and PostgreSQL should not be exposed to the public internet
- **Protect the state directory** — it contains agent configs, credentials, and conversation history
- **Review custom actions** — user-defined Go actions execute within the LocalAGI process
- **Use Docker isolation** — the SSH box and DinD containers limit blast radius of agent-executed code
- **Keep services updated** — regularly update LocalAI and PostgreSQL for security patches
