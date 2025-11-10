#!/usr/bin/env bash

go tool migrate create -ext sql -dir ./migrations/ -seq $1
