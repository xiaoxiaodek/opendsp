package middleware

import (
	"context"
	"os"
	"runtime/debug"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var recoveryLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func UnaryRecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			recoveryLogger.Error().
				Str("method", info.FullMethod).
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("panic recovered")
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}
