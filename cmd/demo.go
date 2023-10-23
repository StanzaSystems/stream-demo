package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	hubv1 "github.com/StanzaSystems/stream-demo/gen/go/stanza/hub/v1"
	pb "github.com/StanzaSystems/stream-demo/gen/go/stanza/hub/v1"
	"go.openly.dev/pointy"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	hub          string
	verbose      bool
	hub_insecure bool
)

const apikey = "sb-demo-apikey"

// Tests against a demo Guard with a overall limit of 50 and a per-customer-id limit of 15.
func main() {
	flag.StringVar(&hub, "hub", "hub.dev.getstanza.dev:9020", "The hub address host:port to issue queries against.")
	//flag.StringVar(&hub, "hub", "hub.demo.getstanza.io:9020", "The hub address host:port to issue queries against.") // TODO: when deployed to demo.
	flag.BoolVar(&hub_insecure, "hub_insecure", false, "Skip Hub TLS validation (for local development only).")
	flag.BoolVar(&verbose, "verbose", false, "Print out details on every success/failure.")
	flag.Parse()

	config := &tls.Config{} // use default system CA
	creds := credentials.NewTLS(config)
	if hub_insecure {
		creds = insecure.NewCredentials()
	}
	conn, err := grpc.Dial(hub, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewStreamBalancerServiceClient(conn)

	fmt.Printf(
		`
Welcome to the Stanza Stream Balancer demo. This feature is currently in experimental/evaluation mode (see README.md in this repo for more information).
This demo runs a series of requests against Stanza's control plane to demonstrate how it can balance stream operations.
The demo config in use is as follows:
 * There is an overall limit of 50 units of capacity in the system
 * Each distinct customer is allocated up to 15 units of capacity
 * If the overall system is at capacity, then capacity is shared fairly

`)

	// First, forget any existing streams created by previous runs of this demo
	_, err = sendReq(nil, []string{"a-new-stream", "another-new-stream", "yet-another-new-stream", "cust-3-stream", "cust-4-stream", "cust-1-streamp0", "cust-1-streamp1", "cust-1-streamp2"}, client)
	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}

	// Request one stream - should get full per-customer 15 weight
	reqs := []*pb.StreamRequest{
		{
			StreamId:  "a-new-stream",
			MinWeight: 1,
			MaxWeight: 20,
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer1",
				}},
		},
	}

	fmt.Printf("Next, we request one stream. We expect Stanza to permit 15 units of capacity for this stream, using the customer's allocated capacity")
	res, err := sendReq(reqs, nil, client)

	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}

	// try allocate another stream to same customer - will split the 15 quota between them
	reqs = []*pb.StreamRequest{
		{
			StreamId:  "another-new-stream",
			MinWeight: 1,
			MaxWeight: 20,
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer1",
				}},
		},
	}
	fmt.Printf("Now, we request a second stream for the same customer. It should split the customer's quota between the two streams, so the existing stream is downsized: \n%+v\n", reqs)
	res, err = sendReq(reqs, nil, client)

	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: %+v\n\n", res)

	// try allocate another stream to a new customer - will give 15 again
	reqs = []*pb.StreamRequest{
		{
			StreamId:  "yet-another-new-stream",
			MinWeight: 1,
			MaxWeight: 20,
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer2",
				}},
		},
	}
	fmt.Printf("Attempt to allocate another stream for a different customer. It should be allocated 15 units of capacity: \n%+v\n", reqs)
	res, err = sendReq(reqs, nil, client)

	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: %+v\n\n", res)

	// add more streams for 2 further customers and hit the overall limit of 50 - every customer limited to 12.5 total
	reqs = []*pb.StreamRequest{
		{
			StreamId:  "cust-3-stream",
			MinWeight: 1,
			MaxWeight: 20,
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer3",
				}},
		},
		{
			StreamId:  "cust-4-stream",
			MinWeight: 1,
			MaxWeight: 20,
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer4",
				}},
		},
	}

	fmt.Printf("Two more customers request streams. The system now has more requests than capacity, so some existing streams are downsized: \n%+v\n", reqs)
	res, err = sendReq(reqs, nil, client)

	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: %+v\n\n", res)

	// remove some streams and see some remaining streams upsized
	res, err = sendReq(nil, []string{"another-new-stream", "cust-3-stream"}, client)
	fmt.Printf("Removing some existing streams leaves more capacity for the rest, so they are upsized - removing \"another-new-stream\" and \"cust-3-stream\"\n")
	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: %+v\n\n", res)

	// overload the quota for customer1 requests - it will prioritise the highest priority requests
	reqs = []*pb.StreamRequest{
		{
			StreamId:      "cust-1-streamp0",
			MinWeight:     5,
			MaxWeight:     20,
			PriorityBoost: pointy.Int32(5),
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer1",
				}},
		},
		{
			StreamId:      "cust-1-streamp1",
			MinWeight:     5,
			MaxWeight:     20,
			PriorityBoost: pointy.Int32(4),
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer1",
				}},
		},
		// won't get allocated because we can't allocate its minweight in addition to the current p5 stream
		// a-new-stream plus the above two higher priority streams
		// we don't currently preempt running streams, although we could add that as an option
		{
			StreamId:      "cust-1-streamp2",
			MinWeight:     5,
			MaxWeight:     20,
			PriorityBoost: pointy.Int32(3),
			Tags: []*pb.Tag{
				{
					Key:   "customer_id",
					Value: "customer1",
				}},
		},
	}

	fmt.Printf("Finally, we request too many streams for customer1 - we cannot serve the minimum requested stream size for each stream. Stanza serves the two higher priority streams in this request, and continues to serve the existing stream a-new-stream:\n %+v \n", reqs)

	res, err = sendReq(reqs, nil, client)
	if err != nil {
		fmt.Printf("Got error from stanza: %+v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %+v\n\n", res)
}

func sendReq(reqs []*pb.StreamRequest, rms []string, client pb.StreamBalancerServiceClient) (*pb.UpdateStreamsResponse, error) {
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "X-Stanza-Key", apikey)

	req := pb.UpdateStreamsRequest{
		GuardName:   "Stream Balancer Quota",
		Environment: "sb_quota",
		Requests:    reqs,
		Ended:       rms,
	}
	res, err := client.UpdateStreams(ctx, &req)
	printResult(res)

	return res, err
}

func eq(b, a float32) bool {
	diff := math.Abs(float64(b) - float64(a))
	return diff < 0.01
}

func printResult(res *hubv1.UpdateStreamsResponse) {
	fmt.Printf("Result: \n%+v\n\n", prototext.Format(res))

}
