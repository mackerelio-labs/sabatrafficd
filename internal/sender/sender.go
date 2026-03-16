package sender

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type sendFunc interface {
	Send(context.Context, string, []*mackerel.MetricValue) error
}

type queue interface {
	Dequeue() (hostid string, metrics []*mackerel.MetricValue, ok bool)
	Len() int
	ReEnqueue(string, []*mackerel.MetricValue)
}

type Sender struct {
	shutdown    chan struct{}
	isShutdown  atomic.Bool
	serveClosed atomic.Bool

	queue    queue
	sendFunc sendFunc
}

type item struct {
	hostID  string
	metrics []*mackerel.MetricValue
}

type noopSendFunc struct{}

func (noopSendFunc) Send(_ context.Context, _ string, _ []*mackerel.MetricValue) error {
	return nil
}

func New(sendFunc sendFunc, queue queue) *Sender {
	if sendFunc == nil {
		sendFunc = &noopSendFunc{}
	}
	return &Sender{
		shutdown: make(chan struct{}),
		queue:    queue,
		sendFunc: sendFunc,
	}
}

func (q *Sender) Serve() error {
	var wg sync.WaitGroup
	ch := make(chan *item, 100)

	for range 10 {
		wg.Go(func() {
			for v := range ch {
				if err := q.sendFunc.Send(context.Background(), v.hostID, v.metrics); err != nil {
					slog.Warn("failed post", slog.String("error", err.Error()))
					q.queue.ReEnqueue(v.hostID, v.metrics)
					time.Sleep(100 * time.Millisecond)
				}
			}
		})
	}

	for {
		select {
		case <-q.shutdown:
			slog.Debug("Serve stopped")
			// close(p.shutdown) が実行されている時点で、残存キューはないとされている
			close(ch)

			// ch の残存ジョブが全て捌けるまで待つ
			wg.Wait()
			q.serveClosed.Store(true)
			return nil
		default:
			hostID, metrics, ok := q.queue.Dequeue()
			if !ok {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			ch <- &item{hostID: hostID, metrics: metrics}
		}
	}
}

func (*Sender) Reload(conf *config.CollectorConfig) {
	// no support
}

func (*Sender) CollectorID() string {
	// no support
	return ""
}
func (q *Sender) Alive() bool {
	return !q.isShutdown.Load()
}

func (q *Sender) Shutdown(ctx context.Context) error {
	if !q.isShutdown.CompareAndSwap(false, true) {
		return nil
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			if len := q.queue.Len(); len > 0 {
				slog.InfoContext(ctx, "draining...", slog.Int("remain", len))
				time.Sleep(time.Second)
				continue
			}
			break loop
		}
	}

	close(q.shutdown)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if q.serveClosed.Load() {
				return nil
			}
		}
	}
}
