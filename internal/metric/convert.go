package metric

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/collector"
)

type Converter struct {
	prevSnapshot  []collector.MetricsDutum
	lastExecution time.Time
}

func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Convert(rawMetrics []collector.MetricsDutum, now time.Time) []*mackerel.MetricValue {
	defer func() {
		c.prevSnapshot = rawMetrics
		c.lastExecution = now
	}()

	if len(c.prevSnapshot) == 0 {
		return nil
	}
	return convert(rawMetrics, c.prevSnapshot, now, c.lastExecution)
}

func convert(rawMetrics, prevSnapshot []collector.MetricsDutum, now, lastExecution time.Time) []*mackerel.MetricValue {
	metrics := make([]*mackerel.MetricValue, 0)
	for _, metric := range rawMetrics {
		prevValue := metric.Value
		var found bool
		for _, v := range prevSnapshot {
			if v.IfIndex == metric.IfIndex && v.Mib == metric.Mib {
				prevValue = v.Value
				found = true
				break
			}
		}
		// はじめて追加された値は、0として戻り値に追加するのは誤りなので、スキップする
		if !found {
			continue
		}

		value := calcurateDiff(prevValue, metric.Value, overflowValue(metric.Mib))

		var name string
		ifName := escapeInterfaceName(metric.IfName)
		if deltaValues(metric.Mib) {
			direction := "txBytes"
			if receiveDirection(metric.Mib) {
				direction = "rxBytes"
			}
			name = fmt.Sprintf("interface.%s.%s.delta", ifName, direction)
			value /= uint64(now.Sub(lastExecution).Seconds())
		} else {
			name = fmt.Sprintf("custom.interface.%s.%s", metric.Mib, ifName)
		}
		metrics = append(metrics, &mackerel.MetricValue{
			Name:  name,
			Time:  now.Unix(),
			Value: value,
		})
	}
	return metrics
}

func escapeInterfaceName(ifName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(ifName, "/", "-"), ".", "_"), " ", "")
}

func overflowValue(mib string) uint64 {
	if mib == "ifInOctets" || mib == "ifOutOctets" {
		return math.MaxUint32
	}
	return math.MaxUint64
}

func receiveDirection(mib string) bool {
	return (mib == "ifInOctets" || mib == "ifHCInOctets")
}

func deltaValues(mib string) bool {
	return mib == "ifInOctets" || mib == "ifOutOctets" || mib == "ifHCInOctets" || mib == "ifHCOutOctets"
}

func calcurateDiff(a, b, overflow uint64) uint64 {
	if b < a {
		return overflow - a + b
	} else {
		return b - a
	}
}
