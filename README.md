# Stanza Stream Balancer Demo

This repo contains a self-service demo of a new experimental Stanza feature that lets users fairly balance concurrent access to 
limited resources (such as streaming bandwidth, or even Lambda executions and so on).

The API is available here: [stream.proto](https://github.com/StanzaSystems/apis/blob/main/stanza/hub/v1/stream.proto).

The API allows you to request allocations for new Streams, and to indicate that existing Streams have completed.
Each StreamRequest has a minimum and maximum desired quota. If the minimum quota cannot be met, then zero will be returned to indicate that the stream cannot be allocated.
When quota is constrained (i.e. )

If minimum quotas cannot be served, then Stanza will serve:
 * existing running streams
 * then the highest priority newly requested streams

Request weights are allocated in proportion to minimum weights requested.

## How to run this demo

Install Go 1.21.0 or greater (see [go site](https://go.dev/doc/install)).

Check out this repo.

Run the code:
```
go run cmd/demo.go
```

The demo will exercise the Stream Balancer functionality and print the requests and responses going to/from the Stanza control-plane 
with some commentary.

## Caveats and TODOs

This feature is new and experimental. 
 * It runs only in us-east-1 for now, so latency will be higher to further-flung regions. 
 * There are some scenarios where it won't binpack quite perfectly to use all available quota
 * There is currently no mechanism to preempt lower-priority running streams in favour of higher-priority new streams
 * When constrained, it doesn't currently allocate proportionally more weight to higher-priority streams, which would probably be desirable
 * Best-effort burst is not yet supported (as it is in regular per-qps Stanza ratelimiting - see [our other backend demo](https://github.com/StanzaSystems/stanza-api-demo))
 * If streams aren't marked as terminated by the user, quota might 'leak' away. Mechanisms to prevent this should be added, including a full state resync and/or a TTL mechanism as a fallback.