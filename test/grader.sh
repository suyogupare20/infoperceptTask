#!/usr/bin/env bash
set -euo pipefail

# basic smoke test: start server, upload+download small file
go run ../main.go &
PID=$!
sleep 1

echo "hello" > /tmp/hello.txt
curl -s -X PUT --data-binary @/tmp/hello.txt http://localhost:8080/myb/hello.txt > /tmp/etag.json
curl -s http://localhost:8080/myb/hello.txt -o /tmp/out.txt
diff /tmp/hello.txt /tmp/out.txt && echo "smoke OK"

kill -SIGINT "$PID"