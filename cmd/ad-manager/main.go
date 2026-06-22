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
	"github.com/opendsp/opendsp/internal/ai"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/dmp"
	"github.com/opendsp/opendsp/internal/middleware"
	"github.com/opendsp/opendsp/internal/service/admanager"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	filegatewaypb "github.com/opendsp/opendsp/gen/filegateway/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	campaignRepo := data.NewCampaignRepo(d)
	adGroupRepo := data.NewAdGroupRepo(d)
	creativeRepo := data.NewCreativeRepo(d)
	reportRepo := data.NewReportRepo(d)
	advertiserRepo := data.NewAdvertiserRepo(d)
	proofRepo := data.NewProofMaterialRepo(d)
	balanceRepo := data.NewBalanceRepo(d)
	mediaRepo := data.NewMediaRepo(d)
	adPositionRepo := data.NewAdPositionRepo(d)
	adminRepo := data.NewAdminRepo(d)

	dmpRepo := data.NewDmpRepo(d)
	tagStore := dmp.NewTagStore(d.Rdb)
	resolver := dmp.NewAudienceResolver(tagStore, d.Rdb)
	lookalike := dmp.NewLookalikeEngine(dmpRepo, tagStore)

	campaignUC := biz.NewCampaignUseCase(campaignRepo, d.Rdb)
	adGroupUC := biz.NewAdGroupUseCase(adGroupRepo, d.Rdb)
	creativeUC := biz.NewCreativeUseCase(creativeRepo)
	reportUC := biz.NewReportUseCase(reportRepo)
	advertiserUC := biz.NewAdvertiserUseCase(advertiserRepo)
	proofMaterialUC := biz.NewProofMaterialUseCase(proofRepo)
	balanceUC := biz.NewBalanceUseCase(balanceRepo)
	mediaUC := biz.NewMediaUseCase(mediaRepo)
	adPositionUC := biz.NewAdPositionUseCase(adPositionRepo)
	adminUC := biz.NewAdminUseCase(adminRepo)
	syncRepo := data.NewSyncRepo(d)
	syncUC := biz.NewSyncUseCase(syncRepo)

	svc := admanager.NewAdManagerServiceFull(
		campaignUC, adGroupUC, creativeUC, reportUC,
		advertiserUC, proofMaterialUC, balanceUC,
		mediaUC, adPositionUC, adminUC,
		dmpRepo, tagStore, resolver, lookalike, d,
	)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9091"
	}
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	rbac := middleware.NewRBACInterceptor(nil)
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryRecoveryInterceptor,
			middleware.UnaryLoggingInterceptor,
			middleware.UnaryAuthInterceptor,
			rbac.UnaryInterceptor,
		),
	)
	pb.RegisterAdManagerServer(grpcServer, svc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("ad-manager gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	mux := runtime.NewServeMux(
		runtime.WithDisablePathLengthFallback(),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := pb.RegisterAdManagerHandlerFromEndpoint(ctx, mux, "localhost:"+grpcPort, opts); err != nil {
		log.Fatalf("register gateway: %v", err)
	}

	fgConn, err := grpc.NewClient(
		getEnv("FILE_GATEWAY_ADDR", "file-gateway:9092"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("file-gateway client: %v", err)
	}
	defer fgConn.Close()
	fgClient := filegatewaypb.NewFileGatewayClient(fgConn)

	uploadHandler := admanager.NewUploadHandler(fgClient, proofMaterialUC)
	syncHandler := admanager.NewSyncHandler(syncUC, syncRepo, creativeUC, adGroupUC, campaignUC)

	httpMux := http.NewServeMux()
	httpMux.Handle("/metrics", promhttp.Handler())
	httpMux.Handle("/api/v1/upload/", uploadHandler)
	httpMux.Handle("/api/v1/sync/", syncHandler)

	aiEnabled := os.Getenv("AI_ENABLED")
	if aiEnabled != "false" {
		llmClient := ai.NewLLMClient()
		toolRegistry := ai.NewToolRegistry(
			campaignRepo, adGroupRepo, creativeRepo, reportRepo,
			advertiserRepo, balanceRepo, adminRepo,
		)
		chatService := ai.NewChatService(llmClient, toolRegistry, d.Rdb)
		insightService := ai.NewInsightService(reportRepo, llmClient, d.Rdb)
		aiHandler := admanager.NewAIHandler(chatService, insightService)
		httpMux.Handle("/api/v1/ai/", aiHandler)
	}

	httpMux.Handle("/", mux)

	httpServer := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      httpMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Printf("ad-manager HTTP listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	aggregator := admanager.NewReportAggregator(d)
	go aggregator.Run(ctx)

	syncScheduler := admanager.NewSyncScheduler(syncUC, syncRepo)
	go syncScheduler.Run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	grpcServer.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
