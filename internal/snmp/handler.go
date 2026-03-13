package snmp

import (
	"context"
	"fmt"

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
	transport := "udp"

	if param.V2c != nil {
		return &snmpHandler{
			gosnmp.GoSNMP{
				Context:            ctx,
				Target:             param.Host,
				Port:               param.Port,
				Transport:          transport,
				Timeout:            param.Timeout,
				Retries:            param.Retry,
				ExponentialTimeout: true,
				MaxOids:            gosnmp.MaxOids,

				Version:   gosnmp.Version2c,
				Community: param.V2c.Community,
			},
		}, nil
	} else if param.V3 != nil {
		return &snmpHandler{
			gosnmp.GoSNMP{
				Context:            ctx,
				Target:             param.Host,
				Port:               param.Port,
				Transport:          transport,
				Timeout:            param.Timeout,
				Retries:            param.Retry,
				ExponentialTimeout: true,
				MaxOids:            gosnmp.MaxOids,

				Version:            gosnmp.Version3,
				SecurityModel:      gosnmp.UserSecurityModel,
				MsgFlags:           param.V3.MsgFlags(),
				SecurityParameters: param.V3.SecurityParameters(),
			},
		}, nil
	}

	return nil, fmt.Errorf("invalid params")
}

func (x *snmpHandler) Close() error {
	return x.Conn.Close()
}
