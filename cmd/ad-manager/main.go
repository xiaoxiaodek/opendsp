package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/opendsp/opendsp/internal/ai"
	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/config"
	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/dmp"
	clickhouseinfra "github.com/opendsp/opendsp/internal/infrastructure/persistence/clickhouse"
	"github.com/opendsp/opendsp/internal/middleware"
	"github.com/opendsp/opendsp/internal/service/admanager"
	fraudhttp "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/fraud"
	roiinfra "github.com/opendsp/opendsp/internal/infrastructure/persistence/postgres/roi"
	pb "github.com/opendsp/opendsp/gen/admanager/v1"
	filegatewaypb "github.com/opendsp/opendsp/gen/filegateway/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()

	cfg, _, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	d, cleanup, err := data.NewData(ctx, cfg.Database, cfg.Redis)
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

	publisher := data.NewRedisPublisher(d.Rdb)
	campaignUC := biz.NewCampaignUseCase(campaignRepo, publisher)
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

	grpcPort := strconv.Itoa(cfg.Server.GRPCPort)
	httpPort := strconv.Itoa(cfg.Server.Port)

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
		os.Getenv("FILE_GATEWAY_ADDR"),
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
	fraudHandler := fraudhttp.NewHTTPHandler(fraudhttp.NewBlacklistRepo(d.Rdb))
	httpMux.Handle("/api/antifraud/blacklist", fraudHandler)
	httpMux.Handle("/api/antifraud/blacklist/", fraudHandler)
	eventsHandler := fraudhttp.NewEventsHandler(d.Pool)
	httpMux.Handle("/api/antifraud/events", eventsHandler)
	httpMux.Handle("/api/antifraud/stats", eventsHandler)
	httpMux.Handle("/metrics", promhttp.Handler())
	httpMux.Handle("/api/v1/upload/", uploadHandler)
	httpMux.Handle("/api/v1/sync/", syncHandler)

	roiHandler := admanager.NewROIHandler(roiinfra.NewConversionRepo(d.Pool))
	httpMux.Handle("/api/roi/", roiHandler)

	dpaHandler := admanager.NewDPAHandler()
	httpMux.Handle("/api/dpa/", dpaHandler)

	chClient, err := clickhouseinfra.NewClient(clickhouseinfra.Config{
		Host:     cfg.ClickHouse.Host,
		Port:     cfg.ClickHouse.Port,
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	if err != nil {
		log.Printf("clickhouse: %v (settlement API disabled)", err)
	} else {
		chWriter := clickhouseinfra.NewWriter(chClient)
		settlementHandler := admanager.NewSettlementHandler(chWriter)
		httpMux.Handle("/api/settlement/", settlementHandler)
	}

	aiEnabled := os.Getenv("AI_ENABLED")
	if aiEnabled != "false" {
		llmClient := ai.NewLLMClientWithConfig(cfg.AI.APIKey, cfg.AI.BaseURL, cfg.AI.Model)
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

