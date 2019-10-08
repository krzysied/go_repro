#!/bin/bash

go build ./...

./http_memory --duration=10m --type=etcd --port=38088 --certPath=./certs/localhost.crt --keyPath=./certs/localhost.key &> etcd.log &
./http_memory --duration=10m --type=apiserver --port=38080 --certPath=./certs/localhost.crt --keyPath=./certs/localhost.key --sleepDuration=5m --endpoint=https://localhost:38088 --qps=2000 --poolSize=10 --threshold=100ms &> server.log &
./http_memory --duration=10m --type=client --certPath=./certs/localhost.crt --endpoint=https://localhost:38080 --qps=400 --poolSize=5000 &> client.log &

wait
