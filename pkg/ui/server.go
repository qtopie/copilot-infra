package ui

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	taskv1 "github.com/qtopie/copilot-infra/pkg/api/proto/v1"
	"github.com/qtopie/copilot-infra/frontend"
)

type Server struct {
	port     int
	grpcPort int
}

func NewServer(port, grpcPort int) *Server {
	return &Server{
		port:     port,
		grpcPort: grpcPort,
	}
}

func (s *Server) Start() error {
	// 1. Get the sub-filesystem for the dist folder
	strippedFS, err := fs.Sub(frontend.DistFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to get sub FS: %w", err)
	}

	// 2. Setup handlers
	mux := http.NewServeMux()

	// API Proxy using grpc-gateway
	ctx := context.Background()
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := fmt.Sprintf("localhost:%d", s.grpcPort)
	
	if err := taskv1.RegisterTaskServiceHandlerFromEndpoint(ctx, gwMux, endpoint, opts); err != nil {
		return fmt.Errorf("failed to register grpc gateway: %w", err)
	}

	mux.Handle("/api/", http.StripPrefix("/api", gwMux))

	// Static files
	fileServer := http.FileServer(http.FS(strippedFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the file doesn't exist, serve index.html (for SPA routing)
		_, err := strippedFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil && r.URL.Path != "/" {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}))

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Admin UI server listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}
