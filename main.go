package main

import (
	"math/rand"
	"runtime"
	"time"

	gobagcontext "github.com/danielkrainas/gobag/context"
	"go.uber.org/zap"

	"github.com/danielkrainas/sake/pkg/cmd"
	"github.com/danielkrainas/sake/pkg/util/log"
)

var appVersion string

const defaultVersion = "0.0.0-dev"

func main() {
	if appVersion == "" {
		appVersion = defaultVersion
	}

	rand.Seed(time.Now().Unix())

	log.Info("starting", zap.String("app_version", appVersion), zap.String("go_version", runtime.Version()))
	ctx := gobagcontext.WithVersion(gobagcontext.Background(), appVersion)
	cmd.Execute(ctx)
}
