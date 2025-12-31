package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
	"github.com/mackerelio-labs/sabatrafficd/internal/mackerel"
	"github.com/mackerelio-labs/sabatrafficd/internal/sendqueue"
	"github.com/mackerelio-labs/sabatrafficd/internal/ticker"
	"github.com/mackerelio-labs/sabatrafficd/internal/worker"
)

type serveAndShutdown interface {
	Serve() error
	Shutdown(ctx context.Context) error

	CollectorID() string
	Reload(conf *config.CollectorConfig)
	Alive() bool
}

var (
	srvs []serveAndShutdown
	conf *config.Config

	configFilename string

	doShutdown   atomic.Bool
	idleShutdown = make(chan struct{})

	client       *mackerel.Mackerel
	queueHandler *sendqueue.Queue
)

func main() {
	ctx := context.Background()
	flag.StringVar(&configFilename, "config", "config.yaml", "config `filename`")
	flag.Parse()

	var err error
	conf, err = config.Init(configFilename)
	if err != nil {
		slog.ErrorContext(ctx, "failed read config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	client = mackerel.New(conf.ApiKey)
	queueHandler = sendqueue.New(client)

	srvs = append(srvs, queueHandler)

	for idx := range conf.Collector {
		if len(conf.Collector[idx].CustomMIBsGraphDefs) > 0 {
			if err = client.CreateGraphDefs(ctx, conf.Collector[idx].CustomMIBsGraphDefs); err != nil {
				slog.WarnContext(ctx, "failed CreateGraphDefs", slog.String("error", err.Error()))
			}
		}

		srvs = append(srvs,
			worker.New(ticker.MetadataNew(conf.Collector[idx], client), 3*time.Hour),
			worker.New(ticker.New(conf.Collector[idx], queueHandler), time.Minute),
		)
	}

	trapSignalInterrupt()
	trapSignals()

	for _, s := range srvs {
		go func(s serveAndShutdown) {
			if err := s.Serve(); err != nil {
				slog.Error("failed Serve", slog.String("error", err.Error()))
				os.Exit(1)
			}
		}(s)
		time.Sleep(300 * time.Millisecond)
	}

	slog.Info("initialized.")
	<-idleShutdown
}

func trapSignalInterrupt() {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)

		for i := 0; true; i++ {
			<-quit

			if i > 0 {
				slog.Info("force shutdown", slog.String("Signal", "SIGINT"))
				os.Exit(2)
			}

			slog.Info("shutdown...", slog.String("Signal", "SIGINT"))
			go shutdown()
		}
	}()
}

func shutdown() {
	if !doShutdown.CompareAndSwap(false, true) {
		return
	}

	sdNotifyHelper(daemon.SdNotifyStopping)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	for _, s := range srvs {
		wg.Add(1)
		go func(ctx context.Context, s serveAndShutdown) {
			defer wg.Done()
			if err := s.Shutdown(ctx); err != nil {
				slog.ErrorContext(ctx, "failed Shutdown", slog.String("error", err.Error()))
				os.Exit(2)
			}
		}(ctx, s)
	}
	wg.Wait()
	close(idleShutdown)
}
