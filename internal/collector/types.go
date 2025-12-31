package collector

import "fmt"

type MetricsDutum struct {
	IfIndex uint64 `json:"ifIndex"`
	Mib     string `json:"mib"`
	IfName  string `json:"ifName"`
	Value   uint64 `json:"value"`
}

func (m *MetricsDutum) String() string {
	return fmt.Sprintf("%d\t%s\t%s\t%d", m.IfIndex, m.IfName, m.Mib, m.Value)
}

type Interface struct {
	IfName     string
	IpAddress  []string
	MacAddress string
}
