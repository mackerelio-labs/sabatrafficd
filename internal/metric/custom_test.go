package metric

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mackerelio/mackerel-client-go"
)

func TestConvertCustom(t *testing.T) {
	c := &Custom{
		mapping: map[string]string{
			"foo": "1.2.3.4",
			"bar": "2.3.4.5",
		},
	}

	input := map[string]float64{
		"1.2.3.4": 1.2345,
		"3.4.5.6": 0.1234,
	}

	actual := c.Convert(input)

	expected := []*mackerel.MetricValue{
		{
			Name:  "foo",
			Time:  time.Now().Unix(),
			Value: 1.2345,
		},
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("value is mismatch (-actual +expected):%s", diff)
	}
}
