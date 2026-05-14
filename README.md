# Copilot Infra

A Go-based infrastructure utility for managing Copilot-related runtime and task workflows.

## Overview

This repository contains a command-line Go application with support for task execution, state storage, and runtime orchestration.

## Requirements

- Go 1.26+
- Git

## Build

```bash
go build -o bin/copilot-infra ./cmd/copilot-infra
```

## Run

```bash
./bin/copilot-infra
```

## Project structure

- `cmd/copilot-infra/` - application entrypoint
- `pkg/api/` - HTTP/API handlers
- `pkg/runtime/` - runtime orchestration logic
- `pkg/state/` - state storage abstractions
- `pkg/task/` - task execution engine and types
- `pkg/worker/` - background worker implementation

## Notes

- A `Taskfile.yml` is included for task automation.
- Configure environment variables or runtime settings as needed.
