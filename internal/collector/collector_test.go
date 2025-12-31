package collector

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type mockSnmpClient struct {
}

var errInvalid = errors.New("invalid error")

func (m *mockSnmpClient) BulkWalk(oid string, length uint64) (map[uint64]uint64, error) {
	switch oid {
	case "1.3.6.1.2.1.31.1.1.1.6":
		return map[uint64]uint64{
			1: 60,
			2: 60,
			3: 60,
			4: 60,
		}, nil
	case "1.3.6.1.2.1.31.1.1.1.10":
		return map[uint64]uint64{
			1: 120,
			2: 120,
			3: 120,
			4: 120,
		}, nil
	default:
		return nil, errInvalid
	}
}
func (m *mockSnmpClient) BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error) {
	return map[uint64]string{
		1: "lo0",
		2: "eth0",
		3: "eth1",
		4: "eth2",
	}, nil
}
func (m *mockSnmpClient) BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error) {
	return map[uint64]bool{
		1: true,
		2: false,
		3: true,
		4: true,
	}, nil
}
func (m *mockSnmpClient) Close() error {
	return nil
}
func (m *mockSnmpClient) GetInterfaceNumber() (uint64, error) {
	return 4, nil
}

func (m *mockSnmpClient) BulkWalkGetInterfaceIPAddress() (map[uint64][]string, error) {
	return map[uint64][]string{
		1: {"127.0.0.1"},
		2: {"192.0.2.1"},
		3: {"192.0.2.2", "192.0.2.3"},
		4: {"198.51.100.1"},
		5: {"198.51.100.2"},
	}, nil
}
func (m *mockSnmpClient) BulkWalkGetInterfacePhysAddress(length uint64) (map[uint64]string, error) {
	return map[uint64]string{
		2: "00:00:87:12:34:56",
		3: "00:00:4C:23:45:67",
		4: "00:00:0E:34:56:78",
	}, nil
}

func (m *mockSnmpClient) GetValues(mibs []string) ([]float64, error) {
	var values []float64
	for idx := range mibs {
		sp := strings.Split(mibs[idx], ".")
		v, _ := strconv.ParseFloat(sp[len(sp)-1], 64)
		values = append(values, v)
	}
	return values, nil
}

func TestDo(t *testing.T) {
	t.Run("non skip", func(t *testing.T) {
		conf := &config.CollectorConfig{
			MIBs: []string{"ifHCInOctets", "ifHCOutOctets"},
		}
		actual, err := do(t.Context(), &mockSnmpClient{}, conf)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 1, Mib: "ifHCInOctets", IfName: "lo0", Value: 60},
			{IfIndex: 2, Mib: "ifHCInOctets", IfName: "eth0", Value: 60},
			{IfIndex: 3, Mib: "ifHCInOctets", IfName: "eth1", Value: 60},
			{IfIndex: 4, Mib: "ifHCInOctets", IfName: "eth2", Value: 60},
			{IfIndex: 1, Mib: "ifHCOutOctets", IfName: "lo0", Value: 120},
			{IfIndex: 2, Mib: "ifHCOutOctets", IfName: "eth0", Value: 120},
			{IfIndex: 3, Mib: "ifHCOutOctets", IfName: "eth1", Value: 120},
			{IfIndex: 4, Mib: "ifHCOutOctets", IfName: "eth2", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})

	t.Run("skip include", func(t *testing.T) {
		conf := &config.CollectorConfig{
			MIBs:          []string{"ifHCInOctets", "ifHCOutOctets"},
			IncludeRegexp: regexp.MustCompile("lo?"),
		}
		actual, err := do(t.Context(), &mockSnmpClient{}, conf)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 1, Mib: "ifHCInOctets", IfName: "lo0", Value: 60},
			{IfIndex: 1, Mib: "ifHCOutOctets", IfName: "lo0", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})
	t.Run("skip exclude", func(t *testing.T) {
		conf := &config.CollectorConfig{
			MIBs:          []string{"ifHCInOctets", "ifHCOutOctets"},
			ExcludeRegexp: regexp.MustCompile("0$"),
		}
		actual, err := do(t.Context(), &mockSnmpClient{}, conf)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 3, Mib: "ifHCInOctets", IfName: "eth1", Value: 60},
			{IfIndex: 4, Mib: "ifHCInOctets", IfName: "eth2", Value: 60},
			{IfIndex: 3, Mib: "ifHCOutOctets", IfName: "eth1", Value: 120},
			{IfIndex: 4, Mib: "ifHCOutOctets", IfName: "eth2", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})

	t.Run("skip down-linkstate", func(t *testing.T) {
		conf := &config.CollectorConfig{
			MIBs:              []string{"ifHCInOctets", "ifHCOutOctets"},
			SkipDownLinkState: true,
		}
		actual, err := do(t.Context(), &mockSnmpClient{}, conf)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 1, Mib: "ifHCInOctets", IfName: "lo0", Value: 60},
			{IfIndex: 3, Mib: "ifHCInOctets", IfName: "eth1", Value: 60},
			{IfIndex: 4, Mib: "ifHCInOctets", IfName: "eth2", Value: 60},
			{IfIndex: 1, Mib: "ifHCOutOctets", IfName: "lo0", Value: 120},
			{IfIndex: 3, Mib: "ifHCOutOctets", IfName: "eth1", Value: 120},
			{IfIndex: 4, Mib: "ifHCOutOctets", IfName: "eth2", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})

}

func TestDoInterfaceIPAddress(t *testing.T) {
	conf := &config.CollectorConfig{}
	actual, err := doInterfaceIPAddress(t.Context(), &mockSnmpClient{}, conf)
	if err != nil {
		t.Error("invalid raised error")
	}
	expected := []Interface{
		{
			IfName:     "eth0",
			IpAddress:  []string{"192.0.2.1"},
			MacAddress: "00:00:87:12:34:56",
		},
		{
			IfName:     "eth1",
			IpAddress:  []string{"192.0.2.2", "192.0.2.3"},
			MacAddress: "00:00:4C:23:45:67",
		},
		{
			IfName:     "eth2",
			IpAddress:  []string{"198.51.100.1"},
			MacAddress: "00:00:0E:34:56:78",
		},
		{
			IfName:    "lo0",
			IpAddress: []string{"127.0.0.1"},
		},
	}
	if d := cmp.Diff(
		actual,
		expected,
		cmpopts.SortSlices(func(i, j Interface) bool { return i.IfName < j.IfName }),
	); d != "" {
		t.Errorf("invalid result %s", d)
	}
}

func TestDoCustomMIBs(t *testing.T) {
	conf := &config.CollectorConfig{
		CustomMIBs: []string{"1.2.3.4.5.678901", "1.2.3.4.6.789012"},
	}
	actual, err := doCustomMIBs(t.Context(), &mockSnmpClient{}, conf)
	if err != nil {
		t.Error("invalid raised error")
	}
	expected := map[string]float64{
		"1.2.3.4.5.678901": 678901,
		"1.2.3.4.6.789012": 789012,
	}
	if d := cmp.Diff(
		actual,
		expected,
	); d != "" {
		t.Errorf("invalid result %s", d)
	}
}
