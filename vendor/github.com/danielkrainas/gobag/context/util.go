package bagcontext

import (
	"context"
	"time"
)

func GetStringValue(ctx context.Context, key interface{}) string {
	if value, ok := ctx.Value(key).(string); ok {
		return value
	}

	return ""
}

func Since(ctx context.Context, key interface{}) time.Duration {
	if startedAt, ok := ctx.Value(key).(time.Time); ok {
		return time.Since(startedAt)
	}

	return 0
}

func GetInstanceID(ctx context.Context) string {
	return GetStringValue(ctx, "instance.id")
}
