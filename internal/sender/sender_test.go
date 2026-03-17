package sender

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/mackerelio/mackerel-client-go"
)

type mockQueue struct {
	sync.Mutex
	count int
}

func (m *mockQueue) Dequeue() (hostid string, metrics []*mackerel.MetricValue, ok bool) {
	m.Lock()
	defer m.Unlock()
	if m.count > 0 {
		m.count--
		return "hostid", nil, true
	}
	return "", nil, false
}

func (m *mockQueue) Len() int {
	m.Lock()
	defer m.Unlock()
	return m.count
}

func (m *mockQueue) ReEnqueue(string, []*mackerel.MetricValue) {
	m.Lock()
	defer m.Unlock()
	m.count++
}

type mockSender struct {
	sync.Mutex
	count int
}

func (m *mockSender) Send(ctx context.Context, h string, _ []*mackerel.MetricValue) error {
	err := ctx.Err()
	if err != nil {
		slog.Info("ctx", slog.String("error", err.Error()))
		return err
	}
	time.Sleep(1000 * time.Millisecond)

	m.Lock()
	m.count++
	m.Unlock()

	return nil
}

func TestServe(t *testing.T) {
	m := &mockQueue{count: 30}
	s := &mockSender{}
	h := New(s, m)

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := h.Serve(); err != nil {
			t.Error(err)
		}
	})

	if err := h.Shutdown(t.Context()); err != nil {
		t.Error(err)
	}

	wg.Wait()

	if s.count != 30 {
		t.Error("invalid")
	}
}
