package feature

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// RawEvent represents a decoded event from Kafka for feature aggregation.
type RawEvent struct {
	EventType string
	UserID    string
	MediaID   string
	GeoCity   string
	Timestamp int64
	CTR       float64
	CVR       float64
}

// ConsumerConfig holds Kafka consumer settings.
type ConsumerConfig struct {
	Brokers []string
	GroupID string
}

// FeatureConsumer reads events from Kafka topics for feature aggregation.
type FeatureConsumer struct {
	reader *kafka.Reader
	buffer chan RawEvent
}

// NewFeatureConsumer creates a consumer reading from multiple topics.
func NewFeatureConsumer(cfg ConsumerConfig) *FeatureConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.GroupID,
		GroupTopics: []string{"bid.events", "impression.events", "click.events", "conversion.events"},
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     100 * time.Millisecond,
	})

	return &FeatureConsumer{
		reader: reader,
		buffer: make(chan RawEvent, 10000),
	}
}

// Buffer returns the channel for raw events.
func (c *FeatureConsumer) Buffer() <-chan RawEvent {
	return c.buffer
}

// Run starts consuming events and pushing to the buffer.
func (c *FeatureConsumer) Run(ctx context.Context) {
	defer close(c.buffer)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("feature-consumer: read error: %v", err)
			continue
		}

		event := decodeEvent(msg.Topic, msg.Value)
		if event.EventType != "" {
			select {
			case c.buffer <- event:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Close shuts down the consumer.
func (c *FeatureConsumer) Close() error {
	return c.reader.Close()
}

func decodeEvent(topic string, data []byte) RawEvent {
	switch topic {
	case "bid.events":
		var e struct {
			UserID  string  `json:"user_id"`
			MediaID string  `json:"media_id"`
			GeoCity string  `json:"geo_city"`
			PredCTR float64 `json:"pctr"`
			PredCVR float64 `json:"pcvr"`
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return RawEvent{}
		}
		return RawEvent{
			EventType: "bid",
			UserID:    e.UserID,
			MediaID:   e.MediaID,
			GeoCity:   e.GeoCity,
			Timestamp: time.Now().UnixMilli(),
			CTR:       e.PredCTR,
			CVR:       e.PredCVR,
		}

	case "impression.events":
		var e struct {
			UserID  string `json:"user_id"`
			MediaID string `json:"media_id"`
			GeoCity string `json:"geo_city"`
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return RawEvent{}
		}
		return RawEvent{
			EventType: "impression",
			UserID:    e.UserID,
			MediaID:   e.MediaID,
			GeoCity:   e.GeoCity,
			Timestamp: time.Now().UnixMilli(),
		}

	case "click.events":
		var e struct {
			UserID  string `json:"user_id"`
			MediaID string `json:"media_id"`
			GeoCity string `json:"geo_city"`
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return RawEvent{}
		}
		return RawEvent{
			EventType: "click",
			UserID:    e.UserID,
			MediaID:   e.MediaID,
			GeoCity:   e.GeoCity,
			Timestamp: time.Now().UnixMilli(),
		}

	case "conversion.events":
		var e struct {
			UserID  string `json:"user_id"`
			MediaID string `json:"media_id"`
			GeoCity string `json:"geo_city"`
		}
		if err := json.Unmarshal(data, &e); err != nil {
			return RawEvent{}
		}
		return RawEvent{
			EventType: "conversion",
			UserID:    e.UserID,
			MediaID:   e.MediaID,
			GeoCity:   e.GeoCity,
			Timestamp: time.Now().UnixMilli(),
		}
	}

	return RawEvent{}
}
