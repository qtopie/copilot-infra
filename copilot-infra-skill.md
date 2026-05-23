---
name: copilot-infra
description: Offload long-running tasks (e.g., builds, tests, server starts) to the copilot-infra background worker.
---

# Background Task Execution (copilot-infra)

This skill allows the agent to offload long-running tasks (e.g., builds, tests, server starts) to the `copilot-infra` background worker.

## Integration
The service exposes a gRPC interface (default `:31415`) and is integrated with Dapr and MCP. For the Gemini agent, it is most efficient to use the provided **MCP Tools**.

## MCP Tools

### 1. Task Management
- `submit_task`: Submit a new background task.
- `get_task_status`: Check the status of a task by ID.
- `restart_task`: Restart a previously submitted task.
- `cancel_task`: Cancel a running task.

### 2. Log & Search
- `search_logs`: Search logs for a specific task with regex and context (`before`/`after` lines).
- `global_search`: Search any file or directory on the system with regex and context.

### 3. Infrastructure
- `deploy_infra`: Deploy persistent infrastructure (via Pulumi).
- `register_connection`: Register a new Dapr connection component (e.g., SurrealDB).

## Workflow Examples (Internal)

### Submit a Task
Agent calls `submit_task(command="go build ./...")`.

### Search for Errors
Agent calls `search_logs(task_id="...", pattern="ERROR", context=2)`.

## Prerequisites
- `copilot-infra` must be running.
- Start it: `./bin/copilot-infra &` (or use `-mcp` mode for direct stdio communication).
