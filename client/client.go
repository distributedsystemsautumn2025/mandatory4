package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	// "context"
	// "time"
	pb "mandatory4/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type node struct {
	pb.UnimplementedRicartAgrawalaServiceServer
	id    string
	clock int64
	state string

	//key: peer node ID, value: grpc client
	peers map[string]pb.RicartAgrawalaServiceClient
}

var idToPort = map[string]string{
	"1": "localhost:9000",
	"2": "localhost:9001",
	"3": "localhost:9002",
}

func main() {
	//here, we let clients define their id at instantiation in terminal
	//when starting the program, run : "go run client.go --id='ID'"
	id := flag.String("id", "", "node ID")
	flag.Parse()

	if *id == "" {
		log.Fatalf("Usage: go run client.go --id='id'")
	}

	nodeAddress, ok := idToPort[*id]
	if !ok {
		log.Fatalf("Did not find address for id=%s", *id)
	}

	lis, err := net.Listen("tcp", nodeAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Node %s listening on %s", *id, nodeAddress)

	node := &node{
		id:    *id,
		clock: 0,
		state: "released",
		peers: make(map[string]pb.RicartAgrawalaServiceClient),
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRicartAgrawalaServiceServer(grpcServer, node)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	node.connectToPeers()

	//graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// blocks code below, till signal is received
	<-signalChan
	log.Println("received termination signal, shutting down gracefully ...")
	fmt.Println("received termination signal, shutting down gracefully ...")

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

}

/*func (node *node) multicast(msg *pb.AccessRequest) {
	for peer := range n.peers {
		node.peers[peer] <- msg
	}
}*/

// here we attempt to connect to other nodes
func (node *node) connectToPeers() {
	//check that you don't attempt to connect to yourself:
	for peerID, addr := range idToPort {
		if peerID == node.id {
			continue
		}

		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to connect to peer %s at %s: %v", peerID, addr, err)
			continue
		}

		client := pb.NewRicartAgrawalaServiceClient(conn)
		node.peers[peerID] = client

		log.Printf("Node %s connected to peer %s at %s", node.id, peerID, addr)
	}
}
