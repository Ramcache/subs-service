package logger

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

func WithLogger(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}
	return zap.NewNop()
}
