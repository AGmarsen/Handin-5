package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	auc "github.com/AGmarsen/Handin-5/proto"
	"google.golang.org/grpc"
)

const BID = "bid"
const STATUS = "status"

var name string
var node auc.AuctionClient
var inbox chan string
var kill chan bool
var result chan string
var mutex sync.Mutex
var port int32

func main() {
	name = os.Args[1]
	inbox = make(chan string)
	kill = make(chan bool, 1)
	result = make(chan string)

	//dial nodes
	port = int32(5000)
	var conn *grpc.ClientConn
	log.Printf("Trying to dial: %v\n", port)
	conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Could not connect: %s", err)
	}
	defer conn.Close()
	node = auc.NewAuctionClient(conn)

	fmt.Println("Welcome to the auction!")
	fmt.Printf("Type \"%s {amount here}\" or result to place a bid.\n", BID)
	fmt.Printf("Type \"%s\" to see the current state of the auction\n", STATUS)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		readInput(scanner.Text())
	}
}

func readInput(s string) {
	if strings.HasPrefix(s, BID) {
		amount, err := strconv.ParseInt(strings.Split(s, " ")[1], 10, 32)
		if err != nil {
			log.Printf("Failed to convert amount to an integer. Try with something else.")
		} else {
			bid(int32(amount))
		}
	} else if strings.HasPrefix(s, STATUS) {
		bidStatus()
	}
}

func bid(amount int32) {
	for i := 0; i < 2; i++ { //try twice in case of a crash (assume max one crash)
		go func() {
			ack, err := node.Bid(context.Background(), &auc.Bid{Name: name, Amount: amount})
			if err == nil {
				inbox <- ack.Outcome
			}
		}()
		go waitForResponse()

		res := <-result
		if res != "" {
			log.Println(res)
			return
		}
	}
	log.Println("Failed to receive a response from any nodes.")

}

func bidStatus() {
	for i := 0; i < 2; i++ { //try twice in case of a crash (assume max one crash)
		go func() {
			ack, err := node.GetBiddingStatus(context.Background(), &auc.Empty{})
			if err == nil {
				inbox <- ack.Status
			}
		}()
		go waitForResponse()

		res := <-result
		if res != "" {
			log.Println(res)
			return
		}
	}
	log.Println("Failed to receive a response from any nodes.")
}

func waitForResponse() { //assume all messages arrive whitin the given time. If they assume crashed

	end := time.Now().Add(3000 * time.Millisecond)

	for {
		select { //waits for a response in the inbox or a kill order
		case resp := <-inbox:
			mutex.Lock()
			defer mutex.Unlock()
			result <- resp
			return
		case <-kill:
			log.Println("Failed to receive a response from leader node. Trying to redirect...")
			port++
			var conn *grpc.ClientConn
			conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock()) //dial the next node
			if err != nil {
				log.Printf("%v", err)
			}
			node = auc.NewAuctionClient(conn)
			log.Println("Redirect successful.")
			result <- ""
			return
		default: //if too much time passes. Send kill order 66
			if time.Now().After(end) {
				kill <- true
			}
		}
	}
}
