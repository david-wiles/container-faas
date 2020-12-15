#!/bin/sh

# TODO make sure the server's url is configurable here
wget "host.docker.internal:5000/health/$(cat /etc/hostname)" -q -O - > /dev/null 2>&1

node /home/app/index.js
