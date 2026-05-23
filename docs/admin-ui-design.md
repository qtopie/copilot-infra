# Copilot-Infra Admin UI Design

## 1. Overview
The Admin UI provides a modern, web-based interface for managing `copilot-infra` background tasks, viewing logs, and configuring system connections. It is served directly by the `copilot-infra` binary on port `41415`.

## 2. Tech Stack
*   **Frontend Framework**: React 18 + TypeScript
*   **Build Tool**: Vite
*   **UI Components**: `@fluentui/react-components` (Fluent UI v9)
*   **Routing**: `react-router-dom` v6
*   **Backend Hosting**: Go `net/http` + `go:embed`

## 3. Architecture & API Strategy
To avoid CORS issues and simplify frontend API calls to the gRPC-based backend, the UI adopts a proxy architecture:
1.  **Frontend Build**: Vite compiles the React app into static assets in `frontend/dist`.
2.  **Go Embedding**: The Go package `pkg/ui` uses `go:embed` to compile the `dist` folder directly into the binary.
3.  **HTTP Server (Port 41415)**: 
    *   Serves the embedded static files.
    *   Implements an API proxy: All requests to `/api/*` are intercepted and reverse-proxied to the local Dapr HTTP endpoint (`http://localhost:1415`). Dapr handles the HTTP-to-gRPC translation.

## 4. Directory Structure
```text
copilot-infra/
├── frontend/                 # Frontend React application
│   ├── src/
│   │   ├── api/              # API wrapper functions
│   │   ├── components/       # Reusable UI components
│   │   ├── pages/            # Page views (Dashboard, Tasks, Connections)
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
├── pkg/ui/                   # Go host and proxy package
│   └── server.go             
└── docs/
    └── admin-ui-design.md    # This file
```

## 5. UI/UX Design

### Global Layout
*   **Top Header**: Project Logo, Theme Toggle (Light/Dark).
*   **Left Navigation (NavDrawer)**: Links to Dashboard, Tasks, and Connections.

### Pages
1.  **Dashboard**: 
    *   Overview metrics (Active Tasks, Total Tasks, Errors) using Fluent UI `Card` components.
2.  **Tasks**: 
    *   DataGrid listing tasks with status badges.
    *   Actions: Restart, Cancel.
    *   Slide-out Drawer: Details pane with a Log Viewer and search integration (leveraging the `/search` API capability).
3.  **Connections**:
    *   List of active Dapr components.
    *   Dialog form for registering new connections.
