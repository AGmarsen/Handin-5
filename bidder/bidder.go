package main

import (
	"bufio"
	"log"
	"os"
	"fmt"
	"strconv"
	"time"

	auc "github.com/AGmarsen/Handin-5/proto"
	"google.golang.org/grpc"
)

type peer struct {
	id       int32
	nodes  map[int32]auc.AuctionClient
}

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5010

	p := &peer{
		id:       ownPort,
		nodes:  make(map[int32]auc.AuctionClient),
	}

	//dial nodes
	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)
		var conn *grpc.ClientConn
		log.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := auc.NewAuctionClient(conn)
		p.nodes[port] = c
	}


	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		
		time.Sleep(3000 * time.Millisecond)
	}
}