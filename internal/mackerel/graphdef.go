package mackerel

import "github.com/mackerelio/mackerel-client-go"

var graphDefs = []*mackerel.GraphDefsParam{
	{
		Name:        "custom.interface.ifInDiscards",
		Unit:        "integer",
		DisplayName: "In Discards",
		Metrics: []*mackerel.GraphDefsMetric{
			{
				Name:        "custom.interface.ifInDiscards.*",
				DisplayName: "%1",
			},
		},
	},
	{
		Name:        "custom.interface.ifOutDiscards",
		Unit:        "integer",
		DisplayName: "Out Discards",
		Metrics: []*mackerel.GraphDefsMetric{
			{
				Name:        "custom.interface.ifOutDiscards.*",
				DisplayName: "%1",
			},
		},
	},
	{
		Name:        "custom.interface.ifInErrors",
		Unit:        "integer",
		DisplayName: "In Errors",
		Metrics: []*mackerel.GraphDefsMetric{
			{
				Name:        "custom.interface.ifInErrors.*",
				DisplayName: "%1",
			},
		},
	},
	{
		Name:        "custom.interface.ifOutErrors",
		Unit:        "integer",
		DisplayName: "Out Errors",
		Metrics: []*mackerel.GraphDefsMetric{
			{
				Name:        "custom.interface.ifOutErrors.*",
				DisplayName: "%1",
			},
		},
	},
}
