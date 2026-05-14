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

## Data Flow
1. AI Agent sends a `POST /tasks` request with a command or task name.
2. API Layer generates a unique Task ID and saves the "Pending" state to Dapr.
3. The request is queued for the Background Worker.
4. Background Worker picks up the task, updates state to "Running", and starts the Task Executor.
5. Task Executor captures output to a log file.
6. Upon completion, Worker updates state to "Success" or "Failure".
7. AI Agent polls `GET /tasks/{id}` and eventually retrieves logs to verify functionality.
