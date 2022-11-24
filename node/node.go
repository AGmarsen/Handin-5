package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	auc "github.com/AGmarsen/Handin-5/proto"
	"google.golang.org/grpc"
)

type Node struct {
	auc.UnimplementedAuctionServer

	ownPort       int32
	highestBid    int32
	highestBidder string
	auctionStatus string
	countDown     int32
	started       bool
	nodes         map[int32]auc.AuctionClient
	inbox         map[int32]bool
	leaderNode    int32
	lastBeat      time.Time

	mutex sync.Mutex
}

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	node := &Node{
		ownPort:       int32(arg1) + 5000,
		highestBid:    0,
		highestBidder: "",
		auctionStatus: "open",
		countDown:     60,
		started:       false,
		nodes:         make(map[int32]auc.AuctionClient),
		inbox:         make(map[int32]bool),
		leaderNode:    5000,
	}

	// Create listener tcp on given port or default port 5400
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", node.ownPort))
	if err != nil {
		log.Printf("Failed to listen on port %d: %v", node.ownPort, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}
	log.Printf("Now listening on port %d", node.ownPort)

	grpcServer := grpc.NewServer()

	auc.RegisterAuctionServer(grpcServer, node) //Registers the server to the gRPC server.

	log.Printf("Server: Listening at %v", list.Addr())

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == node.ownPort {
			continue
		}

		var conn *grpc.ClientConn
		log.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		n := auc.NewAuctionClient(conn)
		node.nodes[port] = n
		node.inbox[port] = false
	}

	go node.CountDown()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() { //keep it alive
		if scanner.Text() == "end" {
			break
		}
	}
}

func (n *Node) Bid(ctx context.Context, bid *auc.Bid) (*auc.Ack, error) {
	n.started = true
	n.mutex.Lock()
	defer n.mutex.Unlock()
	outcome := ""
	if n.countDown == 0 {
		outcome = "Bid not accepted, auction has ended. Type \"status\" to see the result."
	} else if bid.Amount > n.highestBid {
		n.highestBid = bid.Amount
		n.highestBidder = bid.Name
		outcome = fmt.Sprintf("Bid accepted. Highest bid is now %d", bid.Amount)
		n.SyncronizeBackups()
	} else {
		outcome = fmt.Sprintf("Bid not accepted, current highest bid is %d", n.highestBid)
	}
	log.Println(outcome)
	return &auc.Ack{Outcome: outcome}, nil
}

func (n *Node) GetBiddingStatus(ctx context.Context, e *auc.Empty) (*auc.Result, error) {
	n.started = true
	n.UpdateBittingStatus()
	log.Println(n.auctionStatus)
	return &auc.Result{Status: n.auctionStatus}, nil
}

func (n *Node) UpdateBittingStatus() {
	if n.countDown == 0 && n.highestBid == 0 {
		n.auctionStatus = "Auction has ended with 0 bids."
	} else if n.countDown == 0 {
		n.auctionStatus = fmt.Sprintf("Auction has ended. %s had the highest bid of %d", n.highestBidder, n.highestBid)
	} else if n.highestBid == 0 {
		n.auctionStatus = fmt.Sprintf("Auction is open. No one has bidded yet. Ends in %d seconds", n.countDown)
	} else {
		n.auctionStatus = fmt.Sprintf("Auction is open. %s is in the lead with a bid of %d. Ends in %d seconds", n.highestBidder, n.highestBid, n.countDown)
	}
}

func (n *Node) SyncronizeBackups() {
	go func() {
		for i, node := range n.nodes {
			nd := node
			id := i
			_, err := nd.Syncronize(context.Background(), &auc.Sync{HighestBidder: n.highestBidder, HighestBid: n.highestBid, CountDown: n.countDown})
			if err == nil {
				n.inbox[id] = true
			}
		}
	}()
	n.waitForResponse()
}

func (n *Node) Syncronize(ctx context.Context, sync *auc.Sync) (*auc.Empty, error) {
	n.highestBidder = sync.HighestBidder
	n.highestBid = sync.HighestBid
	n.countDown = sync.CountDown
	log.Printf("Highest bidder: %s, Highest bid: %d, countDown: %d", n.highestBidder, n.highestBid, n.countDown)
	return &auc.Empty{}, nil
}

func (n *Node) HeartBeat(ctx context.Context, beat *auc.Beat) (*auc.Empty, error) {
	if n.lastBeat.IsZero() {
		n.lastBeat = time.Now()
		go n.DetectCrash() //start looking for craches after first heart beat
	}
	n.countDown = beat.CountDown
	n.lastBeat = time.Now()
	return &auc.Empty{}, nil
}

func (n *Node) GiveHeartBeat() {
	for _, node := range n.nodes {
		nd := node
		go func() {
			nd.HeartBeat(context.Background(), &auc.Beat{Id: n.ownPort, CountDown: n.countDown})
		}()
	}

}

func (n *Node) CountDown() {
	for {
		if n.started {
			break
		} else if n.IsLeader() { //only the leader node gives a heat beat to all other nodes
			time.Sleep(time.Second)
			n.GiveHeartBeat()
		}
	}
	for n.countDown > 0 { //only the leader goes here when the auction starts or when a backup node takes over
		time.Sleep(time.Second)
		n.countDown -= 1
		n.GiveHeartBeat()
	}
	log.Println("Auction closed.")
	for {
		time.Sleep(time.Second)
		n.GiveHeartBeat()
	}
}

func (n *Node) DetectCrash() {
	for !n.IsLeader() {
		if time.Now().After(n.lastBeat.Add(3000 * time.Millisecond)) { //if no heartbeat for x seconds
			log.Printf("Port %d seems dead.", n.leaderNode)
			delete(n.nodes, n.leaderNode)
			delete(n.inbox, n.leaderNode)
			n.leaderNode = n.NewLeader()
			log.Printf("New leader selected: %d", n.leaderNode)
			n.started = n.IsLeader() && (n.countDown < 60 || n.highestBid > 0) //starts countDown and heatbeat if Leader and auction started
			time.Sleep(time.Second)
			n.lastBeat = time.Now() //give the next node some time to give a beat
		}
	}
}

func (n *Node) IsLeader() bool { //am i the leader
	return n.leaderNode == n.ownPort
}

func (n *Node) NewLeader() int32 {
	leader := n.ownPort
	for id := range n.nodes { //find smallest id. That is the new leader
		if leader > id {
			leader = id
		}
	}
	return leader
}

func (n *Node) waitForResponse() {
	time.Sleep(1000 * time.Millisecond)
	var killList []int32
	for id, alive := range n.inbox {
		if !alive {
			log.Printf("Port %d seems dead.", id)
			killList = append(killList, id)
		} else {
			n.inbox[id] = false
		}
	}
	for _, id := range killList {
		delete(n.nodes, id)
		delete(n.inbox, id)
	}
}
