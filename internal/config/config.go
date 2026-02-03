package config

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"

	"github.com/mackerelio/mackerel-client-go"
	"gopkg.in/yaml.v3"

	"github.com/mackerelio-labs/sabatrafficd/internal/mib"
)

type yamlCollectorConfig struct {
	HostID   string `yaml:"host-id"`
	HostName string `yaml:"hostname,omitempty"`

	// for snmp/conn
	Community string `yaml:"community"`
	Host      string `yaml:"host"`
	Port      uint16 `yaml:"port"`
	Version   string `yaml:"version"`

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
	Host string
	Port uint16

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
			slog.Warn("skipped because failed parse config", slog.Int("index", i))
			continue
		}
		cs = append(cs, conf)
	}

	return &Config{
		ApiKey:    apiKey,
		Collector: cs,
	}, nil
}

const (
	SNMPV2c = "SNMPv2c"
	SNMPV3  = "SNMPv3"
)

func snmpProtocolVersion(v string) (string, error) {
	switch v {
	case "":
		return SNMPV2c, nil
	case "v2c":
		return SNMPV2c, nil
	case "v3":
		return SNMPV3, nil
	}
	return "", fmt.Errorf("invalid snmp protocol version (v2c, v3) : %s", v)
}

func convertCollector(t *yamlCollectorConfig) (*CollectorConfig, error) {
	if t.Host == "" {
		return nil, fmt.Errorf("host is needed")
	}
	if t.HostID == "" {
		return nil, fmt.Errorf("host-id is needed")
	}

	snmpConfig := CollectorSNMPConfig{
		Host: t.Host,
		Port: cmp.Or(t.Port, 161),
	}

	version, err := snmpProtocolVersion(t.Version)
	if err != nil {
		return nil, err
	}
	if version == SNMPV2c {
		if t.Community == "" {
			return nil, fmt.Errorf("community is needed")
		}
		snmpConfig.V2c = &collectorSNMPConfigV2c{
			Community: t.Community,
		}
	}
	if version == SNMPV3 {
		if t.SNMPv3 == nil {
			return nil, fmt.Errorf("snmpv3 not found")
		}
		if ok := parseSeurity(t.SNMPv3.SecLevel); !ok {
			return nil, fmt.Errorf("snmpv3.security is invalid : %s", t.SNMPv3.SecLevel)
		}
		if ok := parseAuthenticationProtocol(t.SNMPv3.AuthenticationProtocol); !ok {
			return nil, fmt.Errorf("snmpv3.auth-protocol is invalid : %s", t.SNMPv3.AuthenticationProtocol)
		}
		if ok := parsePrivacyProtocol(t.SNMPv3.PrivacyProtocol); !ok {
			return nil, fmt.Errorf("snmpv3.priv-protocol is invalid : %s", t.SNMPv3.PrivacyProtocol)
		}
		snmpConfig.V3 = &collectorSNMPConfigV3{
			secLevel:                 t.SNMPv3.SecLevel,
			usename:                  t.SNMPv3.UserName,
			authenticationProtocol:   t.SNMPv3.AuthenticationProtocol,
			authenticationPassphrase: t.SNMPv3.AuthenticationPassphrase,
			privacyProtocol:          t.SNMPv3.PrivacyProtocol,
			privacyPassphrase:        t.SNMPv3.PrivacyPassphrase,
		}
	}

	c := &CollectorConfig{
		HostID:   t.HostID,
		HostName: t.HostName,

		SNMP: snmpConfig,

		SkipDownLinkState:             t.SkipLinkdown,
		CustomMIBmetricNameMappedMIBs: map[string]string{},
	}

	if t.Interface != nil {
		if t.Interface.Include != nil && t.Interface.Exclude != nil {
			return nil, fmt.Errorf("Interface.Exclude, Interface.Include is exclusive control")
		}
		if t.Interface.Include != nil {
			c.IncludeRegexp, err = regexp.Compile(*t.Interface.Include)
			if err != nil {
				return nil, err
			}
		}
		if t.Interface.Exclude != nil {
			c.ExcludeRegexp, err = regexp.Compile(*t.Interface.Exclude)
			if err != nil {
				return nil, err
			}
		}
	}

	c.MIBs, err = mib.Validate(t.Mibs)
	if err != nil {
		return nil, err
	}
	// Reload 処理で差分を抑制するためのソート
	slices.Sort(c.MIBs)

	for i := range t.CustomMibs {
		res, err := generateCustomMIB(t.CustomMibs[i])
		if err != nil {
			return nil, err
		}
		c.CustomMIBs = append(c.CustomMIBs, res.customMIBs...)
		c.CustomMIBsGraphDefs = append(c.CustomMIBsGraphDefs, res.graphDefs)
		for metricName, mib := range res.metricNameMappedMIBs {
			c.CustomMIBmetricNameMappedMIBs[metricName] = mib
		}
	}
	return c, nil
}

var metricRe = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

func customMIBMackerelMetricNameParent(graphDisplayName string) string {
	a := md5.Sum([]byte(graphDisplayName))
	return fmt.Sprintf("custom.custommibs.%x", a)
}

func customMIBMackerelMetricName(graphDisplayName, metricName string) string {
	return fmt.Sprintf("%s.%s", customMIBMackerelMetricNameParent(graphDisplayName), metricName)
}

type customMIBConfig struct {
	customMIBs []string

	// metricName:MIB
	metricNameMappedMIBs map[string]string

	graphDefs *mackerel.GraphDefsParam
}

func generateCustomMIB(t *customMIB) (*customMIBConfig, error) {
	var customMIBs []string
	var metrics []*mackerel.GraphDefsMetric
	var metricNameMappedMIBs = make(map[string]string, 0)

	for idx := range t.Mibs {
		metricName := t.Mibs[idx].MetricName
		if !metricRe.MatchString(metricName) {
			return nil, fmt.Errorf("metricName is not valid : %s", metricName)
		}

		mackerelMetricName := customMIBMackerelMetricName(t.DisplayName, metricName)
		metrics = append(metrics, &mackerel.GraphDefsMetric{
			Name:        mackerelMetricName,
			DisplayName: cmp.Or(t.Mibs[idx].DisplayName, t.Mibs[idx].MetricName),
		})

		err := mib.ValidateCustom(t.Mibs[idx].MIB)
		if err != nil {
			return nil, err
		}
		customMIBs = append(customMIBs, t.Mibs[idx].MIB)

		metricNameMappedMIBs[mackerelMetricName] = t.Mibs[idx].MIB
	}

	return &customMIBConfig{
		graphDefs: &mackerel.GraphDefsParam{
			Name:        customMIBMackerelMetricNameParent(t.DisplayName),
			Unit:        t.Unit,
			DisplayName: t.DisplayName,
			Metrics:     metrics,
		},
		customMIBs:           customMIBs,
		metricNameMappedMIBs: metricNameMappedMIBs,
	}, nil
}
