package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/opendsp/opendsp/internal/data"
	"github.com/opendsp/opendsp/internal/index"
	"github.com/opendsp/opendsp/internal/syncer"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d, cleanup, err := data.NewData(ctx)
	if err != nil {
		log.Fatalf("data: %v", err)
	}
	defer cleanup()

	idx := index.New()

	fullSync := syncer.NewFullSyncer(idx, d)
	eventSub := syncer.NewEventSubscriber(d.Rdb, idx)

	go fullSync.Run(ctx)
	go eventSub.Run(ctx)

	log.Println("ad-syncer started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	cancel()
}
