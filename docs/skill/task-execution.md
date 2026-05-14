# Skill: Background Task Execution (copilot-infra)

This skill allows the agent to offload long-running tasks (e.g., builds, tests, server starts) to the `copilot-infra` background worker.

## When to use
- Running `gowails` or other GUI apps that take time to build/start.
- Running long test suites.
- Deploying infrastructure that shouldn't block the main agent loop.
- Any task where you need to poll for status and check logs later.

## Prerequisites
- `copilot-infra` must be running (default port `:8080`).
- If not running, start it: `./bin/copilot-infra &`.

## Workflow

### 1. Submit a Task
Use `curl` or a Go client to submit a command or Taskfile task.

```bash
# Example: Submit a shell command
curl -X POST http://localhost:8080/tasks \
     -H "Content-Type: application/json" \
     -d '{"cmd": "go build ./..."}'

# Example: Submit a Taskfile task
curl -X POST http://localhost:8080/tasks \
     -H "Content-Type: application/json" \
     -d '{"name": "test"}'
```

### 2. Monitor Status
Poll the task status until it reaches `succeeded` or `failed`.

```bash
curl http://localhost:8080/tasks/<task_id>
```

### 3. Retrieve Logs
Check the console output for debugging or verification.

```bash
curl http://localhost:8080/tasks/<task_id>/logs
```

## Integration Hint
When writing code for an agent, use the `SubmitTaskRequest` and `Task` structs from `github.com/qtopie/copilot-infra/pkg/task` to interact with the API programmatically.
