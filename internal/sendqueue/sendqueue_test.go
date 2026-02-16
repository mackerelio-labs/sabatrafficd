package sendqueue

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mackerelio/mackerel-client-go"
)

func TestSendQueue(t *testing.T) {
	t.Run("exist queue", func(t *testing.T) {
		tm := time.Now().Unix()
		q := New()

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

		actual := q.FrontN(100)
		expected := []Item{
			{
				Metrics: []*mackerel.MetricValue{
					{
						Name:  "name12345",
						Time:  tm,
						Value: 1.2345,
					},
				},
			}, {
				Metrics: []*mackerel.MetricValue{

					{
						Name:  "name12345678",
						Time:  tm,
						Value: 1.2345678,
					},
				},
			},
		}
		if diff := cmp.Diff(actual, expected); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
	})

	t.Run("chunk", func(t *testing.T) {
		tm := time.Now().Unix()
		q := New()

		queue := []*mackerel.MetricValue{}
		for range 101 {
			queue = append(queue, &mackerel.MetricValue{
				Name:  "name12345",
				Time:  tm,
				Value: 1.2345,
			})
		}
		q.Enqueue("", queue)

		if len(q.FrontN(100)) != 3 {
			t.Error("invalid. chunk")
		}
	})
}
