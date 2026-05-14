package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dapr/go-sdk/client"
	"github.com/gin-gonic/gin"
	"github.com/qtopie/copilot-infra/pkg/api"
	"github.com/qtopie/copilot-infra/pkg/runtime"
	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
	"github.com/qtopie/copilot-infra/pkg/worker"
	"strconv"
)

func main() {
	apiPort := flag.Int("port", 18080, "API server port")
	daprHTTPPort := flag.Int("dapr-http-port", 3580, "Embedded Dapr HTTP port")
	daprGRPCPort := flag.Int("dapr-grpc-port", 50080, "Embedded Dapr gRPC port")
	daprAppID := flag.String("app-id", "copilot-infra", "Dapr app ID")
	logDir := flag.String("log-dir", ".logs", "Directory for task logs")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(*logDir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	// 1. Initialize Task Executor
	executor, err := task.NewExecutor(*logDir)
	if err != nil {
		log.Fatalf("failed to create executor: %v", err)
	}

	// 2. Initialize Embedded Dapr Runtime
	rt, err := runtime.NewEmbeddedRuntime(*daprAppID, *daprHTTPPort, *daprGRPCPort)
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

	// 3. Initialize State Store
	store := state.NewStore(daprClient)

	// 4. Initialize Background Worker
	w := worker.NewWorker(executor, store)
	go w.Start(ctx)

	// 5. Initialize API Server
	componentsDir := "./components"
	server := api.NewServer(store, w, *logDir, componentsDir)
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	server.RegisterRoutes(r)

	// 6. Start Server
	addr := fmt.Sprintf(":%d", *apiPort)
	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	log.Printf("Copilot-Infra is running on %s", addr)
	<-ctx.Done()
	log.Println("Shutting down...")
}

