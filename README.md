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

Default Ports:
- **gRPC API**: `31415`
- **Dapr HTTP Proxy**: `1415`
- **Dapr gRPC Proxy**: `51415`

## Project structure

- `api/proto/v1/` - Protobuf definitions
- `cmd/copilot-infra/` - Application entrypoint
- `pkg/api/` - gRPC and MCP handlers
- `pkg/runtime/` - Runtime orchestration logic (Dapr)
- `pkg/state/` - state storage abstractions
- `pkg/task/` - task execution engine and types
- `pkg/worker/` - background worker implementation

## Notes

- A `Taskfile.yml` is included for task automation.
- Configure environment variables or runtime settings as needed.
