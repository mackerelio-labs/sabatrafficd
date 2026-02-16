package sendqueue

import (
	"container/list"
	"slices"
	"sync"

	"github.com/mackerelio/mackerel-client-go"
)

type Queue struct {
	mu sync.Mutex

	buffers *list.List
}

func New() *Queue {
	return &Queue{
		buffers: list.New(),
	}
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.buffers.Len()
}

type Item struct {
	HostID  string
	Metrics []*mackerel.MetricValue
}

func (q *Queue) Enqueue(hostID string, rawMetrics []*mackerel.MetricValue) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for chunk := range slices.Chunk(rawMetrics, 50) {
		q.buffers.PushBack(Item{HostID: hostID, Metrics: chunk})
	}
}
func (q *Queue) ReEnqueue(hostID string, rawMetrics []*mackerel.MetricValue) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.buffers.PushFront(Item{HostID: hostID, Metrics: rawMetrics})
}

func (q *Queue) Dequeue() (hostID string, rawMetrics []*mackerel.MetricValue, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	e := q.buffers.Front()
	if e == nil {
		return "", nil, false
	}
	q.buffers.Remove(e)

	value := e.Value.(Item)
	return value.HostID, value.Metrics, true
}

func (q *Queue) FrontN(length int) (items []Item) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for range length {
		e := q.buffers.Front()
		if e == nil {
			continue
		}
		items = append(items, e.Value.(Item))
		q.buffers.Remove(e)
	}

	return
}
