package metric

import (
	"time"

	"github.com/mackerelio/mackerel-client-go"
)

type Custom struct {
	mapping map[string]string
}

func NewCustom(mapping map[string]string) *Custom {
	return &Custom{mapping: mapping}
}

func (c *Custom) Convert(resp map[string]float64) []*mackerel.MetricValue {
	now := time.Now().Unix()

	metrics := make([]*mackerel.MetricValue, 0)
	for metricName, mib := range c.mapping {
		if f, ok := resp[mib]; ok {
			metrics = append(metrics, &mackerel.MetricValue{
				Name:  metricName,
				Time:  now,
				Value: f,
			})
		}
	}
	if len(metrics) > 0 {
		return metrics
	}
	return nil
}
