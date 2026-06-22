package syncer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/opendsp/opendsp/internal/index"
	"github.com/redis/go-redis/v9"
)

type EventSubscriber struct {
	rdb   *redis.Client
	index *index.InvertedIndex
}

func NewEventSubscriber(rdb *redis.Client, idx *index.InvertedIndex) *EventSubscriber {
	return &EventSubscriber{rdb: rdb, index: idx}
}

type changeEvent struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

func (s *EventSubscriber) Run(ctx context.Context) {
	pubsub := s.rdb.Subscribe(ctx, "ad:change")
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Println("event subscriber started on channel ad:change")

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			var evt changeEvent
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				log.Printf("parse event error: %v", err)
				continue
			}
			log.Printf("received event: %s id=%d", evt.Type, evt.ID)
		}
	}
}
