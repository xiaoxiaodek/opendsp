package rta

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/opendsp/opendsp/gen/adserver/v1"
)

type Client struct {
	registry *Registry
	conns    map[int64]*grpc.ClientConn
	breakers map[int64]*gobreaker.CircuitBreaker
}

func NewClient(registry *Registry) *Client {
	c := &Client{
		registry: registry,
		conns:    make(map[int64]*grpc.ClientConn),
		breakers: make(map[int64]*gobreaker.CircuitBreaker),
	}

	for _, entry := range registry.entries {
		st := gobreaker.Settings{
			Name:        fmt.Sprintf("rta-%d", entry.AdvertiserID),
			MaxRequests: 1,
			Interval:    60 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5
			},
		}
		c.breakers[entry.AdvertiserID] = gobreaker.NewCircuitBreaker(st)
	}

	return c
}

func (c *Client) Query(ctx context.Context, advertiserID int64, deviceID, mediaID, requestID string) (bool, error) {
	entry, ok := c.registry.Get(advertiserID)
	if !ok {
		return true, nil
	}

	breaker, ok := c.breakers[advertiserID]
	if !ok {
		return true, nil
	}

	conn, err := c.getConn(entry)
	if err != nil {
		return true, nil
	}

	timeout := time.Duration(entry.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 15 * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := breaker.Execute(func() (interface{}, error) {
		client := pb.NewRTAServiceClient(conn)
		resp, err := client.Query(callCtx, &pb.RTARequest{
			RequestId:    requestID,
			DeviceId:     deviceID,
			MediaId:      mediaID,
			AdvertiserId: advertiserID,
		})
		if err != nil {
			return false, err
		}
		return resp.Allowed, nil
	})

	if err != nil {
		return true, nil
	}

	allowed, _ := result.(bool)
	return allowed, nil
}

func (c *Client) getConn(entry RTAEntry) (*grpc.ClientConn, error) {
	if conn, ok := c.conns[entry.AdvertiserID]; ok {
		return conn, nil
	}
	conn, err := grpc.NewClient(entry.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c.conns[entry.AdvertiserID] = conn
	return conn, nil
}

func (c *Client) Close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}
