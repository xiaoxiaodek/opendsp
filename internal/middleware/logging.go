package middleware

import (
	"context"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

var (
	grpcRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "admanager_grpc_requests_total",
		Help: "Total gRPC requests handled by ad-manager.",
	}, []string{"method", "code"})

	grpcLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "admanager_grpc_latency_seconds",
		Help:    "gRPC request latency in seconds.",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"method"})
)

func UnaryLoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	st, _ := status.FromError(err)
	code := st.Code().String()

	grpcRequests.WithLabelValues(info.FullMethod, code).Inc()
	grpcLatency.WithLabelValues(info.FullMethod).Observe(duration.Seconds())

	logger.Info().
		Str("method", info.FullMethod).
		Dur("duration", duration).
		Str("code", code).
		Msg("gRPC request")

	return resp, err
}
