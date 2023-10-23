# Stanza Stream Balancer Demo

This repo contains a self-service demo of a new experimental Stanza feature that lets users fairly balance concurrent access to 
limited resources (such as streaming bandwidth, or even Lambda executions and so on).

The API is available here: [stream.proto](https://github.com/StanzaSystems/apis/blob/main/stanza/hub/v1/stream.proto).


## How to run this demo


*
1) describe repo
2) how to run - get go 1.21.0, go run cmd/demo.go
3) what it does
4) caveats - this is experimental, us-east-1 only, etc

link to grpc api
requests allocated in proportion to requested max/mins
no preemption for now
go through todos etc

ttls