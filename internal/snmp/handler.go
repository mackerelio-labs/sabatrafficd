package snmp

import (
	"context"

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

func NewHandler(param config.CollectorSNMPConfig) *snmpHandler {
	transport := "udp"

	if param.V2c != nil {
		return &snmpHandler{
			gosnmp.GoSNMP{
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
		}
	} else if param.V3 != nil {
		return &snmpHandler{
			gosnmp.GoSNMP{
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
		}
	}

	panic("unreachable")
}

func (x *snmpHandler) Close() error {
	return x.Conn.Close()
}

func (x *snmpHandler) SetContext(ctx context.Context) {
	x.Context = ctx
}
