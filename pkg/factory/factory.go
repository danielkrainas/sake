package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/danielkrainas/gobag/context"
	"github.com/danielkrainas/gobag/log"
	"github.com/danielkrainas/sake/pkg/api"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/sirupsen/logrus"
)

type RootContext context.Context

func InitializeCoordinator(ctx context.Context, hub service.HubConnector) (service.CoordinatorService, error) {
	return service.NewCoordinator(ctx, hub), nil
}

func InitializeHub(ctx context.Context) (service.HubConnector, error) {
	return service.NewTestHub(), nil
}

func InitializeServer(ctx context.Context, config *service.Config, mux *api.Mux) (service.APIServer, error) {
	return api.NewServer(ctx, mux, api.ServerConfig{
		Addr: config.HTTP.Addr,
	})
}

func InitializeAPI() (*api.Mux, error) {
	return api.NewMux()
}

func InitializeLoggingContext(rctx RootContext, config *service.Config) (context.Context, error) {
	ctx := context.Context(rctx)
	logrus.SetLevel(logLevel(config.Log.Level))
	formatter := config.Log.Formatter
	if formatter == "" {
		formatter = "text"
	}

	switch formatter {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})

	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339Nano,
		})

	default:
		if config.Log.Formatter != "" {
			return nil, fmt.Errorf("unsupported formatter: %q", config.Log.Formatter)
		}
	}

	if len(config.Log.Fields) > 0 {
		var fields []interface{}
		for k := range config.Log.Fields {
			fields = append(fields, k)
		}

		ctx = bagcontext.WithValues(ctx, config.Log.Fields)
		ctx = bagcontext.WithLogger(ctx, bagcontext.GetLogger(ctx, fields...))
	}

	ctx = bagcontext.WithLogger(ctx, bagcontext.GetLogger(ctx))
	log.Info(ctx, "using %q logging formatter", config.Log.Formatter)
	return ctx, nil
}

func logLevel(level string) logrus.Level {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		l = logrus.InfoLevel
		logrus.Warnf("error parsing level %q: %v, using %q", level, err, l)
	}

	return l
}
