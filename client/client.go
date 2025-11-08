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
)

type node struct {
	pb.UnimplementedRicartAgrawalaServiceServer
	ID    string
	peers map[string]chan *pb.AccessRequest
	clock int64
	state string
}

func main() {
	//here, we let clients define the port they will be listening at on instantiation in terminal
	port := flag.String("port", "", "listen on this port")
	id := flag.String("id", "", "node ID")
	flag.Parse()

	if *port == "" || *id == "" {
		log.Fatal("Usage: go run main.go --port='port' --id='id'")
	}

	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", *port, err)
	}

	node := &node{
		ID:    *id,
		peers: make(map[string]chan *pb.AccessRequest),
		clock: 0,
		state: "released",
	}

	log.Printf("Node %s listening on port %s", *id, *port)
	grpcServer := grpc.NewServer()
	pb.RegisterRicartAgrawalaServiceServer(grpcServer, node)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

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

func (n *node) multicast(msg *pb.AccessRequest) {
	for peer := range n.peers {
		n.peers[peer] <- msg
	}
}
