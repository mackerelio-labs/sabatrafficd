package ticker

import (
	"cmp"
	"context"
	"log/slog"
	"reflect"
	"sync"

	"github.com/mackerelio-labs/sabatrafficd/internal/collector"
	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type updateHost interface {
	UpdateHost(ctx context.Context, hostID, hostAddr string, hostname string, ifs []collector.Interface) error
}

type MetadataTicker struct {
	mu sync.RWMutex

	conf   *config.CollectorConfig
	client updateHost

	// cache
	interfaces []collector.Interface
}

func MetadataNew(conf *config.CollectorConfig, m updateHost) *MetadataTicker {
	return &MetadataTicker{
		conf:   conf,
		client: m,

		interfaces: make([]collector.Interface, 0),
	}
}

func (t *MetadataTicker) Tick(ctx context.Context) {
	interfaces, err := collector.New(t.conf).DoInterfaceIPAddress(ctx)
	if err != nil {
		slog.WarnContext(ctx, "failed getting interfaces", slog.String("error", err.Error()))
	}

	if reflect.DeepEqual(t.interfaces, interfaces) {
		slog.InfoContext(ctx, "skip update metadata")
		return
	}
	t.interfaces = interfaces

	if err := t.client.UpdateHost(ctx, t.conf.HostID, t.conf.Host, cmp.Or(t.conf.HostName, t.conf.Host), interfaces); err != nil {
		slog.WarnContext(ctx, "failed UpdateHost", slog.String("error", err.Error()))
	}
}

func (t *MetadataTicker) Reload(conf *config.CollectorConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.conf = conf
}

func (t *MetadataTicker) CollectorID() string {
	return t.conf.CollectorID()
}
