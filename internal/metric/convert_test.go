package metric

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/collector"
)

func compare[T any](t *testing.T, a, b T) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("invalid %v %v", a, b)
	}
}

func TestEscapeInterfaceName(t *testing.T) {
	compare(t, escapeInterfaceName("a/1.hello hello"), "a-1_hellohello")
}
func TestCalcurateDiff(t *testing.T) {
	compare(t, calcurateDiff(1, 2, 4), 1)
	compare(t, calcurateDiff(2, 2, 4), 0)
	compare(t, calcurateDiff(3, 2, 4), 3)
	compare(t, calcurateDiff(4, 2, 4), 2)
	compare(t, calcurateDiff(5, 2, 4), 1)
}

func Test_convert(t *testing.T) {
	now := time.Now()
	lastExecution := now.Add(-time.Minute)
	prevSnapshot := []collector.MetricsDutum{
		{
			IfIndex: 1,
			Mib:     "ifHCInOctets",
			IfName:  "eth0",
			Value:   1,
		},
		{
			IfIndex: 1,
			Mib:     "ifHCOutOctets",
			IfName:  "eth0",
			Value:   math.MaxUint64,
		},
		{
			IfIndex: 1,
			Mib:     "ifInDiscards",
			IfName:  "eth0",
			Value:   0,
		},
	}

	actual := convert([]collector.MetricsDutum{
		{
			IfIndex: 1,
			Mib:     "ifHCInOctets",
			IfName:  "eth0",
			Value:   1,
		},
		{
			IfIndex: 1,
			Mib:     "ifHCOutOctets",
			IfName:  "eth0",
			Value:   60,
		},
		{
			IfIndex: 1,
			Mib:     "ifInDiscards",
			IfName:  "eth0",
			Value:   1,
		},
	}, prevSnapshot, now, lastExecution)

	expected := []*mackerel.MetricValue{
		{
			Name:  "interface.eth0.rxBytes.delta",
			Time:  time.Now().Unix(),
			Value: uint64(0),
		},
		{
			Name:  "interface.eth0.txBytes.delta",
			Time:  time.Now().Unix(),
			Value: uint64(1),
		},
		{
			Name:  "custom.interface.ifInDiscards.eth0",
			Time:  time.Now().Unix(),
			Value: uint64(1),
		},
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("value is mismatch (-actual +expected):%s", diff)
	}
}

func Test_convert_scenario(t *testing.T) {
	now := time.Now()
	lastExecution := now.Add(-time.Minute)

	tests := []struct {
		input    []collector.MetricsDutum
		expected []*mackerel.MetricValue
	}{
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCInOctets",
					IfName:  "eth0",
					Value:   0,
				},
			},
			expected: []*mackerel.MetricValue{},
		},
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCInOctets",
					IfName:  "eth0",
					Value:   60,
				},
				{
					IfIndex: 1,
					Mib:     "ifHCOutOctets",
					IfName:  "eth0",
					Value:   60,
				},
			},
			expected: []*mackerel.MetricValue{
				{
					Name: "interface.eth0.rxBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
			},
		},
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCInOctets",
					IfName:  "eth0",
					Value:   120,
				},
				{
					IfIndex: 1,
					Mib:     "ifHCOutOctets",
					IfName:  "eth0",
					Value:   120,
				},
			},
			expected: []*mackerel.MetricValue{
				{
					Name: "interface.eth0.rxBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
				{
					Name: "interface.eth0.txBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
			},
		},
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCInOctets",
					IfName:  "eth0",
					Value:   180,
				},
				{
					IfIndex: 1,
					Mib:     "ifHCOutOctets",
					IfName:  "eth0",
					Value:   180,
				},
			},
			expected: []*mackerel.MetricValue{
				{
					Name: "interface.eth0.rxBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
				{
					Name: "interface.eth0.txBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
			},
		},
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCOutOctets",
					IfName:  "eth0",
					Value:   240,
				},
			},
			expected: []*mackerel.MetricValue{
				{
					Name: "interface.eth0.txBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
			},
		},
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCInOctets",
					IfName:  "eth0",
					Value:   300,
				},
				{
					IfIndex: 1,
					Mib:     "ifHCOutOctets",
					IfName:  "eth0",
					Value:   300,
				},
			},
			expected: []*mackerel.MetricValue{
				{
					Name: "interface.eth0.txBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
			},
		},
		{
			input: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "ifHCInOctets",
					IfName:  "eth0",
					Value:   360,
				},
				{
					IfIndex: 1,
					Mib:     "ifHCOutOctets",
					IfName:  "eth0",
					Value:   360,
				},
			},
			expected: []*mackerel.MetricValue{
				{
					Name: "interface.eth0.rxBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
				{
					Name: "interface.eth0.txBytes.delta", Time: time.Now().Unix(), Value: uint64(1),
				},
			},
		},
	}

	for i := range tests {
		var prevSnapshot []collector.MetricsDutum
		if i > 0 {
			prevSnapshot = tests[i-1].input
		}
		actual := convert(tests[i].input, prevSnapshot, now, lastExecution)

		if diff := cmp.Diff(actual, tests[i].expected); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
	}
}
