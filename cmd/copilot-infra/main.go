package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dapr/go-sdk/client"
	taskv1 "github.com/qtopie/copilot-infra/pkg/api/proto/v1"
	"github.com/qtopie/copilot-infra/pkg/api"
	"github.com/qtopie/copilot-infra/pkg/infra"
	"github.com/qtopie/copilot-infra/pkg/runtime"
	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
	"github.com/qtopie/copilot-infra/pkg/worker"
	"google.golang.org/grpc"
	"strconv"
)

func main() {
	grpcPort := flag.Int("grpc-port", 31415, "App gRPC server port")
	daprHTTPPort := flag.Int("dapr-http-port", 1415, "Embedded Dapr HTTP port")
	daprGRPCPort := flag.Int("dapr-grpc-port", 51415, "Embedded Dapr gRPC port")
	daprAppID := flag.String("app-id", "copilot-infra", "Dapr app ID")
	logDir := flag.String("log-dir", ".logs", "Directory for task logs")
	infraDir := flag.String("infra-dir", ".infra", "Directory for Pulumi infrastructure state")
	mcpMode := flag.Bool("mcp", false, "Run in MCP mode (stdio)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(*logDir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	// 1. Initialize Infra Manager
	infraManager, err := infra.NewManager(*infraDir)
	if err != nil {
		log.Fatalf("failed to create infra manager: %v", err)
	}

	// 2. Initialize Task Executor
	executor, err := task.NewExecutor(*logDir, infraManager)
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
	grpcHandler := api.NewGRPCHandler(w, store, *logDir, componentsDir)
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

	log.Printf("Copilot-Infra is running on :%d (gRPC)", *grpcPort)
	log.Printf("Dapr HTTP proxy is available on :%d", *daprHTTPPort)
	<-ctx.Done()
	log.Println("Shutting down...")
}

