package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	auc "github.com/AGmarsen/Handin-5/proto"
	"google.golang.org/grpc"
)

var name string
var nodes map[int32]auc.AuctionClient

func main() {

	name = os.Args[1]
	nodes = make(map[int32]auc.AuctionClient)

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
		nodes[port] = c
	}

	fmt.Println("Wlcome to the auction!")
	fmt.Println("Type \"bid {amount here}\" or result to place a bid.")
	fmt.Println("Type \"status\" to see the current state of the auction")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		readInput(scanner.Text())
	}
}

func readInput(s string) {
	if strings.HasPrefix(s, "bid") {
		amount, err := strconv.ParseInt(strings.Split(s, " ")[1], 10, 32)
		if err != nil {
			log.Printf("%v", err)
		} else {
			bid(int(amount))
		}
	} else if strings.HasPrefix(s, "result") {

	}
}

func bid(amount int) {
	for _, node := range nodes {
		node.Bid(context.Background(), &auc.Bid{Name: name, Amount: int32(amount)})
	}
}
