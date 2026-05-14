package runtime

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	httpBinding "github.com/dapr/components-contrib/bindings/http"
	mdnsnr "github.com/dapr/components-contrib/nameresolution/mdns"
	sqlitestate "github.com/dapr/components-contrib/state/sqlite"
	bindingLoader "github.com/dapr/dapr/pkg/components/bindings"
	configurationLoader "github.com/dapr/dapr/pkg/components/configuration"
	lockLoader "github.com/dapr/dapr/pkg/components/lock"
	middlewareLoader "github.com/dapr/dapr/pkg/components/middleware/http"
	nrLoader "github.com/dapr/dapr/pkg/components/nameresolution"
	pubsubLoader "github.com/dapr/dapr/pkg/components/pubsub"
	secretLoader "github.com/dapr/dapr/pkg/components/secretstores"
	stateLoader "github.com/dapr/dapr/pkg/components/state"
	"github.com/dapr/dapr/pkg/healthz"
	"github.com/dapr/dapr/pkg/metrics"
	"github.com/dapr/dapr/pkg/runtime"
	"github.com/dapr/dapr/pkg/runtime/registry"
	"github.com/dapr/dapr/pkg/security/fake"
	"github.com/dapr/kit/logger"
)

type EmbeddedRuntime struct {
	rt     *runtime.DaprRuntime
	cancel context.CancelFunc
}

func NewEmbeddedRuntime(appID string, httpPort, grpcPort int) (*EmbeddedRuntime, error) {
	hz := healthz.New()

	// Name Resolution
	nrReg := nrLoader.NewRegistry()
	nrReg.Logger = logger.NewLogger("dapr.nameresolution")
	nrReg.RegisterComponent(mdnsnr.NewResolver, "mdns")

	// State Store
	stateReg := stateLoader.NewRegistry()
	stateReg.Logger = logger.NewLogger("dapr.statestore")
	stateReg.RegisterComponent(sqlitestate.NewSQLiteStateStore, "sqlite")

	// Bindings
	bindingReg := bindingLoader.NewRegistry()
	bindingReg.RegisterOutputBinding(httpBinding.NewHTTP, "http")

	registryOptions := registry.NewOptions().
		WithNameResolutions(nrReg).
		WithSecretStores(secretLoader.NewRegistry()).
		WithPubSubs(pubsubLoader.NewRegistry()).
		WithStateStores(stateReg).
		WithBindings(bindingReg).
		WithHTTPMiddlewares(middlewareLoader.NewRegistry()).
		WithConfigurations(configurationLoader.NewRegistry()).
		WithLocks(lockLoader.NewRegistry())

	// Ensure components directory exists
	componentsDir := "./components"
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create components directory: %w", err)
	}

	// 1. Create a default state store if not exists
	stateStorePath := componentsDir + "/statestore.yaml"
	if _, err := os.Stat(stateStorePath); os.IsNotExist(err) {
		os.WriteFile(stateStorePath, []byte(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.sqlite
  version: v1
  metadata:
  - name: connectionString
    value: "data.db"
`), 0644)
	}

	// 2. Create SurrealDB HTTP Binding if not exists
	surrealPath := componentsDir + "/surrealdb.yaml"
	if _, err := os.Stat(surrealPath); os.IsNotExist(err) {
		os.WriteFile(surrealPath, []byte(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: surrealdb
spec:
  type: bindings.http
  version: v1
  metadata:
  - name: url
    value: "http://localhost:8000/sql"
`), 0644)
	}

	cfg := &runtime.Config{
		AppID:                         appID,
		DaprHTTPPort:                  strconv.Itoa(httpPort),
		DaprAPIGRPCPort:               strconv.Itoa(grpcPort),
		DaprInternalGRPCPort:          "0",
		DaprInternalGRPCListenAddress: "127.0.0.1",
		ProfilePort:                   "0",
		ApplicationPort:               "0",
		AppProtocol:                   "http",
		AppMaxConcurrency:             -1,
		MaxRequestSize:                -1,
		ReadBufferSize:                -1,
		SchedulerStreams:              1,
		Mode:                          "standalone",
		Config:                        []string{},
		Metrics: metrics.Options{
			Enabled: false,
			Healthz: hz,
		},
		Registry:      registryOptions,
		Healthz:       hz,
		ResourcesPath: []string{componentsDir},
		Security:      fake.New(), // Use fake security handler to avoid nil panic
	}

	ctx, cancel := context.WithCancel(context.Background())

	rt, err := runtime.FromConfig(ctx, cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize Dapr runtime: %w", err)
	}

	return &EmbeddedRuntime{
		rt:     rt,
		cancel: cancel,
	}, nil
}

func (r *EmbeddedRuntime) Start(ctx context.Context) error {
	go func() {
		fmt.Printf("[Dapr] Starting embedded runtime...\n")
		if err := r.rt.Run(ctx); err != nil {
			fmt.Printf("[Dapr] Runtime error: %v\n", err)
		}
	}()

	// Wait for readiness
	time.Sleep(2 * time.Second)
	return nil
}

func (r *EmbeddedRuntime) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}
