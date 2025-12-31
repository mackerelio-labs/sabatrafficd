//go:build linux

package sdnotify

import (
	"fmt"

	"github.com/coreos/go-systemd/v22/daemon"
	"golang.org/x/sys/unix"
)

func SendReloading() string {
	var ts unix.Timespec
	unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts) // nolint:errcheck
	return fmt.Sprintf("%s\nMONOTONIC_USEC=%d", daemon.SdNotifyReloading, ts.Nano()/1000)
}
