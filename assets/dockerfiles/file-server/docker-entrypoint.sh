#!/bin/sh

wget "host.docker.internal:5000/health/$(cat /etc/hostname)" -q -O - > /dev/null 2>&1

$("$@")
