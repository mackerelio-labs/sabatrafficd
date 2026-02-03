package config

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mackerelio/mackerel-client-go"
)

func Test_generateCustomMIB(t *testing.T) {

	tests := []struct {
		source   *customMIB
		expected *customMIBConfig
		wantErr  bool
	}{
		{
			source: &customMIB{
				Mibs: []*mibWithDisplayName{
					{
						MetricName: "foo.bar",
						MIB:        "1.2.3.4",
					},
				},
			},
			expected: &customMIBConfig{
				customMIBs: []string{"1.2.3.4"},
				metricNameMappedMIBs: map[string]string{
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar": "1.2.3.4",
				},
				graphDefs: &mackerel.GraphDefsParam{
					Name: "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e",
					Metrics: []*mackerel.GraphDefsMetric{
						{
							DisplayName: "foo.bar",
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar",
						},
					},
				},
			},
		},
		{
			source: &customMIB{
				Mibs: []*mibWithDisplayName{
					{
						DisplayName: "foobarbaz",
						MetricName:  "foo.bar",
						MIB:         "1.2.3.4",
					},
				},
			},
			expected: &customMIBConfig{
				customMIBs: []string{"1.2.3.4"},
				metricNameMappedMIBs: map[string]string{
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar": "1.2.3.4",
				},
				graphDefs: &mackerel.GraphDefsParam{
					Name: "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e",
					Metrics: []*mackerel.GraphDefsMetric{
						{
							DisplayName: "foobarbaz",
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar",
						},
					},
				},
			},
		},
		{
			source: &customMIB{
				Mibs: []*mibWithDisplayName{
					{
						MetricName: "foo.bar",
						MIB:        "1.2.3.4",
					},
					{
						MetricName: "foo.baz",
						MIB:        "5.6.7.8",
					},
				},
			},
			expected: &customMIBConfig{
				customMIBs: []string{"1.2.3.4", "5.6.7.8"},
				metricNameMappedMIBs: map[string]string{
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar": "1.2.3.4",
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.baz": "5.6.7.8",
				},
				graphDefs: &mackerel.GraphDefsParam{
					Name: "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e",
					Metrics: []*mackerel.GraphDefsMetric{
						{
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar",
							DisplayName: "foo.bar",
						},
						{
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.baz",
							DisplayName: "foo.baz",
						},
					},
				},
			},
		},
		{
			source: &customMIB{
				Mibs: []*mibWithDisplayName{
					{
						MetricName: "foo.bar",
						MIB:        "1.2.3.4...",
					},
				},
			},
			wantErr: true,
		},
		{
			source: &customMIB{
				Mibs: []*mibWithDisplayName{
					{
						MetricName: "foo.あ.bar",
						MIB:        "1.2.3.4",
					},
				},
			},
			wantErr: true,
		},
	}

	opt := cmp.AllowUnexported(customMIBConfig{})
	for _, tc := range tests {
		actual, err := generateCustomMIB(tc.source)
		if (err != nil) != tc.wantErr {
			t.Error(err)
		}

		if diff := cmp.Diff(actual, tc.expected, opt); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
	}
}

func Test_convert(t *testing.T) {
	reg := "^(eth|wlan)"

	tests := []struct {
		source   yamlConfig
		expected *Config
		wantErr  bool
	}{
		{
			source:  yamlConfig{},
			wantErr: true,
		},
		{
			source: yamlConfig{
				Collector: []*yamlCollectorConfig{
					{
						Community: "public",
					},
				},
			},
			wantErr: true,
		},
		{
			source: yamlConfig{
				Collector: []*yamlCollectorConfig{
					{
						Host: "192.0.2.1",
					},
				},
			},
			wantErr: true,
		},
		{
			source: yamlConfig{
				ApiKey: "cat",
				Collector: []*yamlCollectorConfig{
					{
						HostID: "panda",

						Community: "public",
						Host:      "192.0.2.1",
						Port:      161,
					},
				},
			},
			expected: &Config{
				ApiKey: "cat",
				Collector: []*CollectorConfig{
					{
						HostID: "panda",

						SNMP: CollectorSNMPConfig{
							Host: "192.0.2.1",
							Port: 161,
							V2c: &collectorSNMPConfigV2c{
								Community: "public",
							},
						},

						MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets", "ifInDiscards", "ifOutDiscards", "ifInErrors", "ifOutErrors"},
						CustomMIBmetricNameMappedMIBs: map[string]string{},
					},
				},
			},
		},
		{
			source: yamlConfig{
				ApiKey: "cat",
				Collector: []*yamlCollectorConfig{
					{
						HostID: "panda",

						Community: "public",
						Host:      "192.0.2.1",
						Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
					},
				},
			},
			expected: &Config{
				ApiKey: "cat",
				Collector: []*CollectorConfig{
					{
						HostID: "panda",

						SNMP: CollectorSNMPConfig{
							Host: "192.0.2.1",
							Port: 161,
							V2c: &collectorSNMPConfigV2c{
								Community: "public",
							},
						},
						MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
						CustomMIBmetricNameMappedMIBs: map[string]string{},
					},
				},
			},
		},
		{
			source: yamlConfig{
				ApiKey: "cat",
				Collector: []*yamlCollectorConfig{
					{
						HostID: "panda",

						Community: "public",
						Host:      "192.0.2.1",
						Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
						Interface: &yamlInterface{
							Include: &reg,
						},
					},
				},
			},
			expected: &Config{
				ApiKey: "cat",
				Collector: []*CollectorConfig{
					{
						HostID: "panda",

						SNMP: CollectorSNMPConfig{
							Host: "192.0.2.1",
							Port: 161,
							V2c: &collectorSNMPConfigV2c{
								Community: "public",
							},
						},
						MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
						CustomMIBmetricNameMappedMIBs: map[string]string{},
						IncludeRegexp:                 regexp.MustCompile(reg),
					},
				},
			},
		},
		{
			source: yamlConfig{
				ApiKey: "cat",
				Collector: []*yamlCollectorConfig{
					{
						HostID: "panda",

						Community: "public",
						Host:      "192.0.2.1",
						Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
						Interface: &yamlInterface{
							Exclude: &reg,
						},
					},
				},
			},
			expected: &Config{
				ApiKey: "cat",
				Collector: []*CollectorConfig{
					{
						HostID: "panda",

						SNMP: CollectorSNMPConfig{
							Host: "192.0.2.1",
							Port: 161,
							V2c: &collectorSNMPConfigV2c{
								Community: "public",
							},
						},
						MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
						CustomMIBmetricNameMappedMIBs: map[string]string{},
						ExcludeRegexp:                 regexp.MustCompile(reg),
					},
				},
			},
		},
		{
			source: yamlConfig{
				Collector: []*yamlCollectorConfig{
					{
						Community: "public",
						Host:      "192.0.2.1",
						Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
						Interface: &yamlInterface{
							Include: &reg,
							Exclude: &reg,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			source: yamlConfig{
				Collector: []*yamlCollectorConfig{
					{
						Community: "public",
						Host:      "192.0.2.1",
						Mibs:      []string{"^o^"},
					},
				},
			},
			wantErr: true,
		},
		{
			source: yamlConfig{
				ApiKey: "cat",
				Collector: []*yamlCollectorConfig{
					{
						Community: "public",
						Host:      "192.0.2.1",
						Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},

						HostID:   "panda",
						HostName: "dog",
					},
				},
			},
			expected: &Config{
				ApiKey: "cat",
				Collector: []*CollectorConfig{
					{
						SNMP: CollectorSNMPConfig{
							Host: "192.0.2.1",
							Port: 161,
							V2c: &collectorSNMPConfigV2c{
								Community: "public",
							},
						},
						MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
						CustomMIBmetricNameMappedMIBs: map[string]string{},

						HostID:   "panda",
						HostName: "dog",
					},
				},
			},
		},
		{
			source: yamlConfig{
				ApiKey: "cat",
				Collector: []*yamlCollectorConfig{
					{
						HostID: "panda",

						Community: "public",
						Host:      "192.0.2.1",
						Port:      10161,
						Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
						CustomMibs: []*customMIB{
							{
								DisplayName: "zoo",
								Unit:        "float",
								Mibs: []*mibWithDisplayName{
									{
										DisplayName: "foobar",
										MetricName:  "foo.bar",
										MIB:         "1.2.34.56",
									},
								},
							},
						},
					},
				},
			},
			expected: &Config{
				ApiKey: "cat",
				Collector: []*CollectorConfig{
					{
						HostID: "panda",

						SNMP: CollectorSNMPConfig{
							Host: "192.0.2.1",
							Port: 10161,
							V2c: &collectorSNMPConfigV2c{
								Community: "public",
							},
						},
						MIBs: []string{"ifHCInOctets", "ifHCOutOctets"},
						CustomMIBmetricNameMappedMIBs: map[string]string{
							"custom.custommibs.d2cbe65f53da8607e64173c1a83394fe.foo.bar": "1.2.34.56",
						},
						CustomMIBs: []string{"1.2.34.56"},
						CustomMIBsGraphDefs: []*mackerel.GraphDefsParam{
							{
								Name:        "custom.custommibs.d2cbe65f53da8607e64173c1a83394fe",
								DisplayName: "zoo",
								Unit:        "float",
								Metrics: []*mackerel.GraphDefsMetric{
									{
										Name:        "custom.custommibs.d2cbe65f53da8607e64173c1a83394fe.foo.bar",
										DisplayName: "foobar",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	opt1 := cmpopts.SortSlices(func(i, j string) bool { return i < j })

	opt2 := cmp.Comparer(func(x, y *regexp.Regexp) bool {
		if x == nil || y == nil {
			return x == y
		}

		return fmt.Sprint(x) == fmt.Sprint(y)
	})

	for _, tc := range tests {
		actual, err := convert(tc.source)
		if (err != nil) != tc.wantErr {
			t.Error(err)
		}

		if diff := cmp.Diff(actual, tc.expected, opt1, opt2); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
	}
}

func Test_snmpProtocolVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{
			input:    "",
			expected: SNMPV2c,
			wantErr:  false,
		},
		{
			input:    "v1",
			expected: "",
			wantErr:  true,
		},
		{
			input:    "v2c",
			expected: SNMPV2c,
			wantErr:  false,
		},
		{
			input:    "foo",
			expected: "",
			wantErr:  true,
		},
	}
	for _, tc := range tests {
		actual, err := snmpProtocolVersion(tc.input)
		if (err != nil) != tc.wantErr {
			t.Error(err)
		}
		if actual != tc.expected {
			t.Errorf("invalid actual: %s, expected: %s", actual, tc.expected)
		}
	}
}
