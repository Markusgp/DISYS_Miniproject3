package main

import (
	"context"
	"log"
	"net"
	"sync"
	pb "v1/proto"

	grpc "google.golang.org/grpc"
)

var (
	replica  []string                  = []string{":5001", ":5002", ":5003"}
	managers []pb.AuctionServiceClient = make([]pb.AuctionServiceClient, len(replica))
	answers  []*pb.Ack                 = make([]*pb.Ack, len(replica))
	results  []*pb.Outcome             = make([]*pb.Outcome, len(replica))
)

type Server struct {
	pb.UnimplementedAuctionServiceServer
}

type Connection struct {
	Channel chan bool
}

func callBid(context context.Context, amount *pb.Amount, wg *sync.WaitGroup, client pb.AuctionServiceClient, index int) {
	defer wg.Done()

	var ack, err = client.Bid(context, amount)

	if err != nil {
		log.Printf("Call to replica failed: %v : Disconnecting replica", err)
		managers = removeDeadReplica(managers, index)
		return
	}

	answers[index] = ack
}

func (s *Server) Bid(context context.Context, amount *pb.Amount) (*pb.Ack, error) {
	wg := new(sync.WaitGroup)
	wg.Add(len(managers))

	for index, client := range managers {
		go callBid(context, amount, wg, client, index)
	}

	wg.Wait()

	return answers[0], nil
}

func callResult(context context.Context, void *pb.Void, wg *sync.WaitGroup, client pb.AuctionServiceClient, index int) {
	defer wg.Done()

	var result, err = client.Result(context, void)

	if err != nil {
		log.Printf("Call to replica failed: %v : Disconnecting replica", err)
		managers = removeDeadReplica(managers, index)
		return
	}

	results[index] = result
}

func (s *Server) Result(context context.Context, void *pb.Void) (*pb.Outcome, error) {
	wg := new(sync.WaitGroup)
	wg.Add(len(managers))

	for index, client := range managers {
		go callResult(context, void, wg, client, index)
	}

	wg.Wait()

	return results[0], nil
}

func removeDeadReplica(s []pb.AuctionServiceClient, i int) []pb.AuctionServiceClient {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func serve() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed broadcasting local network: %v", err)
	}

	log.Println("Listening on port :8080")

	pb.RegisterAuctionServiceServer(grpcServer, &Server{})

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("gRPC serve failed: %v", err)
	}
}

func main() {
	for index, port := range replica {
		connection, err := grpc.Dial(port, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Failed connecting to port: %v. ErrorCode: %v", port, err)
		}
		defer connection.Close()

		client := pb.NewAuctionServiceClient(connection)

		managers[index] = client

		log.Printf("Connection to port: %v successful", port)
	}

	serve()
}
