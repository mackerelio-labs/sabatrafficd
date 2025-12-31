#!/bin/sh

set -e

if [ ! -f "/etc/sabatrafficd/sabatrafficd.yaml" ]; then
    cp /etc/sabatrafficd/sabatrafficd.yaml.sample /etc/sabatrafficd/sabatrafficd.yaml
fi

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
fi
