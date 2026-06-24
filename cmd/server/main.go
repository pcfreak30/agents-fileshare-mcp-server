package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/agent"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/build"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/cleanup"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/download"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	mcpserver "github.com/pcfreak30/agents-fileshare-mcp-server/internal/mcpserver"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/upload"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	cfg := config.Load()

	if err := os.MkdirAll(cfg.FilesDir(), 0755); err != nil {
		logger.Fatal("create data dir", zap.Error(err))
	}

	store, err := filestore.NewStore(cfg.DBPath(), logger)
	if err != nil {
		logger.Fatal("open store", zap.Error(err))
	}
	defer store.Close()

	agents := agent.NewManager(store)
	uploadHandler := upload.NewHandler(store, agents, cfg, logger)
	downloadHandler := download.NewHandler(store, cfg, logger)

	worker := cleanup.NewWorker(store, cfg, logger)
	worker.Start()
	defer worker.Stop()

	mcpSrv := mcpserver.NewMCPServer(store, agents, cfg, logger)

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    build.Name,
		Version: build.Version,
	}, &mcp.ServerOptions{
		InitializedHandler: mcpserver.InitializedHandler(agents, logger),
	})

	mcpSrv.RegisterTools(mcpServer)

	streamableHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle(model.RouteMCP, streamableHandler)
	mux.HandleFunc(model.RouteUpload, uploadHandler.ServeHTTP)
	mux.HandleFunc(model.RouteFile, downloadHandler.ServeHTTP)
	mux.HandleFunc(model.RouteHealth, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	go func() {
		logger.Info("server starting", zap.String("addr", cfg.ListenAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
