package config

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"gopkg.in/yaml.v3"
)

type yamlCollectorConfig struct {
	HostID   string `yaml:"host-id"`
	HostName string `yaml:"hostname,omitempty"`

	// for snmp/conn
	Community string `yaml:"community"`
	Host      string `yaml:"host"`
	Port      uint16 `yaml:"port"`
	Version   string `yaml:"version"`
	Timeout   string `yaml:"timeout"`
	Retry     int    `yaml:"retry"`

	SNMPv3 *yamlCollectorConfigSNMPv3 `yaml:"snmpv3"`

	// for snmp/rule
	Interface    *yamlInterface `yaml:"interface,omitempty"`
	Mibs         []string       `yaml:"mibs,omitempty"`
	SkipLinkdown bool           `yaml:"skip-linkdown,omitempty"`
	CustomMibs   []*customMIB   `yaml:"custom-mibs,omitempty"`
}

type yamlConfig struct {
	ApiKey string `yaml:"x-api-key"`

	Collector []*yamlCollectorConfig `yaml:"collector"`
}

type yamlInterface struct {
	Include *string `yaml:"include,omitempty"`
	Exclude *string `yaml:"exclude,omitempty"`
}

type customMIB struct {
	DisplayName string                `yaml:"display-name"`
	Unit        string                `yaml:"unit"`
	Mibs        []*mibWithDisplayName `yaml:"mibs,omitempty"`
}

type mibWithDisplayName struct {
	DisplayName string `yaml:"display-name,omitempty"`
	MetricName  string `yaml:"metric-name"`
	MIB         string `yaml:"mib"`
}

type collectorSNMPConfigV2c struct {
	Community string
}

type CollectorSNMPConfig struct {
	Host    string
	Port    uint16
	Timeout time.Duration
	Retry   int

	V2c *collectorSNMPConfigV2c
	V3  *collectorSNMPConfigV3
}

type CollectorConfig struct {
	HostID   string
	HostName string

	// for snmp/conn
	SNMP CollectorSNMPConfig

	// for snmp/rule
	MIBs              []string
	IncludeRegexp     *regexp.Regexp
	ExcludeRegexp     *regexp.Regexp
	SkipDownLinkState bool

	CustomMIBs          []string
	CustomMIBsGraphDefs []*mackerel.GraphDefsParam
	// metricName:mib
	CustomMIBmetricNameMappedMIBs map[string]string
}

func (conf *CollectorConfig) CollectorID() string {
	return fmt.Sprintf("host=%s,port=%d,hostID=%s", conf.SNMP.Host, conf.SNMP.Port, conf.HostID)
}

type Config struct {
	ApiKey string

	Collector []*CollectorConfig
}

func Init(filename string) (*Config, error) {
	f, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var t yamlConfig
	err = yaml.Unmarshal(f, &t)
	if err != nil {
		return nil, err
	}
	return convert(t)
}

func convert(t yamlConfig) (*Config, error) {
	apiKey := cmp.Or(os.Getenv("MACKEREL_APIKEY"), t.ApiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("x-api-key is needed")
	}

	var cs []*CollectorConfig
	for i := range t.Collector {
		conf, err := convertCollector(t.Collector[i])
		if err != nil {
			slog.Warn("skipped because failed parse config", slog.Int("index", i), slog.String("error", err.Error()))
			continue
		}
		cs = append(cs, conf)
	}

	return &Config{
		ApiKey:    apiKey,
		Collector: cs,
	}, nil
}
