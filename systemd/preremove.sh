#!/bin/sh

set -e

if [ -d /run/systemd/system ]; then
    systemctl disable --now sabatrafficd
fi
