package main

import (
	"context"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logDir := ".logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	// 1. Initialize Task Executor
	executor, err := task.NewExecutor(logDir)
	if err != nil {
		log.Fatalf("failed to create executor: %v", err)
	}

	// 2. Initialize Embedded Dapr Runtime
	daprHTTPPort := 3500
	daprGRPCPort := 50001
	daprAppID := "copilot-infra"

	rt, err := runtime.NewEmbeddedRuntime(daprAppID, daprHTTPPort, daprGRPCPort)
	if err != nil {
		log.Fatalf("failed to create embedded dapr runtime: %v", err)
	}
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("failed to start embedded dapr runtime: %v", err)
	}
	defer rt.Stop()

	// 3. Initialize Dapr Client
	daprClient, err := client.NewClientWithPort(strconv.Itoa(daprGRPCPort))
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
	server := api.NewServer(store, w, logDir)
	r := gin.Default()
	server.RegisterRoutes(r)

	// 6. Start Server
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	log.Println("Copilot-Infra is running on :8080")
	<-ctx.Done()
	log.Println("Shutting down...")
}
