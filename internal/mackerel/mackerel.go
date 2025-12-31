package mackerel

import (
	"cmp"
	"context"
	"os"

	mackerel "github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/collector"
)

type mackerelClient interface {
	UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error)
	CreateGraphDefs(payloads []*mackerel.GraphDefsParam) error
	PostHostMetricValuesByHostID(hostID string, metricValues []*mackerel.MetricValue) error
}

type Mackerel struct {
	client mackerelClient
}

func New(apikey string) *Mackerel {
	baseURL := cmp.Or(os.Getenv("MACKEREL_APIBASE"), "https://api.mackerelio.com/")
	client, _ := mackerel.NewClientWithOptions(apikey, baseURL, false)
	return &Mackerel{
		client: client,
	}
}

func (m *Mackerel) UpdateHost(ctx context.Context, hostID, hostAddr, hostname string, ifs []collector.Interface) error {
	var interfaces []mackerel.Interface

	if len(ifs) == 0 {
		interfaces = append(interfaces, mackerel.Interface{
			Name:          "main",
			IPv4Addresses: []string{hostAddr},
		})
	} else {
		for i := range ifs {
			interfaces = append(interfaces, mackerel.Interface{
				Name:          ifs[i].IfName,
				IPv4Addresses: ifs[i].IpAddress,
				MacAddress:    ifs[i].MacAddress,
			})
		}
	}

	_, err := m.client.UpdateHost(hostID, &mackerel.UpdateHostParam{
		Name:       hostname,
		Interfaces: interfaces,
	})
	if err != nil {
		return err
	}

	if err = m.CreateGraphDefs(ctx, graphDefs); err != nil {
		return err
	}
	return nil
}

func (m *Mackerel) CreateGraphDefs(ctx context.Context, d []*mackerel.GraphDefsParam) error {
	return m.client.CreateGraphDefs(d)
}

func (m *Mackerel) Send(ctx context.Context, hostID string, value []*mackerel.MetricValue) error {
	return m.client.PostHostMetricValuesByHostID(hostID, value)
}
