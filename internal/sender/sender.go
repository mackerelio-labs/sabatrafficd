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
	wg         sync.WaitGroup
	shutdown   chan struct{}
	isShutdown atomic.Bool

	queue    queue
	sendFunc sendFunc
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan struct{})

	go func() {
		<-q.shutdown
		defer close(quit)

		cancel()

		slog.Debug("Serve stopped")
	}()


	q.wg.Add(1)
	defer q.wg.Done()
	for {
		select {
		case <-quit:
			return nil
		default:
			hostID, metrics, ok := q.queue.Dequeue()
			if !ok {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// for idx := range value {
			// 	fmt.Printf("%d\t%s\t%v\n", value[idx].Time, value[idx].Name, value[idx].Value)
			// }

			if err := q.sendFunc.Send(ctx, hostID, metrics); err != nil {
				slog.WarnContext(ctx, "failed post", slog.String("error", err.Error()))
				q.queue.ReEnqueue(hostID, metrics)
				time.Sleep(100 * time.Millisecond)
				continue
			}
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
	q.wg.Wait()
	return nil
}
