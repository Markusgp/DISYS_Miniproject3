package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	pb "v1/proto"

	uuid "github.com/nu7hatch/gouuid"
	"google.golang.org/grpc"
)

var (
	nodeID string
	sc     bufio.Scanner
)

func generateNode() {
	Genereateduuid, err := uuid.NewV4()

	if err != nil {
		log.Fatalf("Failed when creating UUID: %v", err)
	}

	nodeID = Genereateduuid.String()
}

func dialGrpc() (connection *grpc.ClientConn, returnClient pb.AuctionServiceClient, returnCtx context.Context) {
	log.Println("Enter the IP-address you wish to connect to: ")
	sc.Scan()
	var address = ":" + sc.Text()

	connection, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed connecting to server: %v", err)
	}

	client := pb.NewAuctionServiceClient(connection)
	ctx := context.Background()

	log.Println("Connection successful and open for requests.")

	return connection, client, ctx
}

func main() {
	sc = *bufio.NewScanner(os.Stdin)

	generateNode()

	connection, client, ctx := dialGrpc()

	defer connection.Close()

	for {
		sc.Scan()
		var inp = sc.Text()
		inpLC := strings.ToLower(inp)
		s := strings.Split(inpLC, " ")

		Outcome, err := client.Result(ctx, &pb.Void{})
		if err != nil {
			log.Fatalln("Result query failed.")
		}

		if Outcome.GetStatus() == "ongoing" && s[0] == "bid" && len(s) == 2 {
			bidAmount, err1 := strconv.ParseInt(s[1], 10, 64)
			if err1 != nil {
				log.Fatalf("Input bid not valid: %v", err1)
			}
			Ack, err2 := client.Bid(ctx, &pb.Amount{Bid: bidAmount, NodeId: nodeID})
			if err2 != nil {
				log.Fatalln("Bid failed.")
			}

			log.Println(Ack.GetBody())
		} else if Outcome.GetStatus() == "ongoing" && s[0] == "result" && len(s) == 1 {
			if Outcome.GetHighestBid() > 0 {
				log.Printf("Auction is ongoing. The highest bid is: %v", Outcome.GetHighestBid())
			} else {
				log.Printf("Auction is ongoing. There has been no bids")
			}
		} else if Outcome.GetStatus() == "finished" && s[0] == "result" && len(s) == 1 {
			log.Printf("The auction is over. The highest bid was: %v", Outcome.GetHighestBid())
		} else if Outcome.GetStatus() == "finished" && s[0] == "bid" && len(s) == 2 {
			log.Printf("The auction is over. No more bids.")
		}
	}
}
