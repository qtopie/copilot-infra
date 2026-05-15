package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	taskv1 "github.com/qtopie/copilot-infra/pkg/api/proto/v1"
)

type MCPHandler struct {
	grpcHandler *GRPCHandler
}

func NewMCPHandler(grpcHandler *GRPCHandler) *MCPHandler {
	return &MCPHandler{grpcHandler: grpcHandler}
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func (h *MCPHandler) ServeStdio() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("MCP Error: %v", err)
			continue
		}

		resp := h.handleRequest(req)
		if err := encoder.Encode(resp); err != nil {
			log.Printf("MCP Encode Error: %v", err)
		}
	}
}

func (h *MCPHandler) handleRequest(req JSONRPCRequest) JSONRPCResponse {
	ctx := context.Background()
	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"serverInfo": map[string]string{
				"name":    "copilot-infra-mcp",
				"version": "0.1.0",
			},
		}
	case "tools/list":
		result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "submit_task",
					"description": "Submit a new background task (shell or Taskfile)",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"command": map[string]string{"type": "string", "description": "The command to run"},
						},
						"required": []string{"command"},
					},
				},
				{
					"name":        "deploy_infra",
					"description": "Deploy persistent infrastructure (via Pulumi)",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"project": map[string]string{"type": "string", "description": "Pulumi project name"},
							"stack":   map[string]string{"type": "string", "description": "Stack name"},
						},
						"required": []string{"project", "stack"},
					},
				},
				{
					"name":        "get_task_status",
					"description": "Get the status of a task",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"task_id": map[string]string{"type": "string"},
						},
						"required": []string{"task_id"},
					},
				},
				{
					"name":        "search_logs",
					"description": "Search logs for a specific task with regex and context",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"task_id": map[string]string{"type": "string"},
							"pattern": map[string]string{"type": "string", "description": "Regex pattern"},
							"before":  map[string]interface{}{"type": "integer", "description": "Lines before match"},
							"after":   map[string]interface{}{"type": "integer", "description": "Lines after match"},
						},
						"required": []string{"task_id", "pattern"},
					},
				},
				{
					"name":        "global_search",
					"description": "Search any file or directory with regex and context",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":    map[string]string{"type": "string", "description": "File or directory path"},
							"pattern": map[string]string{"type": "string", "description": "Regex pattern"},
							"ext":     map[string]string{"type": "string", "description": "Filter by extension (e.g. .log)"},
							"before":  map[string]interface{}{"type": "integer", "description": "Lines before match"},
							"after":   map[string]interface{}{"type": "integer", "description": "Lines after match"},
						},
						"required": []string{"path", "pattern"},
					},
				},
				{
					"name":        "restart_task",
					"description": "Restart a background task",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"task_id": map[string]string{"type": "string"},
						},
						"required": []string{"task_id"},
					},
				},
				{
					"name":        "cancel_task",
					"description": "Cancel a running task",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"task_id": map[string]string{"type": "string"},
						},
						"required": []string{"task_id"},
					},
				},
				{
					"name":        "register_connection",
					"description": "Register a new Dapr connection component (e.g. SurrealDB)",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]string{"type": "string", "description": "Connection name"},
							"url":  map[string]string{"type": "string", "description": "Connection URL"},
							"ns":   map[string]string{"type": "string", "description": "Namespace"},
							"db":   map[string]string{"type": "string", "description": "Database"},
							"auth": map[string]string{"type": "string", "description": "Auth header"},
						},
						"required": []string{"name", "url"},
					},
				},
			},
		}
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		json.Unmarshal(req.Params, &params)

		switch params.Name {
		case "submit_task":
			var args struct {
				Command string `json:"command"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.SubmitTask(ctx, &taskv1.SubmitTaskRequest{Command: args.Command})
			result = res
			err = e
		case "deploy_infra":
			var args struct {
				Project string `json:"project"`
				Stack   string `json:"stack"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.SubmitTask(ctx, &taskv1.SubmitTaskRequest{
				Type:    "infra",
				Project: args.Project,
				Name:    args.Stack,
			})
			result = res
			err = e
		case "get_task_status":
			var args struct {
				TaskID string `json:"task_id"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.GetTask(ctx, &taskv1.GetTaskRequest{TaskId: args.TaskID})
			result = res
			err = e
		case "search_logs":
			var args struct {
				TaskID  string `json:"task_id"`
				Pattern string `json:"pattern"`
				Before  int    `json:"before"`
				After   int    `json:"after"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.SearchLogs(ctx, &taskv1.SearchLogsRequest{
				TaskId:  args.TaskID,
				Pattern: args.Pattern,
				Before:  int32(args.Before),
				After:   int32(args.After),
			})
			result = res
			err = e
		case "global_search":
			var args struct {
				Path    string `json:"path"`
				Pattern string `json:"pattern"`
				Ext     string `json:"ext"`
				Before  int    `json:"before"`
				After   int    `json:"after"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.GlobalSearch(ctx, &taskv1.GlobalSearchRequest{
				Path:    args.Path,
				Pattern: args.Pattern,
				Ext:     args.Ext,
				Before:  int32(args.Before),
				After:   int32(args.After),
			})
			result = res
			err = e
		case "restart_task":
			var args struct {
				TaskID string `json:"task_id"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.RestartTask(ctx, &taskv1.RestartTaskRequest{TaskId: args.TaskID})
			result = res
			err = e
		case "cancel_task":
			var args struct {
				TaskID string `json:"task_id"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.CancelTask(ctx, &taskv1.CancelTaskRequest{TaskId: args.TaskID})
			result = res
			err = e
		case "register_connection":
			var args struct {
				Name string `json:"name"`
				URL  string `json:"url"`
				NS   string `json:"ns"`
				DB   string `json:"db"`
				Auth string `json:"auth"`
			}
			json.Unmarshal(params.Arguments, &args)
			res, e := h.grpcHandler.RegisterConnection(ctx, &taskv1.RegisterConnectionRequest{
				Name: args.Name,
				Url:  args.URL,
				Ns:   args.NS,
				Db:   args.DB,
				Auth: args.Auth,
			})
			result = res
			err = e
		default:
			err = fmt.Errorf("unknown tool: %s", params.Name)
		}
	default:
		err = fmt.Errorf("method not found: %s", req.Method)
	}

	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: map[string]interface{}{
				"code":    -32603,
				"message": err.Error(),
			},
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}
