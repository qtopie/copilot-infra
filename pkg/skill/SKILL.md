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

## CLI Usage

The `copilot-infra` binary can be used directly from the command line to manage tasks, view logs, search, and register connections by interacting with the running background daemon.

### 1. Installation
Install the skill into your agent (e.g. Gemini):
```bash
copilot-infra install
```

### 2. Task Management
* **Submit a task**:
  ```bash
  copilot-infra submit [flags] "<command>"
  ```
  *Flags:*
  * `-n`, `--name`: Deterministic name for the task
  * `-w`, `--workdir`: Work directory
  * `-t`, `--type`: Task type (`taskfile`, `infra`, or `browser`, default `taskfile`)
  * `-l`, `--long-running`: Mark as a persistent daemon/infra task
  * `-e`, `--env`: Environment variables in `KEY=VALUE` format (can be specified multiple times)

* **List all tasks**:
  ```bash
  copilot-infra list
  ```

* **View or follow task logs**:
  ```bash
  copilot-infra logs [flags] <task_id>
  ```
  *Flags:*
  * `-f`, `--follow`: Stream/follow the logs in real-time until the task completes

* **Cancel a running task**:
  ```bash
  copilot-infra cancel <task_id>
  ```

* **Restart a task**:
  ```bash
  copilot-infra restart <task_id>
  ```

### 3. Log & Code Search
Search task logs or perform a global regex search with context:
```bash
copilot-infra search [flags] "<pattern>"
```
*Flags:*
* `--task`: Search within a specific task's logs
* `--path`: Search globally within a path (default `.`)
* `-b`, `--before`: Number of lines of context before match (default 0)
* `-a`, `--after`: Number of lines of context after match (default 0)
* `-x`, `--ext`: File extension filter for global search (e.g. `.go`)

### 4. Connection Registration
Register a Dapr binding connection (e.g. SurrealDB):
```bash
copilot-infra register [flags] <name> <url>
```
*Flags:*
* `--ns`: SurrealDB namespace
* `--db`: SurrealDB database
* `--auth`: HTTP Authorization header

## Daemon Management
* **Start in background**: `copilot-infra start`
* **Stop daemon**: `copilot-infra stop`
* **Restart daemon**: `copilot-infra restart`
* **Check status**: `copilot-infra status`
* **Run in foreground**: `copilot-infra run`

