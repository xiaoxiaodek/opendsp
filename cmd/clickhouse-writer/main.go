package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opendsp/opendsp/internal/config"
	kafkainfra "github.com/opendsp/opendsp/internal/infrastructure/messaging/kafka"
	"github.com/opendsp/opendsp/internal/infrastructure/persistence/clickhouse"
	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	cfg, _, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	kafkaBroker := cfg.Kafka.Brokers[0]

	chClient, err := clickhouse.NewClient(clickhouse.Config{
		Host:     cfg.ClickHouse.Host,
		Port:     cfg.ClickHouse.Port,
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	defer chClient.Close()

	writer := clickhouse.NewWriter(chClient)
	if err := writer.EnsureTables(context.Background()); err != nil {
		log.Fatalf("clickhouse ensure tables: %v", err)
	}

	topics := []string{"bid.events", "impression.events", "click.events", "conversion.events"}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		GroupID:  "clickhouse-writer",
		Topic:    topics[0],
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	log.Printf("clickhouse-writer: consuming topics %v from %s", topics, kafkaBroker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("kafka fetch: %v", err)
				time.Sleep(time.Second)
				continue
			}

			if err := processMessage(ctx, writer, msg); err != nil {
				log.Printf("process %s: %v", msg.Topic, err)
			}

			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("kafka commit: %v", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("clickhouse-writer: shutting down...")
}

func processMessage(ctx context.Context, writer *clickhouse.Writer, msg kafkago.Message) error {
	switch msg.Topic {
	case "bid.events":
		var event kafkainfra.BidEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}
		return writer.WriteBidEvent(ctx, event)
	case "impression.events":
		var event kafkainfra.ImpressionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}
		return writer.WriteImpressionEvent(ctx, event)
	case "click.events":
		var event kafkainfra.ClickEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}
		return writer.WriteClickEvent(ctx, event)
	case "conversion.events":
		var event kafkainfra.ConversionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}
		return writer.WriteConversionEvent(ctx, event)
	}
	return nil
}
