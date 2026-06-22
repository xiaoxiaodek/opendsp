package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/service/filegateway"
	"github.com/opendsp/opendsp/internal/storage"
	pb "github.com/opendsp/opendsp/gen/filegateway/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()

	d, cleanup, err := data.NewData(ctx)
	if err != nil {
		log.Fatalf("data: %v", err)
	}
	defer cleanup()

	cfg := storage.ConfigFromEnv()
	backend, err := storage.NewFromConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	log.Printf("storage backend: %s", cfg.Backend)

	svc := filegateway.NewService(backend, d)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9092"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen gRPC: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFileGatewayServer(grpcServer, svc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("file-gateway gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := pb.RegisterFileGatewayHandlerFromEndpoint(ctx, mux, "localhost:"+grpcPort, opts); err != nil {
		log.Fatalf("register gateway: %v", err)
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "9001"
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/creative/", svc.FileProxyHandler())
	httpMux.Handle("/proof/", svc.FileProxyHandler())
	httpMux.Handle("/asset/", svc.FileProxyHandler())
	httpMux.Handle("/upload/creative/", svc.UploadReceiverHandler())
	httpMux.Handle("/upload/proof/", svc.UploadReceiverHandler())
	httpMux.Handle("/upload/asset/", svc.UploadReceiverHandler())
	httpMux.Handle("/", mux)

	httpServer := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      httpMux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		log.Printf("file-gateway HTTP listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()
}
