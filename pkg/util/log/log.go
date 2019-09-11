package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

func init() {
	logger, _ = zap.NewDevelopment(zap.AddCallerSkip(1))
	sugar = logger.Sugar()
}

func Combine(field zap.Field, fields ...zap.Field) []zap.Field {
	return append([]zap.Field{field}, fields...)
}

func CombineAll(fieldGroups ...[]zap.Field) []zap.Field {
	v := make([]zap.Field, 0)
	for _, g := range fieldGroups {
		v = append(v, g...)
	}

	return v
}

func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

func FatalS(format string, args ...interface{}) {
	if len(args) > 0 {
		sugar.Fatalf(format, args...)
	} else {
		sugar.Fatal(format)
	}
}

func InfoS(format string, args ...interface{}) {
	if len(args) > 0 {
		sugar.Infof(format, args...)
	} else {
		sugar.Info(format)
	}
}

func DebugS(format string, args ...interface{}) {
	if len(args) > 0 {
		sugar.Debugf(format, args...)
	} else {
		sugar.Debug(format)
	}
}

func WarnS(format string, args ...interface{}) {
	if len(args) > 0 {
		sugar.Warnf(format, args...)
	} else {
		sugar.Warn(format)
	}
}

func ErrorS(format string, args ...interface{}) {
	if len(args) > 0 {
		sugar.Errorf(format, args...)
	} else {
		sugar.Error(format)
	}
}

func At(level zapcore.Level, msg string, fields ...zap.Field) {
	switch level {
	case zapcore.FatalLevel:
		logger.Fatal(msg, fields...)
	case zapcore.InfoLevel:
		logger.Info(msg, fields...)
	case zapcore.DebugLevel:
		logger.Debug(msg, fields...)
	}
}

func Atf(level zapcore.Level, format string, args ...interface{}) {
	if len(args) > 0 {
		switch level {
		case zapcore.FatalLevel:
			sugar.Fatalf(format, args...)
		case zapcore.InfoLevel:
			sugar.Infof(format, args...)
		case zapcore.DebugLevel:
			sugar.Debugf(format, args...)
		}
	} else {
		switch level {
		case zapcore.FatalLevel:
			sugar.Fatal(format)
		case zapcore.InfoLevel:
			sugar.Info(format)
		case zapcore.DebugLevel:
			sugar.Debug(format)
		}
	}
}
