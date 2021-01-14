#!/bin/sh

node /bin/health.js > /dev/null &

$("$@")
