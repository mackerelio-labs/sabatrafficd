package ticker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/collector"
	"github.com/mackerelio-labs/sabatrafficd/internal/config"
	"github.com/mackerelio-labs/sabatrafficd/internal/metric"
)

type enqueuer interface {
	Enqueue(hostID string, rawMetrics []*mackerel.MetricValue)
}
type customConverter interface {
	Convert(resp map[string]float64) []*mackerel.MetricValue
}
type converter interface {
	Convert(rawMetrics []collector.MetricsDutum) []*mackerel.MetricValue
}

type collectorIface interface {
	Do(ctx context.Context) ([]collector.MetricsDutum, error)
	DoCustomMIBs(ctx context.Context) (map[string]float64, error)
	DoInterfaceIPAddress(ctx context.Context) ([]collector.Interface, error)
}

type Ticker struct {
	mu sync.RWMutex

	collectorID     string
	hostID          string
	queue           enqueuer
	customConverter customConverter
	converter       converter
	collector       collectorIface
}

func New(conf *config.CollectorConfig, q enqueuer) *Ticker {
	return &Ticker{
		collectorID:     conf.CollectorID(),
		hostID:          conf.HostID,
		queue:           q,
		customConverter: metric.NewCustom(conf.CustomMIBmetricNameMappedMIBs),
		converter:       metric.NewConverter(),
		collector:       collector.New(conf),
	}
}

func (t *Ticker) Tick(ctx context.Context) {
	t.do(ctx)
	t.doCustomMIBs(ctx)
}

func (t *Ticker) do(ctx context.Context) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	metrics, err := t.collector.Do(ctx)
	if err != nil {
		slog.WarnContext(ctx, "failed exec collector.Do()", slog.String("error", err.Error()))
		return
	}
	if m := t.converter.Convert(metrics); m != nil {
		t.queue.Enqueue(t.hostID, m)
	}
}

func (t *Ticker) doCustomMIBs(ctx context.Context) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	customMetrics, err := t.collector.DoCustomMIBs(ctx)
	if err != nil {
		slog.WarnContext(ctx, "failed exec collector.DoCustomMIBs()", slog.String("error", err.Error()))
		return
	}
	if m := t.customConverter.Convert(customMetrics); m != nil {
		t.queue.Enqueue(t.hostID, m)
	}
}

func (t *Ticker) Reload(conf *config.CollectorConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.hostID = conf.HostID
	t.customConverter = metric.NewCustom(conf.CustomMIBmetricNameMappedMIBs)
	t.collector = collector.New(conf)
}

func (t *Ticker) CollectorID() string {
	return t.collectorID
}
