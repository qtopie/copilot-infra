# Architecture Design: Copilot-Infra

## Overview
`copilot-infra` is an AI agent coding infrastructure designed to decouple long-running tasks (like builds and tests) from the agent's main execution loop. It provides a background execution environment where tasks can be submitted, monitored, and analyzed.

## Key Technologies
- **Language**: Go
- **Orchestration & State**: Embedded Dapr Runtime
- **Task Execution**: `go-task` (programmatic library)
- **Infrastructure**: Pulumi Automation API
- **API**: HTTP/JSON (Gin or Echo)

## Components

### 1. Embedded Dapr Runtime
Instead of running Dapr as a separate sidecar, we embed it directly into the `copilot-infra` binary.
- **State Store**: Uses an embedded SQLite component to persist task metadata and status.
- **Pub/Sub**: Used for internal task queuing and distribution.

### 2. Task Executor
A wrapper around the `go-task` library.
- **Taskfile Support**: Can execute tasks defined in a `Taskfile.yml`.
- **Command Execution**: Supports running arbitrary shell commands.
- **Logging**: Each task execution captures stdout/stderr to a unique log file in `.logs/<task_id>.log`.

### 3. Background Worker
A goroutine-based worker that:
- Listens for new task submissions.
- Manages the lifecycle of a task (Pending -> Running -> Success/Failure).
- Updates the Dapr state store with the latest progress.

### 4. API Layer
Provides an interface for AI agents to interact with the infrastructure.
- `POST /tasks`: Submit a task.
- `GET /tasks/{id}`: Poll status.
- `GET /tasks/{id}/logs`: Stream or retrieve logs.

## Multi-Protocol Access (REST, gRPC, MCP)
To maximize flexibility and integration with both traditional clients and AI agents, `copilot-infra` exposes its services through three primary protocols simultaneously, leveraging Dapr as the central protocol gateway.

### 1. gRPC (Core)
The business logic is implemented natively as a gRPC service. gRPC provides high performance, strong typing through Protobuf, and efficient streaming for long-running task updates.
- **Contract**: Defined in `.proto` files.
- **Implementation**: Go gRPC server integrated with Dapr.

### 2. HTTP REST (via Dapr Transcoding)
Dapr's built-in protocol transcoding allows the gRPC service to be accessed via standard RESTful calls without writing additional boilerplate code.
- **Mapping**: Dapr automatically maps HTTP POST requests to gRPC methods.
- **Endpoint**: `http://localhost:<dapr-http-port>/v1.0/invoke/copilot-infra/method/<method_name>`
- **Benefit**: Seamless integration with web dashboards and simple CLI tools.

### 3. MCP (Model Context Protocol)
For deep integration with AI agents (like Claude or Gemini), the service is exposed as an MCP Server.
- **Mechanism**: A lightweight MCP adapter (or Dapr AI component) translates gRPC method definitions into MCP "Tools".
- **Dynamic Discovery**: The adapter reads the service metadata to dynamically generate JSON Schema descriptions for the AI.
- **AI-Native**: Allows AI agents to "see" and "call" the infrastructure's capabilities as native tools.

## Infrastructure Persistence (via Pulumi)
A key requirement for `copilot-infra` is that managed infrastructure (like databases, web servers, or cloud resources) must outlive the `copilot-infra` process itself.

### 1. Desired State Management
We use the **Pulumi Automation API** to achieve this. Unlike raw shell commands that might leave orphaned processes, Pulumi manages the full lifecycle of resources.
- **Independence**: Resources created by Pulumi providers (Docker, Kubernetes, AWS, etc.) are managed by those platforms. Pulumi only tracks the "desired state" in a persistent stack.
- **Durability**: If `copilot-infra` crashes or is restarted, the infrastructure remains running.

### 2. Recovery Workflow
1. **Submit**: Agent submits an "Infra Task".
2. **Execute**: `copilot-infra` runs `pulumi up` using the Automation API.
3. **Persist**: The resource state is saved in the local `.infra` directory (or a remote Pulumi backend).
4. **Restart**: Upon restart, `copilot-infra` can query existing stacks to re-discover and manage resources.

### 3. Example Use Case
- **Agent Action**: "Deploy an Nginx service."
- **Infra Task**: Created with `Type: infra`, `Project: web-svc`, `Stack: prod`.
- **Result**: An Nginx container is started. If `copilot-infra` stops, Nginx continues to serve traffic. When `copilot-infra` comes back, the Agent can still check the status of the "web-svc" stack.
