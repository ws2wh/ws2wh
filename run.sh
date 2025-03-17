#!/usr/bin/env bash

set -eu

go run cmd/ws2wh/main.go -b $1 -metrics-enabled true
