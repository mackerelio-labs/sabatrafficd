package config

import (
	"fmt"
	"math"

	"github.com/docker/go-units"
	"gopkg.in/yaml.v3"
)

type Size struct {
	size int64
}

var _ yaml.Unmarshaler = &Size{}

func (s *Size) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	if v == "-1" { // -1 なら無制限とする
		*s = Size{size: math.MaxInt64}
		return nil
	}

	size, err := units.FromHumanSize(v)
	if err != nil {
		return fmt.Errorf("%s on line %d", err.Error(), value.Line)
	}
	*s = Size{size: size}
	return nil
}

func (s *Size) Size() int64 {
	return s.size
}
