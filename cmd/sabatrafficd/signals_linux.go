//go:build linux

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
	"github.com/mackerelio-labs/sabatrafficd/internal/sdnotify"
	"github.com/mackerelio-labs/sabatrafficd/internal/ticker"
	"github.com/mackerelio-labs/sabatrafficd/internal/worker"
)

func trapSignals() {
	// https://github.com/moby/moby/blob/3bd2edb375af8fab9f6366d57718fcc5561a7d93/cmd/dockerd/main.go#L22-L25
	signal.Ignore(syscall.SIGPIPE)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

		for sig := range sigCh {
			switch sig {
			case syscall.SIGQUIT:
				slog.Info("receive signal", slog.String("signal", "SIGQUIT"))
				os.Exit(2)

			case syscall.SIGTERM:
				slog.Info("receive signal", slog.String("signal", "SIGTERM"))
				shutdown()

			case syscall.SIGHUP:
				slog.Info("receive signal", slog.String("signal", "SIGHUP"))

				sdNotifyHelper(sdnotify.SendReloading())

				newConf, err := config.Init(configFilename)
				if err != nil {
					slog.Warn("failed parse config", slog.String("error", err.Error()))
					sdNotifyHelper(daemon.SdNotifyReady)
					continue
				}

				var (
					oldCollectorID []string
					newCollectorID []string
				)
				for _, conf := range newConf.Collector {
					newCollectorID = append(newCollectorID, conf.CollectorID())
				}
				for _, conf := range srvs {
					if conf.Alive() {
						oldCollectorID = append(oldCollectorID, conf.CollectorID())
					}
				}

				for idx := range newConf.Collector {
					// when exist, reload
					if slices.Contains(oldCollectorID, newConf.Collector[idx].CollectorID()) {
						for oldIdx := range srvs {
							if newConf.Collector[idx].CollectorID() == srvs[oldIdx].CollectorID() && srvs[oldIdx].Alive() {
								slog.Info("Reload", slog.String("detail", newConf.Collector[idx].CollectorID()))
								srvs[oldIdx].Reload(newConf.Collector[idx])
							}
						}
					} else {
						// create
						var workers = []serveAndShutdown{
							worker.New(ticker.MetadataNew(newConf.Collector[idx], client), 3*time.Hour),
							worker.New(ticker.New(newConf.Collector[idx], queueHandler), time.Minute),
						}
						for idx := range workers {
							go func() {
								if err := workers[idx].Serve(); err != nil {
									slog.Warn("failed Serve", slog.String("error", err.Error()))
								}
							}()
							srvs = append(srvs, workers[idx])
						}
						slog.Info("Serve by reload", slog.String("detail", newConf.Collector[idx].CollectorID()))
					}
				}

				for current := range srvs {
					if srvs[current].Alive() &&
						srvs[current].CollectorID() != "" &&
						!slices.Contains(newCollectorID, srvs[current].CollectorID()) {
						slog.Info("Shutdown by reload", slog.String("detail", srvs[current].CollectorID()))
						go func() {

							if err := srvs[current].Shutdown(context.Background()); err != nil {
								slog.Warn("failed Shutdown", slog.String("error", err.Error()))
							}
						}()
					}
				}

				// if err == nil {
				// 	diff := cmp.Diff(conf, newConf, cmp.Comparer(func(x, y *regexp.Regexp) bool {
				// 		if x == nil || y == nil {
				// 			return x == y
				// 		}
				// 		return x.String() == y.String()
				// 	}))
				// 	slog.Info(diff)
				// }

				sdNotifyHelper(daemon.SdNotifyReady)
			}
		}
	}()

	sdNotifyHelper(daemon.SdNotifyReady)
}

func sdNotifyHelper(message string) {
	if _, err := daemon.SdNotify(false, message); err != nil {
		slog.Warn("failed send sd_notify", slog.String("error", err.Error()))
	}
}
