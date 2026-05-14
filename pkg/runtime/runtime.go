package runtime

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	mdnsnr "github.com/dapr/components-contrib/nameresolution/mdns"
	nrLoader "github.com/dapr/dapr/pkg/components/nameresolution"
	"github.com/dapr/dapr/pkg/healthz"
	"github.com/dapr/dapr/pkg/metrics"
	"github.com/dapr/dapr/pkg/runtime"
	"github.com/dapr/dapr/pkg/runtime/registry"
	"github.com/dapr/kit/logger"
)

type EmbeddedRuntime struct {
	rt     *runtime.DaprRuntime
	cancel context.CancelFunc
}

func NewEmbeddedRuntime(appID string, httpPort, grpcPort int) (*EmbeddedRuntime, error) {
	hz := healthz.New()
	nrRegistry := nrLoader.NewRegistry()
	nrRegistry.Logger = logger.NewLogger("dapr.nameresolution")
	nrRegistry.RegisterComponent(mdnsnr.NewResolver, "mdns")
	registryOptions := registry.NewOptions().WithNameResolutions(nrRegistry)

	// Ensure components directory exists
	componentsDir := "./components"
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create components directory: %w", err)
	}

	// Create a default state store if not exists
	stateStorePath := componentsDir + "/statestore.yaml"
	if _, err := os.Stat(stateStorePath); os.IsNotExist(err) {
		err = os.WriteFile(stateStorePath, []byte(`apiVersion: dapr.io/v1alpha1
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
		if err != nil {
			return nil, fmt.Errorf("failed to create default state store: %w", err)
		}
	}

	cfg := &runtime.Config{
		AppID:                         appID,
		DaprHTTPPort:                  strconv.Itoa(httpPort),
		DaprAPIGRPCPort:               strconv.Itoa(grpcPort),
		DaprInternalGRPCPort:          "0",
		DaprInternalGRPCListenAddress: "127.0.0.1",
		ProfilePort:                   "0",
		ApplicationPort:               "0", // No app callback needed for now
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
