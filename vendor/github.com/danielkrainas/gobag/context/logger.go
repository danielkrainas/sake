package bagcontext

import (
	"context"
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Print(args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalln(args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Panicln(args ...interface{})

	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Debugln(args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorln(args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnln(args ...interface{})
}

func WithLogger(ctx context.Context, logger Logger) context.Context {
	return WithValue(ctx, "logger", logger)
}

func GetLoggerWithField(ctx context.Context, key, value interface{}, keys ...interface{}) Logger {
	return getLogrusLogger(ctx, keys...).WithField(fmt.Sprint(key), value)
}

func GetLoggerWithFields(ctx context.Context, fields map[interface{}]interface{}, keys ...interface{}) Logger {
	lfields := make(logrus.Fields, len(fields))
	for k, v := range fields {
		lfields[fmt.Sprint(k)] = v
	}

	return getLogrusLogger(ctx, keys...).WithFields(lfields)
}

func GetLogger(ctx context.Context, keys ...interface{}) Logger {
	return getLogrusLogger(ctx, keys...)
}

func getLogrusLogger(ctx context.Context, keys ...interface{}) *logrus.Entry {
	var logger *logrus.Entry

	loggerInterface := ctx.Value("logger")
	if loggerInterface != nil {
		if l, ok := loggerInterface.(*logrus.Entry); ok {
			logger = l
		}
	}

	if logger == nil {
		fields := logrus.Fields{}

		instanceID := ctx.Value("instance.id")
		if instanceID != nil {
			fields["instance.id"] = instanceID
		}

		fields["go.version"] = runtime.Version()
		logger = logrus.StandardLogger().WithFields(fields)
	}

	fields := logrus.Fields{}
	for _, k := range keys {
		v := ctx.Value(k)
		if v != nil {
			fields[fmt.Sprint(k)] = v
		}
	}

	return logger.WithFields(fields)
}
