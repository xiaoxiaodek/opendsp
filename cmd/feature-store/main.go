package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opendsp/opendsp/internal/config"
	featureinfra "github.com/opendsp/opendsp/internal/infrastructure/persistence/redis/feature"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, _, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	redisAddr := cfg.Redis.Addr
	kafkaBrokers := cfg.Kafka.Brokers
	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "feature-store"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	consumer := featureinfra.NewFeatureConsumer(featureinfra.ConsumerConfig{
		Brokers: kafkaBrokers,
		GroupID: groupID,
	})

	aggCfg := featureinfra.DefaultAggregatorConfig()
	if cfg.FeatureStore.AggregateIntervalSec > 0 {
		aggCfg.AggregateInterval = time.Duration(cfg.FeatureStore.AggregateIntervalSec) * time.Second
	}
	aggregator := featureinfra.NewAggregator(rdb, consumer.Buffer(), aggCfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Run(ctx)
	go aggregator.Run(ctx)

	log.Println("feature-store: running with Kafka consumer + Redis aggregator")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("feature-store: shutting down...")
	cancel()
	consumer.Close()
}
