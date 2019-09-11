package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/danielkrainas/gobag/context"
	baglogger "github.com/danielkrainas/gobag/log"
	"github.com/danielkrainas/sake/pkg/api"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/util/log"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

type RootContext context.Context

func InitializeCoordinator(ctx context.Context, hub service.HubConnector, storage service.StorageService, cache service.CacheService) (service.CoordinatorService, error) {
	coordinator, err := service.NewCoordinator(ctx, hub, cache, storage)
	if err != nil {
		return nil, fmt.Errorf("coordinator init failed: %v", err)
	}

	go func() {
		timeoutWorker := &service.TimeoutWorker{
			ExitOnError:   false,
			ScanFrequency: 1 * time.Second,
		}

		if err := timeoutWorker.Run(ctx, coordinator); err != nil {
			log.Error("expiration worker failed", zap.Error(err))
		}

		log.Info("expiration worker exit")
	}()

	return coordinator, nil
}

func InitializeHub(ctx context.Context, config *service.Config) (service.HubConnector, error) {
	var hub service.HubConnector
	var err error
	switch config.HubProvider {
	case "in-memory":
		hub = service.NewDebugHub()
	case "stan":
		hub, err = service.NewStanHub(config.Nats.ClusterID, config.Nats.Server, config.Nats.ClientID, config.Nats.DurableName)
	default:
		return nil, fmt.Errorf("invalid hub provider %q", config.HubProvider)
	}

	if err != nil {
		return nil, fmt.Errorf("hub init failed: %v", err)
	}

	log.InfoS("%s hub ready", config.HubProvider)
	return hub, nil
}

func InitializeStorage(ctx context.Context, config *service.Config) (service.StorageService, error) {
	storage, err := service.NewDebugStorage([]*service.Workflow{service.Workflows[1]}, nil)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %v", err)
	}

	return storage, nil
}

func InitializeCache(ctx context.Context, config *service.Config, storage service.StorageService) (service.CacheService, error) {
	var cache service.CacheService
	var err error
	cache, err = service.NewInMemoryCache()
	if err != nil {
		return nil, fmt.Errorf("cache init failed: %v", err)
	}

	cache = &service.WriteThruCache{
		CacheService: cache,
		Storage:      storage,
	}

	return cache, nil
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
	baglogger.Info(ctx, "using %q logging formatter", config.Log.Formatter)
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
