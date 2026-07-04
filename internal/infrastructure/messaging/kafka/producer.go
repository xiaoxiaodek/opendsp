package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer wraps Kafka writers for async event publishing.
type Producer struct {
	bidWriter        *kafka.Writer
	impressionWriter *kafka.Writer
	clickWriter      *kafka.Writer
	conversionWriter *kafka.Writer
}

// ProducerConfig holds Kafka connection settings.
type ProducerConfig struct {
	Brokers []string
}

// NewProducer creates a Kafka producer with writers for each event topic.
func NewProducer(cfg ProducerConfig) *Producer {
	sharedTransport := &kafka.Transport{
		DialTimeout: 3 * time.Second,
	}

	newWriter := func(topic string) *kafka.Writer {
		return &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
			Transport:    sharedTransport,
		}
	}

	return &Producer{
		bidWriter:        newWriter("bid.events"),
		impressionWriter: newWriter("impression.events"),
		clickWriter:      newWriter("click.events"),
		conversionWriter: newWriter("conversion.events"),
	}
}

// EmitBid publishes a bid event asynchronously.
func (p *Producer) EmitBid(ctx context.Context, event BidEvent) {
	p.publish(ctx, p.bidWriter, event.RequestID, event)
}

// EmitImpression publishes an impression event asynchronously.
func (p *Producer) EmitImpression(ctx context.Context, event ImpressionEvent) {
	p.publish(ctx, p.impressionWriter, event.ClickID, event)
}

// EmitClick publishes a click event asynchronously.
func (p *Producer) EmitClick(ctx context.Context, event ClickEvent) {
	p.publish(ctx, p.clickWriter, event.ClickID, event)
}

// EmitConversion publishes a conversion event asynchronously.
func (p *Producer) EmitConversion(ctx context.Context, event ConversionEvent) {
	p.publish(ctx, p.conversionWriter, event.ClickID, event)
}

func (p *Producer) publish(ctx context.Context, w *kafka.Writer, key string, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("kafka: marshal error for topic %s: %v", w.Topic, err)
		return
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := w.WriteMessages(bgCtx, kafka.Message{
		Key:   []byte(key),
		Value: data,
	}); err != nil {
		log.Printf("kafka: write error for topic %s: %v", w.Topic, err)
	}
}

// Close flushes and closes all writers.
func (p *Producer) Close() error {
	var errs []error
	for _, w := range []*kafka.Writer{p.bidWriter, p.impressionWriter, p.clickWriter, p.conversionWriter} {
		if err := w.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("kafka close errors: %v", errs)
	}
	return nil
}
