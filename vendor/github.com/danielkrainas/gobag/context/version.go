package bagcontext

import "context"

func WithVersion(ctx context.Context, version string) context.Context {
	ctx = WithValue(ctx, "version", version)
	return WithLogger(ctx, GetLogger(ctx, "version"))
}

func GetVersion(ctx context.Context) string {
	return GetStringValue(ctx, "version")
}
