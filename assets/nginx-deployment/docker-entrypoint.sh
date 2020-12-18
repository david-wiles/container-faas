#!/bin/sh

nginx &

/go/bin/faas-server --addr localhost:1024 --nginx

