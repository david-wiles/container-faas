#!/bin/sh

nginx &

/go/bin/paas-server --addr localhost:1024 --nginx

