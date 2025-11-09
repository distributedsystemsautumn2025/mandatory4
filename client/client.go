package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// "context"
	pb "mandatory4/grpc"
	"time"
)

type node struct {
	pb.UnimplementedRicartAgrawalaServiceServer
	peers    map[string]pb.RicartAgrawalaServiceClient
	channels map[string]chan *pb.AccessRequest
	clock    int64
	state    string
}

var peerPorts = []string{
	"localhost:9001",
	"localhost:9002",
	"localhost:9003",
}

func (n *node) multicast(msg *pb.AccessRequest) {
	for channel := range n.channels {
		n.channels[channel] <- msg
	}
}

func (n *node) connectToPeer(addr string) {
	for {
		conn, err := grpc.NewClient(
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			log.Println("connected to", addr)
			fmt.Println("connected to", addr)
			n.peers[addr] = pb.NewRicartAgrawalaServiceClient(conn)

			return
		}

		log.Println("retrying peer:", addr)
		fmt.Println("retrying peer:", addr)
		time.Sleep(2 * time.Second)
	}
}

func main() {
	//setup logFile
	logFile, err := os.OpenFile("../RicartAgrawala.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// handle server address
	port := "9000"
	if len(os.Args) > 1 { // if the client specifies a port
		port = os.Args[1]
	}

	// server listening
	lis, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		fmt.Printf("failed to listen: %v", err)
	}
	log.Printf("The server is listening at %v", lis.Addr())
	fmt.Printf("The server is listening at %v", lis.Addr())

	grpcServer := grpc.NewServer()
	n := &node{
		channels: make(map[string]chan *pb.AccessRequest),
	}

	pb.RegisterRicartAgrawalaServiceServer(grpcServer, n)

	// connecting to peers
	for _, port := range peerPorts {
		n.connectToPeer(port)
	}

	// accessRequest := &pb.AccessRequest{
	// 	Sender:      port,
	// 	LogicalTime: 1,
	// }
	// n.multicast(accessRequest)
	// fmt.Print("send accessrequest")

	//graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// blocks code below, till signal is received
	<-signalChan
	log.Println("received termination signal, shutting down gracefully ...")
	fmt.Println("received termination signal, shutting down gracefully ...")

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

}
