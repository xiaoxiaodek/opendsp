package sender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opendsp/opendsp/internal/mockadx/config"
	"github.com/opendsp/opendsp/internal/mockadx/encoder"
	"github.com/opendsp/opendsp/internal/mockadx/funnel"
	"github.com/opendsp/opendsp/internal/mockadx/generator"
	"github.com/opendsp/opendsp/internal/mockadx/report"
)

type Pool struct {
	cfg      config.Config
	gen      generator.Generator
	enc      encoder.Encoder
	store    *funnel.Store
	reporter *report.Reporter

	client  *http.Client
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped atomic.Bool
}

func NewPool(cfg config.Config, gen generator.Generator, enc encoder.Encoder, store *funnel.Store, reporter *report.Reporter) *Pool {
	return &Pool{
		cfg:      cfg,
		gen:      gen,
		enc:      enc,
		store:    store,
		reporter: reporter,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (p *Pool) Start(ctx context.Context) {
	p.ctx, p.cancel = context.WithCancel(ctx)

	interval := time.Second / time.Duration(p.cfg.QPS)
	burst := p.cfg.Concurrency
	if burst > p.cfg.QPS {
		burst = p.cfg.QPS
	}

	tokens := make(chan struct{}, burst)
	for i := 0; i < burst; i++ {
		tokens <- struct{}{}
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				select {
				case <-tokens:
					p.wg.Add(1)
					go func() {
						defer p.wg.Done()
						defer func() { tokens <- struct{}{} }()
						p.sendOne()
					}()
				default:
				}
			}
		}
	}()
}

func (p *Pool) Stop() {
	p.stopped.Store(true)
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *Pool) sendOne() {
	spec := p.gen.Next(p.ctx)
	if spec == nil {
		return
	}

	body, err := p.enc.Encode(p.ctx, spec)
	if err != nil {
		log.Printf("encode: %v", err)
		p.reporter.RecordStatus("error")
		return
	}

	req, err := http.NewRequestWithContext(p.ctx, http.MethodPost, p.cfg.Target+p.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		p.reporter.RecordStatus("error")
		return
	}
	req.Header.Set("Content-Type", p.enc.ContentType())
	if p.cfg.Gzip {
		req.Header.Set("Content-Encoding", "gzip")
	}

	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start)
	p.reporter.RecordLatency(latency)

	if err != nil {
		p.reporter.RecordStatus("error")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.reporter.RecordStatus("error")
		return
	}

	status := fmt.Sprintf("%d", resp.StatusCode)
	p.reporter.RecordStatus(status)

	if resp.StatusCode == 200 && len(respBody) > 0 {
		bidSpec, err := p.enc.Decode(respBody)
		if err != nil {
			return
		}
		if bidSpec.BidID != "" {
			p.store.Insert(bidSpec.BidID, &funnel.BidContext{
				BidID:     bidSpec.BidID,
				RequestID: bidSpec.RequestID,
				Price:     bidSpec.Price,
			})
			p.reporter.RecordBid()
		}
	}
}