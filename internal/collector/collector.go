package collector

import (
	"context"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
	"github.com/mackerelio-labs/sabatrafficd/internal/mib"
	"github.com/mackerelio-labs/sabatrafficd/internal/snmp"
)

type snmpClient interface {
	BulkWalk(oid string, length uint64) (map[uint64]uint64, error)
	BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error)
	BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error)
	BulkWalkGetInterfaceIPAddress() (map[uint64][]string, error)
	BulkWalkGetInterfacePhysAddress(length uint64) (map[uint64]string, error)
	Close() error
	GetInterfaceNumber() (uint64, error)
	GetValues(mibs []string) ([]float64, error)
}

type collector struct {
	conf *config.CollectorConfig
}

func New(conf *config.CollectorConfig) *collector {
	return &collector{conf: conf}
}

func (c *collector) Do(ctx context.Context) ([]MetricsDutum, error) {
	client, err := snmp.Connect(ctx, c.conf.SNMP)
	if err != nil {
		return nil, err
	}
	defer client.Close() // nolint
	return do(ctx, client, c.conf)
}

func do(_ context.Context, client snmpClient, conf *config.CollectorConfig) ([]MetricsDutum, error) {
	ifNumber, err := client.GetInterfaceNumber()
	if err != nil {
		return nil, err
	}
	ifDescr, err := client.BulkWalkGetInterfaceName(ifNumber)
	if err != nil {
		return nil, err
	}

	var ifOperStatus map[uint64]bool
	if conf.SkipDownLinkState {
		ifOperStatus, err = client.BulkWalkGetInterfaceState(ifNumber)
		if err != nil {
			return nil, err
		}
	}

	metrics := make([]MetricsDutum, 0)

	for _, mibName := range conf.MIBs {
		values, err := client.BulkWalk(mib.Oidmapping()[mibName], ifNumber)
		if err != nil {
			return nil, err
		}

		for ifIndex, value := range values {
			ifName := ifDescr[ifIndex]
			if conf.IncludeRegexp != nil && !conf.IncludeRegexp.MatchString(ifName) {
				continue
			}

			if conf.ExcludeRegexp != nil && conf.ExcludeRegexp.MatchString(ifName) {
				continue
			}

			// skip when down(2)
			if conf.SkipDownLinkState && !ifOperStatus[ifIndex] {
				continue
			}

			metrics = append(metrics, MetricsDutum{IfIndex: ifIndex, Mib: mibName, IfName: ifName, Value: value})
		}
	}
	return metrics, nil
}

func (c *collector) DoInterfaceIPAddress(ctx context.Context) ([]Interface, error) {
	client, err := snmp.Connect(ctx, c.conf.SNMP)
	if err != nil {
		return nil, err
	}
	defer client.Close() // nolint
	return doInterfaceIPAddress(ctx, client, c.conf)
}

func doInterfaceIPAddress(_ context.Context, client snmpClient, _ *config.CollectorConfig) ([]Interface, error) {
	ifNumber, err := client.GetInterfaceNumber()
	if err != nil {
		return nil, err
	}
	ifDescr, err := client.BulkWalkGetInterfaceName(ifNumber)
	if err != nil {
		return nil, err
	}

	ifIndexIP, err := client.BulkWalkGetInterfaceIPAddress()
	if err != nil {
		return nil, err
	}

	ifPhysAddress, err := client.BulkWalkGetInterfacePhysAddress(ifNumber)
	if err != nil {
		return nil, err
	}

	var interfaces []Interface
	for ifIndex, ip := range ifIndexIP {
		if name, ok := ifDescr[ifIndex]; ok {
			phy := ifPhysAddress[ifIndex]
			interfaces = append(interfaces, Interface{
				IfName:     name,
				IpAddress:  ip,
				MacAddress: phy,
			})
		}
	}

	return interfaces, nil
}

// mib:value
func (c *collector) DoCustomMIBs(ctx context.Context) (map[string]float64, error) {
	client, err := snmp.Connect(ctx, c.conf.SNMP)
	if err != nil {
		return nil, err
	}
	defer client.Close() // nolint
	return doCustomMIBs(ctx, client, c.conf)
}

// mib:value
func doCustomMIBs(_ context.Context, client snmpClient, conf *config.CollectorConfig) (map[string]float64, error) {
	values, err := client.GetValues(conf.CustomMIBs)
	if err != nil {
		return nil, err
	}
	var result = make(map[string]float64, 0)
	for idx := range values {
		result[conf.CustomMIBs[idx]] = values[idx]
	}
	return result, nil
}
