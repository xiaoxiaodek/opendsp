package biz

import "context"

// EventPublisher publishes domain events for inter-service coordination.
type EventPublisher interface {
	Publish(ctx context.Context, channel string, data []byte) error
}
