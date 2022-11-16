package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	auc "github.com/AGmarsen/Handin-5/proto"
	"google.golang.org/grpc"
)

type Node struct {
	auc.UnimplementedAuctionServer

	highestBid    int
	higestBidder  int
	auctionStatus string

	mutex sync.Mutex
}

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5000

	// Create listener tcp on given port or default port 5400
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", ownPort))
	if err != nil {
		log.Printf("Failed to listen on port %d: %v", ownPort, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}
	log.Printf("Now listening on port %d", ownPort)

	grpcServer := grpc.NewServer()

	// makes a new server instance using the name and port from the flags.
	node := &Node{
		highestBid:    0,
		higestBidder:  -1,
		auctionStatus: "open",
	}

	auc.RegisterAuctionServer(grpcServer, node) //Registers the server to the gRPC server.

	log.Printf("Server: Listening at %v", list.Addr())

	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}
