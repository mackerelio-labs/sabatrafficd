package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type ticker interface {
	Tick(context.Context)

	Reload(conf *config.CollectorConfig)
	CollectorID() string
}

type worker struct {
	wg         sync.WaitGroup
	shutdown   chan struct{}
	isShutdown atomic.Bool

	tick ticker
	d    time.Duration
}

func New(tick ticker, d time.Duration) *worker {
	return &worker{
		shutdown: make(chan struct{}),

		tick: tick,
		d:    d,
	}
}
func (w *worker) Serve() error {
	ticker := time.NewTicker(w.d)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan struct{})

	go func() {
		<-w.shutdown
		defer close(quit)

		cancel()
		ticker.Stop()

		slog.Debug("Serve stopped")
	}()

	for {
		w.wg.Add(1)
		defer w.wg.Done()

		w.tick.Tick(ctx)

		select {
		case <-ticker.C:
			continue
		case <-quit:
			return nil
		}
	}
}

func (w *worker) Shutdown(_ context.Context) error {
	if !w.isShutdown.CompareAndSwap(false, true) {
		return nil
	}

	close(w.shutdown)
	w.wg.Wait()
	return nil
}

func (w *worker) Reload(conf *config.CollectorConfig) {
	w.tick.Reload(conf)
}

func (w *worker) CollectorID() string {
	return w.tick.CollectorID()
}

func (w *worker) Alive() bool {
	return !w.isShutdown.Load()
}
