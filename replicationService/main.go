package main

import (
	"bufio"
	"context"
	"log"
	"net"
	"os"
	"time"
	pb "v1/proto"

	grpc "google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedAuctionServiceServer
}

type Connection struct {
	Channel chan bool
}

var (
	connections   = make(map[string]Connection)
	highestBid    = int64(0)
	currentWinner string
	onGoing       = true
)

func (s *Server) Bid(context context.Context, amount *pb.Amount) (*pb.Ack, error) {
	if amount.Bid > highestBid {
		highestBid = amount.Bid
		currentWinner = amount.NodeId
		return &pb.Ack{Body: "SUCCESS"}, nil
	} else {
		return &pb.Ack{Body: "FAIL: BID TOO LOW!"}, nil
	}
}

func (s *Server) Result(context context.Context, void *pb.Void) (*pb.Outcome, error) {
	if onGoing {
		return &pb.Outcome{Status: "ongoing", HighestBid: highestBid}, nil
	} else {
		return &pb.Outcome{Status: "finished", HighestBid: highestBid}, nil
	}
}

func main() {
	sc := bufio.NewScanner(os.Stdin)

	log.Println("Enter the port you want to listen to: ")
	sc.Scan()
	var port = ":" + sc.Text()

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Listener failed: %v", err)
	}

	log.Println("Starting server at port " + port)

	pb.RegisterAuctionServiceServer(grpcServer, &Server{})

	timer := time.NewTimer(30 * time.Second)

	go func() {
		for {
			if highestBid > 0 {
				<-timer.C
				log.Printf("Auction is over! Winner was %v with a bid of %v", currentWinner, highestBid)
				onGoing = false
				time.Sleep(20 * time.Second)
				onGoing = true
				log.Println("New auction has started.")
				timer.Reset(30 * time.Second)
				highestBid = 0
				currentWinner = ""
			} else {
				timer.Reset(30 * time.Second)
			}
		}
	}()

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("gRPC serve failed: %v", err)
	}
}
