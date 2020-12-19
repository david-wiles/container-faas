#!/bin/sh

wget "host.docker.internal:8080/health/$(cat /etc/hostname)" -q -O - > /dev/null 2>&1

$("$@")
