package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/dapr/go-sdk/client"
	"github.com/sevlyar/go-daemon"
	taskv1 "github.com/qtopie/copilot-infra/pkg/api/proto/v1"
	"github.com/qtopie/copilot-infra/pkg/api"
	"github.com/qtopie/copilot-infra/pkg/infra"
	"github.com/qtopie/copilot-infra/pkg/runtime"
	"github.com/qtopie/copilot-infra/pkg/setup"
	"github.com/qtopie/copilot-infra/pkg/skill"
	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
	"github.com/qtopie/copilot-infra/pkg/worker"
	"github.com/qtopie/copilot-infra/pkg/ui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strconv"
	"time"
)

var (
	uiPort         = flag.Int("ui-port", 41415, "Admin UI server port")
	grpcPort       = flag.Int("grpc-port", 31415, "App gRPC server port")
	daprHTTPPort   = flag.Int("dapr-http-port", 1415, "Embedded Dapr HTTP port")
	daprGRPCPort   = flag.Int("dapr-grpc-port", 51415, "Embedded Dapr gRPC port")
	daprAppID      = flag.String("app-id", "copilot-infra", "Dapr app ID")
	logDir         = flag.String("log-dir", ".logs", "Directory for task logs")
	infraDir       = flag.String("infra-dir", ".infra", "Directory for Pulumi infrastructure state")
	mcpMode        = flag.Bool("mcp", false, "Run in MCP mode (stdio)")
	pidFile        = flag.String("pid-file", "copilot-infra.pid", "PID file path")
	daemonLogFile  = flag.String("daemon-log", "copilot-infra.log", "Daemon log file path")
)

func main() {
	flag.Parse()
	command := flag.Arg(0)

	if command == "" {
		command = "run"
	}

	switch command {
	case "start":
		startDaemon()
	case "stop":
		stopDaemon()
	case "status":
		statusDaemon()
	case "restart":
		if len(flag.Args()) >= 2 {
			runRestart(flag.Arg(1))
		} else {
			stopDaemon()
			time.Sleep(1 * time.Second)
			startDaemon()
		}
	case "install", "install-skill":
		if err := skill.InstallSkill(); err != nil {
			fmt.Printf("Failed to install skill: %v\n", err)
			os.Exit(1)
		}
	case "run":
		runServer()
	case "submit":
		submitCmd := flag.NewFlagSet("submit", flag.ExitOnError)
		name := submitCmd.String("name", "", "Deterministic task name")
		project := submitCmd.String("project", "", "Pulumi project name")
		workdir := submitCmd.String("workdir", "", "Work directory")
		taskType := submitCmd.String("type", "taskfile", "Task type (taskfile, infra, browser)")
		longRunning := submitCmd.Bool("long-running", false, "Is a long-running task")
		
		var envs arrayFlags
		submitCmd.Var(&envs, "env", "Environment variable in KEY=VALUE format (can be specified multiple times)")
		
		submitCmd.Parse(flag.Args()[1:])
		cmdArg := submitCmd.Arg(0)
		if cmdArg == "" && *taskType != "infra" {
			fmt.Println("Error: command is required for non-infra tasks")
			fmt.Println("Usage: copilot-infra submit [flags] \"<command>\"")
			os.Exit(1)
		}
		
		runSubmit(cmdArg, *name, *project, *workdir, *taskType, *longRunning, envs)
	case "list":
		runList()
	case "logs":
		logsCmd := flag.NewFlagSet("logs", flag.ExitOnError)
		follow := logsCmd.Bool("follow", false, "Stream/follow the logs")
		
		logsCmd.Parse(flag.Args()[1:])
		taskID := logsCmd.Arg(0)
		if taskID == "" {
			fmt.Println("Error: task ID is required")
			fmt.Println("Usage: copilot-infra logs [flags] <task_id>")
			os.Exit(1)
		}
		
		runLogs(taskID, *follow)
	case "cancel":
		if len(flag.Args()) < 2 {
			fmt.Println("Error: task ID is required")
			fmt.Println("Usage: copilot-infra cancel <task_id>")
			os.Exit(1)
		}
		runCancel(flag.Arg(1))
	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		taskID := searchCmd.String("task", "", "Search logs for specific task")
		path := searchCmd.String("path", ".", "Path to perform global search")
		ext := searchCmd.String("ext", "", "File extension filter for global search")
		before := searchCmd.Int("before", 0, "Lines of context before match")
		after := searchCmd.Int("after", 0, "Lines of context after match")
		
		searchCmd.Parse(flag.Args()[1:])
		pattern := searchCmd.Arg(0)
		if pattern == "" {
			fmt.Println("Error: search pattern is required")
			fmt.Println("Usage: copilot-infra search [flags] \"<pattern>\"")
			os.Exit(1)
		}
		
		runSearch(pattern, *taskID, *path, *ext, *before, *after)
	case "register":
		registerCmd := flag.NewFlagSet("register", flag.ExitOnError)
		ns := registerCmd.String("ns", "", "SurrealDB namespace")
		db := registerCmd.String("db", "", "SurrealDB database")
		auth := registerCmd.String("auth", "", "HTTP Authorization header")
		
		registerCmd.Parse(flag.Args()[1:])
		if registerCmd.NArg() < 2 {
			fmt.Println("Error: connection name and URL are required")
			fmt.Println("Usage: copilot-infra register [flags] <name> <url>")
			os.Exit(1)
		}
		
		runRegister(registerCmd.Arg(0), registerCmd.Arg(1), *ns, *db, *auth)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: copilot-infra [start|stop|restart|status|install|submit|list|logs|cancel|search|register|run] [flags]")
		os.Exit(1)
	}
}

func isProcessAlive(p *os.Process) bool {
	if p == nil {
		return false
	}
	err := p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return false
}

func startDaemon() {
	cntxt := &daemon.Context{
		PidFileName: *pidFile,
		PidFilePerm: 0644,
		LogFileName: *daemonLogFile,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
	}

	// Check if already running
	if d, err := cntxt.Search(); err == nil {
		if isProcessAlive(d) {
			fmt.Printf("Copilot-Infra is already running (PID: %d)\n", d.Pid)
			return
		}
		// Process is dead but PID file exists
		_ = os.Remove(*pidFile)
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		fmt.Printf("Copilot-Infra started in background (PID: %d)\n", d.Pid)
		fmt.Printf("Admin UI: http://localhost:%d\n", *uiPort)
		return
	}
	defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("Daemon started")

	runServer()
}

func stopDaemon() {
	cntxt := &daemon.Context{
		PidFileName: *pidFile,
	}

	d, err := cntxt.Search()
	if err != nil || !isProcessAlive(d) {
		fmt.Printf("Copilot-Infra is not running\n")
		if err == nil {
			_ = os.Remove(*pidFile)
		}
		return
	}

	if err := d.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("Unable to stop Copilot-Infra (PID %d): %v\n", d.Pid, err)
		return
	}

	fmt.Printf("Copilot-Infra (PID %d) stopping...\n", d.Pid)
	// Wait a bit for it to clean up
	for i := 0; i < 5; i++ {
		time.Sleep(500 * time.Millisecond)
		if d, err := cntxt.Search(); err != nil || !isProcessAlive(d) {
			fmt.Println("Stopped.")
			return
		}
	}
}

func statusDaemon() {
	cntxt := &daemon.Context{
		PidFileName: *pidFile,
	}

	d, err := cntxt.Search()
	if err != nil || !isProcessAlive(d) {
		fmt.Println("Copilot-Infra is not running")
		return
	}

	fmt.Printf("Copilot-Infra is running (PID: %d)\n", d.Pid)
	fmt.Printf("Admin UI: http://localhost:%d\n", *uiPort)
	
	// Try to fetch recent tasks from the local UI API proxy
	url := fmt.Sprintf("http://localhost:%d/api/task.v1.TaskService/ListTasks", *uiPort)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("Could not connect to API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API returned status %d\n", resp.StatusCode)
		return
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Tasks []struct {
			TaskId    string `json:"taskId"`
			Status    string `json:"status"`
			CreatedAt string `json:"createdAt"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse tasks: %v\n", err)
		return
	}

	tasks := result.Tasks
	if len(tasks) == 0 {
		fmt.Println("\nNo recent tasks found.")
		return
	}

	fmt.Printf("\nRecent Tasks (up to 100):\n")
	fmt.Printf("%-36s %-12s %-24s\n", "TASK ID", "STATUS", "CREATED AT")
	fmt.Println(strings.Repeat("-", 74))

	// Get last 100 max
	count := len(tasks)
	if count > 100 {
		tasks = tasks[count-100:]
	}

	for _, t := range tasks {
		fmt.Printf("%-36s %-12s %-24s\n", t.TaskId, t.Status, t.CreatedAt)
	}
}

func runServer() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer os.Remove(*pidFile)

	// Ensure Pulumi CLI is available or install it in the background
	setup.EnsurePulumiBackground()

	absLogDir, _ := filepath.Abs(*logDir)
	if err := os.MkdirAll(absLogDir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	// 1. Initialize Infra Manager
	absInfraDir, _ := filepath.Abs(*infraDir)
	infraManager, err := infra.NewManager(absInfraDir)
	if err != nil {
		log.Fatalf("failed to create infra manager: %v", err)
	}

	// 2. Initialize Task Executor
	executor, err := task.NewExecutor(absLogDir, infraManager)
	if err != nil {
		log.Fatalf("failed to create executor: %v", err)
	}

	// 2. Initialize Embedded Dapr Runtime
	rt, err := runtime.NewEmbeddedRuntime(*daprAppID, *daprHTTPPort, *daprGRPCPort, *grpcPort, "grpc")
	if err != nil {
		log.Fatalf("failed to create embedded dapr runtime: %v", err)
	}
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("failed to start embedded dapr runtime: %v", err)
	}
	defer rt.Stop()

	// 3. Initialize Dapr Client
	daprClient, err := client.NewClientWithPort(strconv.Itoa(*daprGRPCPort))
	if err != nil {
		log.Fatalf("failed to connect to dapr runtime: %v", err)
	}
	defer daprClient.Close()

	// 4. Initialize State Store
	store := state.NewStore(daprClient)

	// 5. Initialize Background Worker
	w := worker.NewWorker(executor, store)
	go w.Start(ctx)

	// 6. Start gRPC Server
	componentsDir := "./components"
	grpcHandler := api.NewGRPCHandler(w, store, absLogDir, componentsDir)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}
	s := grpc.NewServer()
	taskv1.RegisterTaskServiceServer(s, grpcHandler)
	go func() {
		if !*mcpMode {
			log.Printf("gRPC server listening on :%d", *grpcPort)
		}
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 7. MCP Mode or HTTP Mode
	if *mcpMode {
		mcpHandler := api.NewMCPHandler(grpcHandler)
		mcpHandler.ServeStdio()
		return
	}

	// 8. Start Admin UI Server
	uiServer := ui.NewServer(*uiPort, *grpcPort)
	go func() {
		if err := uiServer.Start(); err != nil {
			log.Fatalf("failed to start UI server: %v", err)
		}
	}()

	if !*mcpMode {
		log.Printf("Copilot-Infra is running on :%d (gRPC) and :%d (UI)", *grpcPort, *uiPort)
		log.Printf("Dapr HTTP proxy is available on :%d", *daprHTTPPort)
	}
	
	<-ctx.Done()
	log.Println("Shutting down...")
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func getGRPCClient() (taskv1.TaskServiceClient, *grpc.ClientConn, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", *grpcPort)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", addr, err)
	}
	return taskv1.NewTaskServiceClient(conn), conn, nil
}

func runSubmit(cmdStr, name, project, workdir, taskType string, longRunning bool, envs []string) {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	envMap := make(map<string, string>)
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		} else {
			fmt.Printf("Warning: invalid env format %q, should be KEY=VALUE\n", env)
		}
	}

	req := &taskv1.SubmitTaskRequest{
		Command:       cmdStr,
		Env:           envMap,
		WorkDir:       workdir,
		Type:          taskType,
		Project:       project,
		Name:          name,
		IsLongRunning: longRunning,
	}

	resp, err := client.SubmitTask(context.Background(), req)
	if err != nil {
		fmt.Printf("Error submitting task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task submitted successfully!\n")
	fmt.Printf("Task ID: %s\n", resp.TaskId)
	if resp.AccessUrl != "" {
		fmt.Printf("Access URL: %s\n", resp.AccessUrl)
	}
}

func runList() {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	resp, err := client.ListTasks(context.Background(), &taskv1.ListTasksRequest{})
	if err != nil {
		fmt.Printf("Error listing tasks: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	fmt.Printf("%-36s %-16s %-12s %-24s %s\n", "TASK ID", "NAME", "STATUS", "CREATED AT", "INFO")
	fmt.Println(strings.Repeat("-", 100))
	for _, t := range resp.Tasks {
		info := ""
		if t.Error != "" {
			info = "Error: " + t.Error
		} else if t.AccessUrl != "" {
			info = "URL: " + t.AccessUrl
		}
		name := t.Name
		if name == "" {
			name = "-"
		}
		fmt.Printf("%-36s %-16s %-12s %-24s %s\n", t.TaskId, name, t.Status, t.CreatedAt, info)
	}
}

func runLogs(taskID string, follow bool) {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	resp, err := client.GetTaskLogs(ctx, &taskv1.GetTaskLogsRequest{TaskId: taskID})
	if err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(resp.Content)
	lastOffset := len(resp.Content)

	if !follow {
		return
	}

	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	fmt.Println("\n--- Streaming logs (Ctrl+C to exit) ---")
	for {
		select {
		case <-ticker.C:
			taskInfo, err := client.GetTask(ctx, &taskv1.GetTaskRequest{TaskId: taskID})
			if err != nil {
				fmt.Printf("\nError getting task status: %v\n", err)
				return
			}

			resp, err = client.GetTaskLogs(ctx, &taskv1.GetTaskLogsRequest{TaskId: taskID})
			if err != nil {
				fmt.Printf("\nError fetching logs: %v\n", err)
				return
			}

			if len(resp.Content) > lastOffset {
				fmt.Print(resp.Content[lastOffset:])
				lastOffset = len(resp.Content)
			}

			if taskInfo.Status != "pending" && taskInfo.Status != "running" {
				fmt.Printf("\n--- Task finished with status: %s ---\n", taskInfo.Status)
				return
			}
		}
	}
}

func runCancel(taskID string) {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	resp, err := client.CancelTask(context.Background(), &taskv1.CancelTaskRequest{TaskId: taskID})
	if err != nil {
		fmt.Printf("Error canceling task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Success: %s\n", resp.Message)
}

func runRestart(taskID string) {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	resp, err := client.RestartTask(context.Background(), &taskv1.RestartTaskRequest{TaskId: taskID})
	if err != nil {
		fmt.Printf("Error restarting task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task %s restarted successfully!\n", resp.TaskId)
}

func runSearch(pattern, taskID, path, ext string, before, after int) {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	if taskID != "" {
		resp, err := client.SearchLogs(ctx, &taskv1.SearchLogsRequest{
			TaskId:  taskID,
			Pattern: pattern,
			Before:  int32(before),
			After:   int32(after),
		})
		if err != nil {
			fmt.Printf("Error searching logs: %v\n", err)
			os.Exit(1)
		}
		printSearchResults(resp.Matches)
	} else {
		if path == "" {
			path = "."
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("Invalid path %q: %v\n", path, err)
			os.Exit(1)
		}
		resp, err := client.GlobalSearch(ctx, &taskv1.GlobalSearchRequest{
			Path:    absPath,
			Pattern: pattern,
			Ext:     ext,
			Before:  int32(before),
			After:   int32(after),
		})
		if err != nil {
			fmt.Printf("Error performing global search: %v\n", err)
			os.Exit(1)
		}
		printSearchResults(resp.Matches)
	}
}

func printSearchResults(matches []*taskv1.SearchMatch) {
	if len(matches) == 0 {
		fmt.Println("No matches found.")
		return
	}

	fmt.Printf("Found %d matches:\n\n", len(matches))
	for _, m := range matches {
		fmt.Printf("\033[35m%s:%d\033[0m\n", m.Path, m.LineNum)
		for _, line := range m.ContextBefore {
			fmt.Printf("  - %s\n", line)
		}
		fmt.Printf("\033[31m  > %s\033[0m\n", m.Text)
		for _, line := range m.ContextAfter {
			fmt.Printf("  + %s\n", line)
		}
		fmt.Println()
	}
}

func runRegister(name, url, ns, db, auth string) {
	client, conn, err := getGRPCClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	resp, err := client.RegisterConnection(context.Background(), &taskv1.RegisterConnectionRequest{
		Name: name,
		Url:  url,
		Ns:   ns,
		Db:   db,
		Auth: auth,
	})
	if err != nil {
		fmt.Printf("Error registering connection: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Success: %s\n", resp.Message)
	fmt.Printf("Component Name: %s\n", resp.Name)
	fmt.Printf("Dapr Path: %s\n", resp.DaprPath)
}

