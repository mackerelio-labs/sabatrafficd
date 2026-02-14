package sendqueue

import (
	"container/list"
	"context"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type sender interface {
	Send(context.Context, string, []*mackerel.MetricValue) error
}

type Queue struct {
	wg         sync.WaitGroup
	shutdown   chan struct{}
	isShutdown atomic.Bool

	sync.Mutex
	buffers  *list.List
	sendFunc sender
}

type noopSendFunc struct{}

func (noopSendFunc) Send(_ context.Context, _ string, _ []*mackerel.MetricValue) error {
	return nil
}

func New(sendFunc sender) *Queue {
	if sendFunc == nil {
		sendFunc = &noopSendFunc{}
	}
	return &Queue{
		shutdown: make(chan struct{}),
		buffers:  list.New(),
		sendFunc: sendFunc,
	}
}

type Message struct {
	hostID  string
	metrics []*mackerel.MetricValue
}

func (q *Queue) len() int {
	q.Lock()
	defer q.Unlock()
	return q.buffers.Len()
}

func (q *Queue) Serve() error {
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
			if q.len() == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			e := q.buffers.Front()
			value := e.Value.(Message)

			// for idx := range value {
			// 	fmt.Printf("%d\t%s\t%v\n", value[idx].Time, value[idx].Name, value[idx].Value)
			// }

			if err := q.sendFunc.Send(ctx, value.hostID, value.metrics); err != nil {
				slog.WarnContext(ctx, "failed post", slog.String("error", err.Error()))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			q.Lock()
			q.buffers.Remove(e)
			q.Unlock()
		}
	}
}

func (q *Queue) Enqueue(hostID string, rawMetrics []*mackerel.MetricValue) {
	q.Lock()
	defer q.Unlock()
	// When a large item cannot be sent, the error never goes away.
	// Therefore, divide it into appropriate numbers.
	for chunk := range slices.Chunk(rawMetrics, 50) {
		q.buffers.PushBack(Message{hostID: hostID, metrics: chunk})
	}
}

func (*Queue) Reload(conf *config.CollectorConfig) {
	// no support
}

func (*Queue) CollectorID() string {
	// no support
	return ""
}
func (q *Queue) Alive() bool {
	return !q.isShutdown.Load()
}

func (q *Queue) Shutdown(ctx context.Context) error {
	if !q.isShutdown.CompareAndSwap(false, true) {
		return nil
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			if len := q.len(); len > 0 {
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
