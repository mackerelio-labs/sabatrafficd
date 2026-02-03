package snmp

import (
	"context"
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type Handler interface {
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
	BulkWalk(rootOid string, walkFn gosnmp.WalkFunc) error

	Connect() error
	Close() error
}

type snmpHandler struct {
	gosnmp.GoSNMP
}

func NewHandler(ctx context.Context, param config.CollectorSNMPConfig) (Handler, error) {
	if param.V2c != nil {
		return &snmpHandler{
			gosnmp.GoSNMP{
				Context:            ctx,
				Target:             param.Host,
				Port:               param.Port,
				Transport:          "udp",
				Community:          param.V2c.Community,
				Version:            gosnmp.Version2c,
				Timeout:            time.Duration(10) * time.Second,
				Retries:            3,
				ExponentialTimeout: true,
				MaxOids:            gosnmp.MaxOids,
			},
		}, nil
	}

	return nil, fmt.Errorf("invalid params")
}

func (x *snmpHandler) Close() error {
	return x.Conn.Close()
}
