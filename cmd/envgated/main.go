package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcRouter "github.com/whenipush/envgate/internal/delivery/grpc"
	grpcHandler "github.com/whenipush/envgate/internal/delivery/grpc/handler"
	httpRouter "github.com/whenipush/envgate/internal/delivery/http" // Пакет нашего нового роутера
	"github.com/whenipush/envgate/internal/pkg/config"
	database "github.com/whenipush/envgate/internal/pkg/database/boltdb"
	"github.com/whenipush/envgate/internal/repository"
	"github.com/whenipush/envgate/internal/service/project"
	"github.com/whenipush/envgate/internal/service/token"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.MustLoadConfigServer()

	db := database.MustConnect(cfg.Database.DBPath)

	repo := repository.NewRepository(db)
	projectService := project.NewService(repo, cfg.GetAESKey())
	tokenService := token.NewService(repo, projectService, cfg.GetAESKey())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 2)

	gHandler := grpcHandler.NewHandler(tokenService, projectService)
	grpcServer := grpc.NewServer()
	grpcRouter.RegisterServices(grpcServer, gHandler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.App.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen gRPC port %d: %v", cfg.App.GRPCPort, err)
	}

	go func() {
		log.Printf("Starting gRPC server on :%d...", cfg.App.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	router := httpRouter.NewRouter(projectService, tokenService, cfg)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.RESTPort),
		Handler: router,
	}

	go func() {
		log.Printf("Starting Gin HTMX HTTP server on :%d...", cfg.App.RESTPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		log.Printf("Critical error detected, shutting down: %v", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received. Starting graceful shutdown...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		done := make(chan struct{})

		go func() {
			log.Println("Shutting down Gin HTTP server...")
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("HTTP shutdown error: %v", err)
			}

			log.Println("Shutting down gRPC server...")
			grpcServer.GracefulStop()

			log.Println("Closing BoltDB connection...")
			if err := db.Close(); err != nil {
				log.Printf("Error closing BoltDB: %v", err)
			}

			close(done)
		}()

		select {
		case <-done:
			log.Println("All servers stopped successfully. Safe to exit.")
		case <-shutdownCtx.Done():
			log.Println("Shutdown timeout exceeded. Forcing exit.")
			os.Exit(1)
		}
	}
}
