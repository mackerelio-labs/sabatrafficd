package mackerel

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/mackerelio-labs/sabatrafficd/internal/collector"
)

type mackerelClientMock struct {
	updateParam  mackerel.UpdateHostParam
	graphDef     []*mackerel.GraphDefsParam
	hostID       string
	metricValues []*mackerel.MetricValue

	returnHostID        string
	returnError         error
	returnErrorGraphDef error
}

func (m *mackerelClientMock) UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error) {
	m.updateParam = *param
	return m.returnHostID, m.returnError
}
func (m *mackerelClientMock) CreateGraphDefs(payloads []*mackerel.GraphDefsParam) error {
	m.graphDef = payloads
	return m.returnErrorGraphDef
}
func (m *mackerelClientMock) PostHostMetricValuesByHostID(hostID string, metricValues []*mackerel.MetricValue) error {
	m.hostID = hostID
	m.metricValues = metricValues
	return m.returnError
}

func TestInit(t *testing.T) {
	id := "1234567890"
	updateHost := mackerel.UpdateHostParam{
		Name: "hostname",
		Interfaces: []mackerel.Interface{
			{
				Name:          "main",
				IPv4Addresses: []string{"192.0.2.2"},
			},
		},
	}
	e := errors.New("error")
	tests := []struct {
		name                string
		expectedUpdateParam mackerel.UpdateHostParam
		expectedError       error
		expectedGraphDef    []*mackerel.GraphDefsParam
		hostID              string
		returnHostID        *string
		queue               *Mackerel
		mock                *mackerelClientMock
		interfaces          []collector.Interface
	}{
		{
			name:                "update host when hostID is exist",
			expectedUpdateParam: updateHost,
			queue:               &Mackerel{},
			mock:                &mackerelClientMock{},
			expectedGraphDef:    graphDefs,
		},
		{
			name:                "update host is error",
			expectedUpdateParam: updateHost,
			expectedError:       e,
			queue:               &Mackerel{},
			mock: &mackerelClientMock{
				returnError: e,
			},
			expectedGraphDef: nil,
		},
		{
			name:                "createGraphDef is error",
			expectedUpdateParam: updateHost,
			expectedError:       e,
			queue:               &Mackerel{},
			mock: &mackerelClientMock{
				returnErrorGraphDef: e,
			},
			expectedGraphDef: graphDefs,
		},
		{
			name: "[]collector.interface is exist",
			expectedUpdateParam: mackerel.UpdateHostParam{
				Name: "hostname",
				Interfaces: []mackerel.Interface{
					{
						Name:          "eth0",
						IPv4Addresses: []string{"192.0.2.1", "192.0.2.2"},
					},
					{
						Name:          "eth1",
						IPv4Addresses: []string{"192.0.2.3"},
					},
				},
			},
			queue:        &Mackerel{},
			returnHostID: &id,
			mock: &mackerelClientMock{
				returnHostID: "1234567890",
			},
			expectedGraphDef: graphDefs,
			interfaces: []collector.Interface{
				{
					IfName:    "eth0",
					IpAddress: []string{"192.0.2.1", "192.0.2.2"},
				},
				{
					IfName:    "eth1",
					IpAddress: []string{"192.0.2.3"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.queue.client = tc.mock
			err := tc.queue.UpdateHost(t.Context(), "0987654321", "192.0.2.2", "hostname", tc.interfaces)
			if !errors.Is(err, tc.expectedError) {
				t.Error("invalid error")
			}
			if !reflect.DeepEqual(tc.mock.updateParam, tc.expectedUpdateParam) {
				t.Error("updateParam is invalid")
			}
			if !reflect.DeepEqual(tc.mock.graphDef, tc.expectedGraphDef) {
				t.Error("CreateGraphDefs is invalid")
			}
		})
	}

}

func TestSend(t *testing.T) {
	mock := &mackerelClientMock{}
	mc := &Mackerel{
		client: mock,
	}

	if err := mc.Send(t.Context(), "0987654321", nil); err != nil {
		t.Errorf("occur error %v", err)
	}

	if mock.hostID == "" {
		t.Error("invalid need hostID")
	}

}
