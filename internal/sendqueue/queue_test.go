package sendqueue

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mackerelio/mackerel-client-go"
)

func TestNew(t *testing.T) {
	q := New(nil)
	q.sendFunc.Send(t.Context(), "", nil) // nolint
}

type mockSendFunc struct {
	count  int
	values [][]*mackerel.MetricValue
}

func (m *mockSendFunc) Send(_ context.Context, _ string, v []*mackerel.MetricValue) error {
	m.count++
	m.values = append(m.values, v)
	return nil
}

func TestServe(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		mock := &mockSendFunc{}
		q := New(mock)

		go q.Serve() // nolint
		q.Shutdown(t.Context())

		if mock.count != 0 {
			t.Error("invalid. called Send()")
		}
	})

	t.Run("exist queue", func(t *testing.T) {
		tm := time.Now().Unix()
		mock := &mockSendFunc{}
		q := New(mock)

		q.Enqueue("", []*mackerel.MetricValue{
			{
				Name:  "name12345",
				Time:  tm,
				Value: 1.2345,
			},
		})
		q.Enqueue("", []*mackerel.MetricValue{
			{
				Name:  "name12345678",
				Time:  tm,
				Value: 1.2345678,
			},
		})

		go q.Serve() // nolint
		q.Shutdown(t.Context())

		actual := mock.values
		expected := [][]*mackerel.MetricValue{
			{
				{
					Name:  "name12345",
					Time:  tm,
					Value: 1.2345,
				},
			}, {
				{
					Name:  "name12345678",
					Time:  tm,
					Value: 1.2345678,
				},
			},
		}
		if diff := cmp.Diff(actual, expected); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
		if mock.count != 2 {
			t.Error("invalid. called Send()")
		}
	})

	t.Run("chunk", func(t *testing.T) {
		tm := time.Now().Unix()
		mock := &mockSendFunc{}
		q := New(mock)

		queue := []*mackerel.MetricValue{}
		for range 101 {
			queue = append(queue, &mackerel.MetricValue{
				Name:  "name12345",
				Time:  tm,
				Value: 1.2345,
			})
		}
		q.Enqueue("", queue)

		go q.Serve() // nolint
		q.Shutdown(t.Context())

		if len(mock.values) != 3 {
			t.Error("invalid. chunk")
		}
		if mock.count != 3 {
			t.Error("invalid. called Send()")
		}
	})
}
